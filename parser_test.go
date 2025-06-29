package main

import (
	"reflect"
	"testing"
)

func TestParser_ParseError_SimpleCase(t *testing.T) {
	parser := NewParser()
	errorText := "Error: Cycle: aws_security_group.sg_ping, aws_security_group.sg_8080"
	
	cycle, err := parser.ParseError(errorText)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if len(cycle.Nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(cycle.Nodes))
	}
	
	node1 := cycle.Nodes[0]
	if node1.ResourceType != "aws_security_group" || node1.ResourceName != "sg_ping" {
		t.Errorf("Expected aws_security_group.sg_ping, got %s.%s", node1.ResourceType, node1.ResourceName)
	}
	
	node2 := cycle.Nodes[1]
	if node2.ResourceType != "aws_security_group" || node2.ResourceName != "sg_8080" {
		t.Errorf("Expected aws_security_group.sg_8080, got %s.%s", node2.ResourceType, node2.ResourceName)
	}
}

func TestParser_ParseError_WithModules(t *testing.T) {
	parser := NewParser()
	errorText := `Error: Cycle: module.vpc.aws_security_group.sg_ping, module.vpc.module.security.aws_security_group.sg_8080`
	
	cycle, err := parser.ParseError(errorText)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if len(cycle.Nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(cycle.Nodes))
	}
	
	node1 := cycle.Nodes[0]
	expectedPath1 := []string{"module", "vpc"}
	if !reflect.DeepEqual(node1.ModulePath, expectedPath1) {
		t.Errorf("Expected module path %v, got %v", expectedPath1, node1.ModulePath)
	}
	
	node2 := cycle.Nodes[1]
	expectedPath2 := []string{"module", "vpc", "module", "security"}
	if !reflect.DeepEqual(node2.ModulePath, expectedPath2) {
		t.Errorf("Expected module path %v, got %v", expectedPath2, node2.ModulePath)
	}
}

func TestParser_ParseError_WithInstanceKeys(t *testing.T) {
	parser := NewParser()
	errorText := `Error: Cycle: aws_instance.web["key1"], aws_instance.web[0]`
	
	cycle, err := parser.ParseError(errorText)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if len(cycle.Nodes) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(cycle.Nodes))
	}
	
	node1 := cycle.Nodes[0]
	if node1.InstanceKey != "key1" {
		t.Errorf("Expected instance key 'key1', got '%s'", node1.InstanceKey)
	}
	
	node2 := cycle.Nodes[1]
	if node2.InstanceKey != "0" {
		t.Errorf("Expected instance key '0', got '%s'", node2.InstanceKey)
	}
}

func TestParser_ParseError_WithActions(t *testing.T) {
	parser := NewParser()
	errorText := `Error: Cycle: aws_instance.web (destroy), module.app.local.config (expand), module.app (close), aws_security_group.sg (destroy deposed abc123)`
	
	cycle, err := parser.ParseError(errorText)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if len(cycle.Nodes) != 4 {
		t.Fatalf("Expected 4 nodes, got %d", len(cycle.Nodes))
	}
	
	expectedActions := []NodeAction{ActionDestroy, ActionExpand, ActionClose, ActionDestroyDeposed}
	for i, expectedAction := range expectedActions {
		if cycle.Nodes[i].Action != expectedAction {
			t.Errorf("Node %d: expected action %v, got %v", i, expectedAction, cycle.Nodes[i].Action)
		}
	}
	
	if cycle.Nodes[3].Annotations["deposed_id"] != "abc123" {
		t.Errorf("Expected deposed_id 'abc123', got '%s'", cycle.Nodes[3].Annotations["deposed_id"])
	}
}

func TestParser_ParseError_ComplexCase(t *testing.T) {
	parser := NewParser()
	errorText := `Error: Cycle: module.vpc.module.security.aws_security_group.sg_ping["prod"], 
module.vpc.aws_instance.web[0] (destroy), 
module.app.local.config (expand), 
module.app (close),
module.vpc.module.security.aws_security_group.sg_8080 (destroy deposed abc123ef)`
	
	cycle, err := parser.ParseError(errorText)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if len(cycle.Nodes) < 5 {
		t.Logf("Parsed nodes:")
		for i, node := range cycle.Nodes {
			t.Logf("  %d: %s", i, node.RawString)
		}
		t.Fatalf("Expected at least 5 nodes, got %d", len(cycle.Nodes))
	}
	
	node1 := cycle.Nodes[0]
	expectedPath := []string{"module", "vpc", "module", "security"}
	if !reflect.DeepEqual(node1.ModulePath, expectedPath) {
		t.Errorf("Expected module path %v, got %v", expectedPath, node1.ModulePath)
	}
	if node1.InstanceKey != "prod" {
		t.Errorf("Expected instance key 'prod', got '%s'", node1.InstanceKey)
	}
	if node1.Action != ActionNormal {
		t.Errorf("Expected normal action, got %v", node1.Action)
	}
}

func TestParser_ParseError_InvalidInput(t *testing.T) {
	parser := NewParser()
	errorText := "This is not a cycle error"
	
	_, err := parser.ParseError(errorText)
	if err == nil {
		t.Errorf("Expected error for invalid input, got nil")
	}
}

func TestParser_SplitResources(t *testing.T) {
	parser := NewParser()
	
	testCases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "aws_security_group.sg1, aws_security_group.sg2",
			expected: []string{"aws_security_group.sg1", "aws_security_group.sg2"},
		},
		{
			input:    "aws_instance.web[\"key1\"], aws_instance.web[0]",
			expected: []string{"aws_instance.web[\"key1\"]", "aws_instance.web[0]"},
		},
		{
			input:    "resource.name (action, with, commas), other.resource",
			expected: []string{"resource.name (action, with, commas)", "other.resource"},
		},
	}
	
	for i, tc := range testCases {
		result := parser.splitResources(tc.input)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("Test case %d: expected %v, got %v", i, tc.expected, result)
		}
	}
}

func TestCycleNode_FullName(t *testing.T) {
	node := &CycleNode{
		ResourceType: "aws_security_group",
		ResourceName: "sg_test",
		ModulePath:   []string{"module", "vpc"},
		InstanceKey:  "key1",
	}
	
	expected := "module.vpc.aws_security_group.sg_test[key1]"
	if node.FullName() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, node.FullName())
	}
}

func TestCycleNode_String(t *testing.T) {
	node := &CycleNode{
		ResourceType: "aws_security_group",
		ResourceName: "sg_test",
		Action:       ActionDestroy,
	}
	
	expected := "aws_security_group.sg_test (destroy)"
	if node.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, node.String())
	}
}