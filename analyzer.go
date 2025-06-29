package main

import (
	"sort"
	"strings"
)

type CycleAnalyzer struct {
	cycle *TfCycle
}

func NewCycleAnalyzer(cycle *TfCycle) *CycleAnalyzer {
	return &CycleAnalyzer{cycle: cycle}
}

func (ca *CycleAnalyzer) FindMinimalCycles() [][]string {
	nodeNames := make([]string, len(ca.cycle.Nodes))
	for i, node := range ca.cycle.Nodes {
		nodeNames[i] = node.FullName()
	}

	graph := ca.buildHypotheticalGraph(nodeNames)
	
	cycles := ca.findCyclesInGraph(graph, nodeNames)
	
	sort.Slice(cycles, func(i, j int) bool {
		return len(cycles[i]) < len(cycles[j])
	})
	
	ca.cycle.Cycles = cycles
	return cycles
}

func (ca *CycleAnalyzer) buildHypotheticalGraph(nodeNames []string) map[string][]string {
	graph := make(map[string][]string)
	
	for _, name := range nodeNames {
		graph[name] = []string{}
	}
	
	resourceTypes := make(map[string][]*CycleNode)
	for _, node := range ca.cycle.Nodes {
		resourceTypes[node.ResourceType] = append(resourceTypes[node.ResourceType], node)
	}
	
	for i, nodeA := range ca.cycle.Nodes {
		for j, nodeB := range ca.cycle.Nodes {
			if i == j {
				continue
			}
			
			if ca.likelyDependency(nodeA, nodeB) {
				graph[nodeA.FullName()] = append(graph[nodeA.FullName()], nodeB.FullName())
			}
		}
	}
	
	if ca.allNodesHaveConnections(graph) {
		return graph
	}
	
	return ca.buildSequentialFallback(nodeNames)
}

func (ca *CycleAnalyzer) likelyDependency(from, to *CycleNode) bool {
	if from.ResourceType == "aws_security_group" && to.ResourceType == "aws_security_group" {
		return true
	}
	
	if from.ResourceType == "aws_instance" && to.ResourceType == "aws_security_group" {
		return true
	}
	
	if from.ResourceType == "aws_security_group" && to.ResourceType == "aws_instance" {
		return true
	}
	
	if strings.HasPrefix(from.ResourceType, "aws_iam") && strings.HasPrefix(to.ResourceType, "aws_iam") {
		return true
	}
	
	if len(from.ModulePath) > 0 && len(to.ModulePath) > 0 {
		return ca.shareModulePath(from.ModulePath, to.ModulePath)
	}
	
	if from.Action == ActionDestroy && to.Action != ActionDestroy {
		return true
	}
	
	return false
}

func (ca *CycleAnalyzer) shareModulePath(pathA, pathB []string) bool {
	minLen := len(pathA)
	if len(pathB) < minLen {
		minLen = len(pathB)
	}
	
	if minLen == 0 {
		return false
	}
	
	for i := 0; i < minLen; i++ {
		if pathA[i] != pathB[i] {
			return false
		}
	}
	return true
}

func (ca *CycleAnalyzer) allNodesHaveConnections(graph map[string][]string) bool {
	for _, connections := range graph {
		if len(connections) == 0 {
			return false
		}
	}
	return true
}

func (ca *CycleAnalyzer) buildSequentialFallback(nodeNames []string) map[string][]string {
	graph := make(map[string][]string)
	
	for i, name := range nodeNames {
		nextIndex := (i + 1) % len(nodeNames)
		graph[name] = []string{nodeNames[nextIndex]}
	}
	
	return graph
}

func (ca *CycleAnalyzer) findCyclesInGraph(graph map[string][]string, nodeNames []string) [][]string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var cycles [][]string
	
	var dfs func(node string, path []string) bool
	dfs = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)
		
		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor, path) {
					return true
				}
			} else if recStack[neighbor] {
				cycleStart := -1
				for i, pathNode := range path {
					if pathNode == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
				return true
			}
		}
		
		recStack[node] = false
		return false
	}
	
	for _, node := range nodeNames {
		if !visited[node] {
			dfs(node, []string{})
		}
	}
	
	if len(cycles) == 0 {
		cycles = append(cycles, nodeNames)
	}
	
	return ca.deduplicateCycles(cycles)
}

func (ca *CycleAnalyzer) deduplicateCycles(cycles [][]string) [][]string {
	seen := make(map[string]bool)
	var unique [][]string
	
	for _, cycle := range cycles {
		if len(cycle) < 2 {
			continue
		}
		
		normalized := ca.normalizeCycle(cycle)
		key := strings.Join(normalized, ",")
		
		if !seen[key] {
			seen[key] = true
			unique = append(unique, cycle)
		}
	}
	
	return unique
}

func (ca *CycleAnalyzer) normalizeCycle(cycle []string) []string {
	if len(cycle) == 0 {
		return cycle
	}
	
	minIndex := 0
	for i, node := range cycle {
		if node < cycle[minIndex] {
			minIndex = i
		}
	}
	
	normalized := make([]string, len(cycle))
	for i := 0; i < len(cycle); i++ {
		normalized[i] = cycle[(minIndex+i)%len(cycle)]
	}
	
	return normalized
}

func (ca *CycleAnalyzer) GenerateSuggestions(cycle []string) []string {
	var suggestions []string
	
	resourceTypes := make(map[string]int)
	for _, nodeName := range cycle {
		node := ca.cycle.GetNodeByName(nodeName)
		if node != nil {
			resourceTypes[node.ResourceType]++
		}
	}
	
	if resourceTypes["aws_security_group"] >= 2 {
		suggestions = append(suggestions, "Security group cycle detected: Remove mutual references between security groups")
		suggestions = append(suggestions, "Use separate aws_security_group_rule resources instead of inline rules")
		suggestions = append(suggestions, "Consider using data sources for existing security groups")
	}
	
	if resourceTypes["aws_iam_role"] > 0 && resourceTypes["aws_iam_policy"] > 0 {
		suggestions = append(suggestions, "IAM cycle detected: Separate role creation from policy attachment")
		suggestions = append(suggestions, "Use aws_iam_role_policy_attachment instead of inline policies")
	}
	
	hasDestroyAction := false
	for _, nodeName := range cycle {
		node := ca.cycle.GetNodeByName(nodeName)
		if node != nil && (node.Action == ActionDestroy || node.Action == ActionDestroyDeposed) {
			hasDestroyAction = true
			break
		}
	}
	
	if hasDestroyAction {
		suggestions = append(suggestions, "Destroy cycle detected: Add lifecycle { create_before_destroy = true }")
		suggestions = append(suggestions, "Review dependency order during resource replacement")
	}
	
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Break circular dependencies by removing direct references")
		suggestions = append(suggestions, "Use data sources to reference existing resources")
		suggestions = append(suggestions, "Consider splitting resources across multiple Terraform runs")
	}
	
	return suggestions
}