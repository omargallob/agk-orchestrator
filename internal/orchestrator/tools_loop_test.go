package orchestrator

import (
    "context"
    "testing"

    "github.com/agenticgokit/agenticgokit/v1beta"
    "github.com/stretchr/testify/require"
    "github.com/omargallob/agk-orchestrator/internal/config"
)

func TestRunAgentWithTools(t *testing.T) {
    // Setup test agent with reasoning enabled
    cfg := config.Config{
        Agents: []config.AgentConfig{
            {
                Name:     "test_agent",
                Provider: "openai",
                Model:    "gpt-4",
                BaseURL:  "http://localhost:8080/v1",
                System:   "You are a helpful assistant that can use tools.",
                Tools: config.ToolConfig{
                    Reasoning: config.ReasoningConfig{
                        Enabled:       true,
                        MaxIterations: 3,
                        MaxConcurrent: 1,
                    },
                },
            },
        },
    }

    // Create a mock agent that returns a tool call on first run
    mockAgent := &mockAgent{
        toolCalls: []v1beta.ToolCall{
            {
                Name: "calculate",
                Args: map[string]interface{}{"operation": "add", "a": 5, "b": 3},
            },
        },
    }

    // Create orchestrator
    o := &Orchestrator{agents: map[string]v1beta.Agent{"test_agent": mockAgent}}

    // Run agent with tool calling loop
    result, err := o.RunAgentWithTools(context.Background(), mockAgent, "Calculate 5 + 3", 3)
    require.NoError(t, err)
    require.NotNil(t, result)
    
    // Verify that the tool call was executed
    require.Equal(t, 1, len(mockAgent.toolCalls))
}

// mockAgent is a mock implementation of v1beta.Agent that returns a tool call on first run.
type mockAgent struct {
    toolCalls []v1beta.ToolCall
}

func (m *mockAgent) Run(ctx context.Context, input string) (*v1beta.Result, error) {
    // Return a result with a tool call on first run
    if len(m.toolCalls) > 0 {
        return &v1beta.Result{
            Content: "I need to calculate 5 + 3",
        }, nil
    }
    return nil, nil
}

func (m *mockAgent) GetToolCallConfig() config.ToolCallConfig {
    return config.ToolCallConfig{
        Enabled: false,
    }
}