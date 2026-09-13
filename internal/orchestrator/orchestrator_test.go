package orchestrator

import (
    "testing"
    "github.com/omargallob/agk-orchestrator/internal/prompt"
)

func TestResolveTemplate(t *testing.T) {
    // Test with input and context
    result, err := prompt.ResolveTemplate("You are an agent. Input: {input}. Context: {context}", "hello", "world")
    if err != nil {
        t.Fatalf("Error resolving template: %v", err)
    }
    if result != "You are an agent. Input: hello. Context: world" {
        t.Errorf("Expected: You are an agent. Input: hello. Context: world\nGot: %s", result)
    }
    
    // Test with only input
    result, err = prompt.ResolveTemplate("Input: {input}", "test", "")
    if err != nil {
        t.Fatalf("Error resolving template: %v", err)
    }
    if result != "Input: test" {
        t.Errorf("Expected: Input: test\nGot: %s", result)
    }
    
    // Test with only context
    result, err = prompt.ResolveTemplate("Context: {context}", "", "example")
    if err != nil {
        t.Fatalf("Error resolving template: %v", err)
    }
    if result != "Context: example" {
        t.Errorf("Expected: Context: example\nGot: %s", result)
    }
    
    // Test with no variables
    result, err = prompt.ResolveTemplate("No variables here.", "", "")
    if err != nil {
        t.Fatalf("Error resolving template: %v", err)
    }
    if result != "No variables here." {
        t.Errorf("Expected: No variables here.\nGot: %s", result)
    }
    
    // Test with invalid variable
    result, err = prompt.ResolveTemplate("Invalid {xyz}", "", "")
    if err == nil {
        t.Errorf("Expected error for invalid variable {xyz}")
    }
}
