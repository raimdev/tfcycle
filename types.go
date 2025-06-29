package main

import (
	"fmt"
	"strings"
)

type NodeAction int

const (
	ActionNormal NodeAction = iota
	ActionExpand
	ActionDestroy
	ActionClose
	ActionDestroyDeposed
)

func (a NodeAction) String() string {
	switch a {
	case ActionExpand:
		return "expand"
	case ActionDestroy:
		return "destroy"
	case ActionClose:
		return "close"
	case ActionDestroyDeposed:
		return "destroy_deposed"
	default:
		return "normal"
	}
}

type CycleNode struct {
	ResourceType   string            `json:"resource_type"`
	ResourceName   string            `json:"resource_name"`
	ModulePath     []string          `json:"module_path"`
	InstanceKey    string            `json:"instance_key,omitempty"`
	Action         NodeAction        `json:"action"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	RawString      string            `json:"raw_string"`
}

func (n *CycleNode) FullName() string {
	parts := make([]string, 0, len(n.ModulePath)+2)
	parts = append(parts, n.ModulePath...)
	parts = append(parts, n.ResourceType+"."+n.ResourceName)
	
	result := strings.Join(parts, ".")
	if n.InstanceKey != "" {
		result += "[" + n.InstanceKey + "]"
	}
	
	return result
}

func (n *CycleNode) String() string {
	name := n.FullName()
	if n.Action != ActionNormal {
		name += fmt.Sprintf(" (%s", n.Action.String())
		if n.Action == ActionDestroyDeposed && n.Annotations["deposed_id"] != "" {
			name += " " + n.Annotations["deposed_id"]
		}
		name += ")"
	}
	return name
}

type TfCycle struct {
	Nodes     []*CycleNode `json:"nodes"`
	RawError  string       `json:"raw_error"`
	Cycles    [][]string   `json:"cycles,omitempty"`
}

func (tc *TfCycle) GetNodeByName(name string) *CycleNode {
	for _, node := range tc.Nodes {
		if node.FullName() == name {
			return node
		}
	}
	return nil
}

func (tc *TfCycle) GetResourceTypes() map[string]int {
	types := make(map[string]int)
	for _, node := range tc.Nodes {
		types[node.ResourceType]++
	}
	return types
}