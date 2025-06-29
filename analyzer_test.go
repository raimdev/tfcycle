package main

import (
	"reflect"
	"testing"
)

func TestCycleAnalyzer_FindMinimalCycles(t *testing.T) {
	cycle := &TfCycle{
		Nodes: []*CycleNode{
			{ResourceType: "aws_security_group", ResourceName: "sg1"},
			{ResourceType: "aws_security_group", ResourceName: "sg2"},
			{ResourceType: "aws_instance", ResourceName: "web"},
		},
	}
	
	analyzer := NewCycleAnalyzer(cycle)
	cycles := analyzer.FindMinimalCycles()
	
	if len(cycles) == 0 {
		t.Errorf("Expected at least one cycle, got none")
	}
	
	if len(cycles[0]) > len(cycle.Nodes) {
		t.Errorf("Minimal cycle should not be larger than total nodes")
	}
}

func TestCycleAnalyzer_LikelyDependency(t *testing.T) {
	analyzer := &CycleAnalyzer{}
	
	sg1 := &CycleNode{ResourceType: "aws_security_group", ResourceName: "sg1"}
	sg2 := &CycleNode{ResourceType: "aws_security_group", ResourceName: "sg2"}
	instance := &CycleNode{ResourceType: "aws_instance", ResourceName: "web"}
	
	if !analyzer.likelyDependency(sg1, sg2) {
		t.Errorf("Security groups should have likely dependency")
	}
	
	if !analyzer.likelyDependency(instance, sg1) {
		t.Errorf("Instance should have likely dependency on security group")
	}
}

func TestCycleAnalyzer_GenerateSuggestions_SecurityGroups(t *testing.T) {
	cycle := &TfCycle{
		Nodes: []*CycleNode{
			{ResourceType: "aws_security_group", ResourceName: "sg1"},
			{ResourceType: "aws_security_group", ResourceName: "sg2"},
		},
	}
	
	analyzer := NewCycleAnalyzer(cycle)
	suggestions := analyzer.GenerateSuggestions([]string{
		"aws_security_group.sg1",
		"aws_security_group.sg2",
	})
	
	found := false
	for _, suggestion := range suggestions {
		if contains(suggestion, "security group") || contains(suggestion, "Security group") {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected security group specific suggestion, got: %v", suggestions)
	}
}

func TestCycleAnalyzer_GenerateSuggestions_IAM(t *testing.T) {
	cycle := &TfCycle{
		Nodes: []*CycleNode{
			{ResourceType: "aws_iam_role", ResourceName: "role1"},
			{ResourceType: "aws_iam_policy", ResourceName: "policy1"},
		},
	}
	
	analyzer := NewCycleAnalyzer(cycle)
	suggestions := analyzer.GenerateSuggestions([]string{
		"aws_iam_role.role1",
		"aws_iam_policy.policy1",
	})
	
	found := false
	for _, suggestion := range suggestions {
		if contains(suggestion, "IAM") {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected IAM specific suggestion, got: %v", suggestions)
	}
}

func TestCycleAnalyzer_GenerateSuggestions_DestroyAction(t *testing.T) {
	cycle := &TfCycle{
		Nodes: []*CycleNode{
			{ResourceType: "aws_instance", ResourceName: "web", Action: ActionDestroy},
			{ResourceType: "aws_security_group", ResourceName: "sg1"},
		},
	}
	
	analyzer := NewCycleAnalyzer(cycle)
	suggestions := analyzer.GenerateSuggestions([]string{
		"aws_instance.web",
		"aws_security_group.sg1",
	})
	
	found := false
	for _, suggestion := range suggestions {
		if contains(suggestion, "create_before_destroy") {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("Expected create_before_destroy suggestion for destroy action, got: %v", suggestions)
	}
}

func TestCycleAnalyzer_ShareModulePath(t *testing.T) {
	analyzer := &CycleAnalyzer{}
	
	pathA := []string{"module", "vpc", "module", "security"}
	pathB := []string{"module", "vpc"}
	pathC := []string{"module", "app"}
	
	if !analyzer.shareModulePath(pathA, pathB) {
		t.Errorf("Paths sharing 'module.vpc' should return true")
	}
	
	if analyzer.shareModulePath(pathA, pathC) {
		t.Errorf("Paths not sharing prefix should return false")
	}
}

func TestCycleAnalyzer_NormalizeCycle(t *testing.T) {
	analyzer := &CycleAnalyzer{}
	
	cycle := []string{"resource.c", "resource.a", "resource.b"}
	normalized := analyzer.normalizeCycle(cycle)
	expected := []string{"resource.a", "resource.b", "resource.c"}
	
	if !reflect.DeepEqual(normalized, expected) {
		t.Errorf("Expected %v, got %v", expected, normalized)
	}
}

func TestTfCycle_GetNodeByName(t *testing.T) {
	cycle := &TfCycle{
		Nodes: []*CycleNode{
			{ResourceType: "aws_security_group", ResourceName: "sg1"},
			{ResourceType: "aws_instance", ResourceName: "web", InstanceKey: "key1"},
		},
	}
	
	node := cycle.GetNodeByName("aws_security_group.sg1")
	if node == nil {
		t.Errorf("Expected to find node 'aws_security_group.sg1'")
	}
	
	node = cycle.GetNodeByName("aws_instance.web[key1]")
	if node == nil {
		t.Errorf("Expected to find node 'aws_instance.web[key1]'")
	}
	
	node = cycle.GetNodeByName("nonexistent.resource")
	if node != nil {
		t.Errorf("Expected nil for nonexistent resource")
	}
}

func TestTfCycle_GetResourceTypes(t *testing.T) {
	cycle := &TfCycle{
		Nodes: []*CycleNode{
			{ResourceType: "aws_security_group", ResourceName: "sg1"},
			{ResourceType: "aws_security_group", ResourceName: "sg2"},
			{ResourceType: "aws_instance", ResourceName: "web"},
		},
	}
	
	types := cycle.GetResourceTypes()
	
	if types["aws_security_group"] != 2 {
		t.Errorf("Expected 2 security groups, got %d", types["aws_security_group"])
	}
	
	if types["aws_instance"] != 1 {
		t.Errorf("Expected 1 instance, got %d", types["aws_instance"])
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
		 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}