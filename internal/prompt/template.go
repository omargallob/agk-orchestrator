package prompt

// Package prompt provides template resolution for agent prompts.
package prompt

import (
    "fmt"
    "strings"
)

// ResolveTemplate resolves a prompt template by replacing variables like {input}, {context}, {tool_call}.
// Returns the resolved prompt and any error.
func ResolveTemplate(template, input, context string) (string, error) {
    // Replace {input} with input value
    if input != "" {
        template = strings.ReplaceAll(template, "{input}", input)
    }
    
    // Replace {context} with context value
    if context != "" {
        template = strings.ReplaceAll(template, "{context}", context)
    }
    
    // Replace {tool_call} with tool call result (optional)
    // This can be extended later with tool result injection
    template = strings.ReplaceAll(template, "{tool_call}", "")
    
    return template, nil
}
