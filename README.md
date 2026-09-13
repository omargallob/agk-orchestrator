## Prompt Templates

Agents can use prompt templates to dynamically generate system prompts with variables like:

- `{input}`: The user input to the agent
- `{context}`: Relevant context or history
- `{tool_call}`: (Optional) Result of a tool call (future extension)

### Example

```toml
[agents]\n[agents.my-agent]\nname = "my-agent"
provider = "vllm"
model = "mlx-7b"
base_url = "http://localhost:8080/v1"
prompt_template = "You are an agent. Input: {input}. Context: {context}"
```

The template is resolved at runtime, with variables replaced by their values.