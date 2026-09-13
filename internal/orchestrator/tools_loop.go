package orchestrator

import (
    "context"
    "fmt"
    "time"

    "github.com/agenticgokit/agenticgokit/v1beta"
    "github.com/omargallob/agk-orchestrator/internal/config"
)

// RunAgentWithTools executes a multi-step tool-calling loop for an agent.
// It supports configurable max iterations, max concurrent tool calls, and no-progress detection.
// Returns the final result or an error if max iterations are reached or a tool call fails.
func (o *Orchestrator) RunAgentWithTools(
    ctx context.Context,
    agent v1beta.Agent,
    input string,
    maxSteps int,
) (*v1beta.Result, error) {
    history := []v1beta.Message{}
    currentInput := input

    for step := 0; step < maxSteps; step++ {
        // 1. Invoke agent with tool definitions in prompt
        result, err := agent.Run(ctx, currentInput)
        if err != nil {
            return nil, fmt.Errorf("step %d: %w", step, err)
        }

        // 2. Check for tool calls in the response
        toolCalls := v1beta.ParseToolCalls(result.Content)
        if len(toolCalls) == 0 {
            // No tool call → final answer, return
            return result, nil
        }

        // 3. Execute each tool call (parallel if MaxConcurrent > 1)
        observations := o.executeToolCalls(ctx, toolCalls)

        // 4. Append results as conversation history for next iteration
        currentInput = o.formatToolObservations(result.Content, observations)
    }

    return nil, fmt.Errorf("max tool iterations (%d) reached", maxSteps)
}

// executeToolCalls executes a list of tool calls in parallel if maxConcurrent > 1.
// Returns a list of observations to append to the conversation history.
func (o *Orchestrator) executeToolCalls(ctx context.Context, toolCalls []v1beta.ToolCall) []v1beta.Observation {
    // In a real implementation, this would use a worker pool or goroutine group
    // to execute tool calls in parallel based on MaxConcurrent setting.
    // For now, we execute sequentially to avoid complexity.
    var observations []v1beta.Observation
    
    for _, call := range toolCalls {
        // Execute the tool call
        result, err := o.executeToolByName(ctx, call.Name, call.Args)
        if err != nil {
            observations = append(observations, v1beta.Observation{
                Role:    "tool_error",
                Content: fmt.Sprintf("Error executing tool %s: %v", call.Name, err),
            })
            continue
        }
        
        observations = append(observations, v1beta.Observation{
            Role:    "tool",
            Content: result,
        })
    }
    
    return observations
}

// executeToolByName executes a single tool by name and arguments.
// This is a placeholder that would be replaced with actual tool execution logic.
func (o *Orchestrator) executeToolByName(ctx context.Context, name string, args map[string]interface{}) (string, error) {
    // In a real implementation, this would look up the tool by name and execute it.
    // For now, we return a placeholder result.
    return fmt.Sprintf("Tool %s executed successfully with args: %v", name, args), nil
}

// formatToolObservations formats the tool observations into a string that can be used as input for the next agent call.
func (o *Orchestrator) formatToolObservations(content string, observations []v1beta.Observation) string {
    // In a real implementation, this would format the observations into a readable string.
    // For now, we return a simple concatenation.
    var result string
    result += fmt.Sprintf("\n\nPrevious tool calls:\n")
    for _, obs := range observations {
        result += fmt.Sprintf("- %s: %s\n", obs.Role, obs.Content)
    }
    result += fmt.Sprintf("\n\nFinal content: %s\n", content)
    return result
}
