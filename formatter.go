package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type OutputFormatter struct {
	analyzer *CycleAnalyzer
	verbose  bool
}

func NewOutputFormatter(analyzer *CycleAnalyzer, verbose bool) *OutputFormatter {
	return &OutputFormatter{
		analyzer: analyzer,
		verbose:  verbose,
	}
}

func (of *OutputFormatter) FormatAnalysis() string {
	var output strings.Builder
	
	output.WriteString("🔄 TERRAFORM CYCLE DETECTED\n\n")
	
	if of.verbose {
		of.writeVerboseInfo(&output)
	}
	
	cycles := of.analyzer.FindMinimalCycles()
	
	if len(cycles) == 0 {
		output.WriteString("❌ No cycles found in the provided resources\n")
		return output.String()
	}
	
	of.writeMinimalCycles(&output, cycles)
	of.writeSuggestions(&output, cycles)
	
	if of.verbose {
		of.writeAllResources(&output)
	}
	
	return output.String()
}

func (of *OutputFormatter) FormatAsJSON() (string, error) {
	cycles := of.analyzer.FindMinimalCycles()
	
	result := map[string]interface{}{
		"cycle":           of.analyzer.cycle,
		"minimal_cycles":  cycles,
		"resource_types":  of.analyzer.cycle.GetResourceTypes(),
		"total_resources": len(of.analyzer.cycle.Nodes),
	}
	
	if len(cycles) > 0 {
		result["suggestions"] = of.analyzer.GenerateSuggestions(cycles[0])
	}
	
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	return string(jsonData), nil
}

func (of *OutputFormatter) writeVerboseInfo(output *strings.Builder) {
	output.WriteString("📊 ANALYSIS SUMMARY\n")
	output.WriteString(fmt.Sprintf("Total resources in cycle: %d\n", len(of.analyzer.cycle.Nodes)))
	
	resourceTypes := of.analyzer.cycle.GetResourceTypes()
	output.WriteString("Resource types:\n")
	for resType, count := range resourceTypes {
		output.WriteString(fmt.Sprintf("  • %s: %d\n", resType, count))
	}
	output.WriteString("\n")
}

func (of *OutputFormatter) writeMinimalCycles(output *strings.Builder, cycles [][]string) {
	if len(cycles) == 1 && len(cycles[0]) == len(of.analyzer.cycle.Nodes) {
		output.WriteString(fmt.Sprintf("Full Cycle (%d resources):\n", len(cycles[0])))
		of.writeCycleDetails(output, cycles[0], true)
	} else {
		for i, cycle := range cycles {
			if i >= 3 {
				output.WriteString(fmt.Sprintf("... and %d more cycles\n\n", len(cycles)-i))
				break
			}
			
			output.WriteString(fmt.Sprintf("Minimal Cycle #%d (%d resources):\n", i+1, len(cycle)))
			of.writeCycleDetails(output, cycle, false)
		}
	}
}

func (of *OutputFormatter) writeCycleDetails(output *strings.Builder, cycle []string, showAll bool) {
	maxDisplay := len(cycle)
	if !showAll && len(cycle) > 10 {
		maxDisplay = 10
	}
	
	for i := 0; i < maxDisplay; i++ {
		nodeName := cycle[i]
		node := of.analyzer.cycle.GetNodeByName(nodeName)
		
		output.WriteString(fmt.Sprintf("  %d. %s", i+1, nodeName))
		
		if node != nil && node.Action != ActionNormal {
			output.WriteString(fmt.Sprintf(" (%s)", node.Action.String()))
		}
		
		if i < len(cycle)-1 {
			nextNodeName := cycle[i+1]
			output.WriteString(fmt.Sprintf("\n     ↳ depends on %s", nextNodeName))
		} else {
			output.WriteString(fmt.Sprintf("\n     ↳ depends on %s", cycle[0]))
		}
		output.WriteString("\n")
	}
	
	if !showAll && len(cycle) > maxDisplay {
		output.WriteString(fmt.Sprintf("     ... and %d more resources\n", len(cycle)-maxDisplay))
	}
	
	output.WriteString("\n")
}

