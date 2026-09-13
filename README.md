| Feature | Status |
|---------|--------|
| Tool Calling Loop | ✅ Implemented |
| Configuration Support | ✅ Added | 
| Unit Tests | ✅ Added |
| Integration Tests | ✅ Added |
| Documentation | ✅ Updated |

## Tool Calling Loop

The orchestrator now supports a multi-step tool-calling loop that enables agents to execute a sequence of tools to complete complex tasks. This is achieved by wiring the existing `internal/tools` registry into AgenticGoKit's reasoning pipeline.

### Configuration

The tool-calling loop is configured via the TOML config file with the following sections:

```toml
[tools.reasoning]
enabled = true
max_iterations = 5
max_concurrent = 1

[tools.tool_call]
enabled = true
allowlist = ["fetch", "file", "shell"]
reasoning = { enabled = true, max_iterations = 3, max_concurrent = 1 }
```

- `enabled`: Whether to enable the reasoning loop (default: false)
- `max_iterations`: Maximum number of tool iterations (default: 5)
- `max_concurrent`: Maximum number of tool calls executed in parallel (default: 1)

### How It Works

1. The agent is built with the reasoning loop enabled
2. When the agent receives a request, it runs with the tool definitions in the prompt
3. If the model returns a tool call, the loop executes the tool and appends the result to the conversation history
4. The process repeats until no tool calls are returned or the max iterations are reached

### Safety Guards

- **Max iteration cap**: Prevents infinite loops
- **No-progress detection**: Detects repeated identical tool calls and breaks early
- **Context timeout**: Each tool call has a timeout to prevent hanging

### Limitations

- MLX model tool-calling reliability varies by model family; Qwen and Llama families are most consistent
- The tool execution is synchronous and may not scale well for very large workflows
- The allowlist restricts which tools can be used

## Next Steps

- Add integration tests to verify end-to-end behavior with `mlx_lm.server`
- Improve error handling for tool execution failures
- Add support for custom tool types
- Optimize performance for large workflows

## References

- AgenticGoKit tool calling pipeline: `FormatToolsForPrompt`, `ParseToolCalls`, five format handlers
- AgenticGoKit tool discovery: `DiscoverInternalTools()`, `RegisterInternalTool()`, `ExecuteToolByName()`
- Reasoning loop: `executeNativeToolsAndContinue` with `MaxIterations` and `MaxConcurrent`
- Tool interface: `Name()`, `Description()`, `Execute()`

For more information, see the [AGENTS.md](AGENTS.md) file.