package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Parser struct {
	cycleRegex     *regexp.Regexp
	resourceRegex  *regexp.Regexp
	moduleRegex    *regexp.Regexp
	instanceRegex  *regexp.Regexp
	actionRegex    *regexp.Regexp
	deposedRegex   *regexp.Regexp
}

func NewParser() *Parser {
	return &Parser{
		cycleRegex:     regexp.MustCompile(`(?s)Error:\s*Cycle:\s*(.+)`),
		resourceRegex:  regexp.MustCompile(`([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_-]+)`),
		moduleRegex:    regexp.MustCompile(`^((?:module\.[a-zA-Z0-9_-]+\.)*)`),
		instanceRegex:  regexp.MustCompile(`\[([^\]]+)\]`),
		actionRegex:    regexp.MustCompile(`\s*\((expand|destroy|close|destroy\s+deposed\s+[a-f0-9]+)\)`),
		deposedRegex:   regexp.MustCompile(`destroy\s+deposed\s+([a-f0-9]+)`),
	}
}

func (p *Parser) ParseError(errorText string) (*TfCycle, error) {
	cycle := &TfCycle{
		RawError: errorText,
		Nodes:    make([]*CycleNode, 0),
	}

	matches := p.cycleRegex.FindStringSubmatch(errorText)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not extract cycle from error message")
	}

	cycleText := matches[1]
	resourceStrings := p.splitResources(cycleText)

	for _, resourceStr := range resourceStrings {
		node, err := p.parseResource(strings.TrimSpace(resourceStr))
		if err != nil {
			fmt.Printf("Warning: failed to parse resource '%s': %v\n", resourceStr, err)
			continue
		}
		cycle.Nodes = append(cycle.Nodes, node)
	}

	if len(cycle.Nodes) == 0 {
		return nil, fmt.Errorf("no valid resources found in cycle")
	}

	return cycle, nil
}

func (p *Parser) splitResources(cycleText string) []string {
	cycleText = strings.ReplaceAll(cycleText, "\n", " ")
	cycleText = strings.ReplaceAll(cycleText, "\t", " ")
	
	var resources []string
	var current strings.Builder
	inBrackets := 0
	inParens := 0
	
	for _, char := range cycleText {
		switch char {
		case '[':
			inBrackets++
			current.WriteRune(char)
		case ']':
			inBrackets--
			current.WriteRune(char)
		case '(':
			inParens++
			current.WriteRune(char)
		case ')':
			inParens--
			current.WriteRune(char)
		case ',':
			if inBrackets == 0 && inParens == 0 {
				if current.Len() > 0 {
					resources = append(resources, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}
	
	if current.Len() > 0 {
		resources = append(resources, current.String())
	}
	
	var filtered []string
	for _, resource := range resources {
		trimmed := strings.TrimSpace(resource)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	
	return filtered
}

func (p *Parser) parseResource(resourceStr string) (*CycleNode, error) {
	node := &CycleNode{
		RawString:   resourceStr,
		Action:      ActionNormal,
		Annotations: make(map[string]string),
	}

	cleanStr := resourceStr
	
	actionMatches := p.actionRegex.FindStringSubmatch(resourceStr)
	if len(actionMatches) >= 2 {
		actionStr := strings.TrimSpace(actionMatches[1])
		cleanStr = p.actionRegex.ReplaceAllString(cleanStr, "")
		
		switch {
		case actionStr == "expand":
			node.Action = ActionExpand
		case actionStr == "close":
			node.Action = ActionClose
		case actionStr == "destroy":
			node.Action = ActionDestroy
		case strings.HasPrefix(actionStr, "destroy deposed"):
			node.Action = ActionDestroyDeposed
			deposedMatches := p.deposedRegex.FindStringSubmatch(actionStr)
			if len(deposedMatches) >= 2 {
				node.Annotations["deposed_id"] = deposedMatches[1]
			}
		}
	}

	instanceMatches := p.instanceRegex.FindStringSubmatch(cleanStr)
	if len(instanceMatches) >= 2 {
		node.InstanceKey = strings.Trim(instanceMatches[1], `"`)
		cleanStr = p.instanceRegex.ReplaceAllString(cleanStr, "")
	}

	moduleMatches := p.moduleRegex.FindStringSubmatch(cleanStr)
	if len(moduleMatches) >= 2 && moduleMatches[1] != "" {
		modulePath := strings.TrimSuffix(moduleMatches[1], ".")
		if modulePath != "" {
			node.ModulePath = strings.Split(modulePath, ".")
		}
		cleanStr = strings.TrimPrefix(cleanStr, moduleMatches[1])
	}

	resourceMatches := p.resourceRegex.FindStringSubmatch(cleanStr)
	if len(resourceMatches) < 3 {
		return nil, fmt.Errorf("could not parse resource type and name from '%s'", cleanStr)
	}
	
	node.ResourceType = resourceMatches[1]
	node.ResourceName = resourceMatches[2]

	return node, nil
}