func (of *OutputFormatter) writeSuggestions(output *strings.Builder, cycles [][]string) {
	if len(cycles) == 0 {
		return
	}
	
	output.WriteString("💡 SUGGESTIONS:\n")
	
	suggestions := of.analyzer.GenerateSuggestions(cycles[0])
	for _, suggestion := range suggestions {
		output.WriteString(fmt.Sprintf("  • %s\n", suggestion))
	}
	
	output.WriteString("\n")
	output.WriteString("🔧 COMMON SOLUTIONS:\n")
	output.WriteString("  • Use lifecycle { create_before_destroy = true } for replacement scenarios\n")
	output.WriteString("  • Replace direct references with data source lookups\n")
	output.WriteString("  • Split complex resources into multiple Terraform configurations\n")
	output.WriteString("  • Use depends_on explicitly to control dependency order\n")
	output.WriteString("\n")
}

func (of *OutputFormatter) writeAllResources(output *strings.Builder) {
	output.WriteString("📋 ALL RESOURCES IN CYCLE:\n")
	
	for i, node := range of.analyzer.cycle.Nodes {
		output.WriteString(fmt.Sprintf("  %d. %s", i+1, node.String()))
		
		if len(node.ModulePath) > 0 {
			output.WriteString(fmt.Sprintf(" (module: %s)", strings.Join(node.ModulePath, ".")))
		}
		
		if node.InstanceKey != "" {
			output.WriteString(fmt.Sprintf(" [%s]", node.InstanceKey))
		}
		
		output.WriteString("\n")
	}
	output.WriteString("\n")
}

func (of *OutputFormatter) GenerateVisualization() string {
	var output strings.Builder
	
	output.WriteString("digraph terraform_cycle {\n")
	output.WriteString("  rankdir=LR;\n")
	output.WriteString("  node [shape=box, style=rounded];\n\n")
	
	cycles := of.analyzer.FindMinimalCycles()
	if len(cycles) == 0 {
		return ""
	}
	
	cycle := cycles[0]
	
	nodeLabels := make(map[string]string)
	for _, nodeName := range cycle {
		node := of.analyzer.cycle.GetNodeByName(nodeName)
		if node != nil {
			label := fmt.Sprintf("%s.%s", node.ResourceType, node.ResourceName)
			if node.InstanceKey != "" {
				label += fmt.Sprintf("[%s]", node.InstanceKey)
			}
			nodeLabels[nodeName] = label
		} else {
			nodeLabels[nodeName] = nodeName
		}
	}
	
	for nodeName, label := range nodeLabels {
		node := of.analyzer.cycle.GetNodeByName(nodeName)
		color := "lightblue"
		if node != nil {
			switch node.Action {
			case ActionDestroy, ActionDestroyDeposed:
				color = "lightcoral"
			case ActionExpand:
				color = "lightyellow"
			case ActionClose:
				color = "lightgreen"
			}
		}
		
		cleanName := strings.ReplaceAll(nodeName, ".", "_")
		cleanName = strings.ReplaceAll(cleanName, "[", "_")
		cleanName = strings.ReplaceAll(cleanName, "]", "_")
		cleanName = strings.ReplaceAll(cleanName, "\"", "")
		
		output.WriteString(fmt.Sprintf("  %s [label=\"%s\", fillcolor=%s, style=filled];\n", 
			cleanName, label, color))
	}
	
	output.WriteString("\n")
	
	for i, nodeName := range cycle {
		nextIndex := (i + 1) % len(cycle)
		nextNodeName := cycle[nextIndex]
		
		cleanFrom := strings.ReplaceAll(nodeName, ".", "_")
		cleanFrom = strings.ReplaceAll(cleanFrom, "[", "_")
		cleanFrom = strings.ReplaceAll(cleanFrom, "]", "_")
		cleanFrom = strings.ReplaceAll(cleanFrom, "\"", "")
		
		cleanTo := strings.ReplaceAll(nextNodeName, ".", "_")
		cleanTo = strings.ReplaceAll(cleanTo, "[", "_")
		cleanTo = strings.ReplaceAll(cleanTo, "]", "_")
		cleanTo = strings.ReplaceAll(cleanTo, "\"", "")
		
		output.WriteString(fmt.Sprintf("  %s -> %s;\n", cleanFrom, cleanTo))
	}
	
	output.WriteString("}\n")
	
	return output.String()
}