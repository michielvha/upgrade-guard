package analyst

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// AnthropicAnalyst uses the Claude API with web search to analyze upgrade risks.
type AnthropicAnalyst struct{}

// NewAnthropicAnalyst creates a new AnthropicAnalyst.
// The API key is read from the ANTHROPIC_API_KEY environment variable.
func NewAnthropicAnalyst() *AnthropicAnalyst {
	return &AnthropicAnalyst{}
}

// Analyze runs the LLM analysis on the given component changes and platform state.
func (a *AnthropicAnalyst) Analyze(ctx context.Context, changes []ComponentChange, state *PlatformState, guardLevel string, model string) (*AnalysisResult, error) {
	client := anthropic.NewClient()

	prompt := BuildPrompt(changes, state, guardLevel)

	claudeModel := resolveModel(model)

	// Enable web search as a server-side tool
	tools := []anthropic.ToolUnionParam{
		{OfWebSearchTool20250305: &anthropic.WebSearchTool20250305Param{}},
	}

	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     claudeModel,
		MaxTokens: 8192,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
		Tools: tools,
	})
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Extract text content from the response, handling the agentic loop
	// where Claude may use web search and then produce text.
	result := extractResult(message, changes, guardLevel)

	// If the model used web search (stop_reason == "tool_use"), we need to
	// continue the conversation to get the final text response.
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	for message.StopReason == "tool_use" {
		// Add assistant's response (which includes tool use blocks)
		messages = append(messages, message.ToParam())

		// Add tool results - for server tools like web search,
		// the results are handled automatically by the API.
		// We just need to send back the message to continue.
		message, err = client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     claudeModel,
			MaxTokens: 8192,
			Messages:  messages,
			Tools:     tools,
		})
		if err != nil {
			return nil, fmt.Errorf("claude API continuation failed: %w", err)
		}

		result = extractResult(message, changes, guardLevel)
	}

	return result, nil
}

func extractResult(msg *anthropic.Message, changes []ComponentChange, guardLevel string) *AnalysisResult {
	var textParts []string

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			tb := block.AsText()
			textParts = append(textParts, tb.Text)
		}
	}

	markdown := strings.Join(textParts, "\n")

	return &AnalysisResult{
		RawMarkdown: markdown,
		GuardLevel:  guardLevel,
	}
}

func resolveModel(model string) anthropic.Model {
	switch model {
	case "claude-opus-4-6":
		return anthropic.ModelClaudeOpus4_6
	case "claude-sonnet-4-6":
		return anthropic.ModelClaudeSonnet4_6
	default:
		return anthropic.ModelClaudeSonnet4_6
	}
}
