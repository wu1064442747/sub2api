package service

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOpenAIResponsesToolAccountCompatible(t *testing.T) {
	cc := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{"openai_responses_supported": false}}
	native := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{"openai_responses_supported": true}}
	tools := []byte(`{"tools":[{"type":"function","name":"read_file","parameters":{"type":"object"}}]}`)
	require.False(t, OpenAIResponsesToolAccountCompatible(cc, "gpt-6.1-sol", tools))
	require.False(t, OpenAIResponsesToolAccountCompatible(cc, "gpt-6.1-terra", tools))
	require.True(t, OpenAIResponsesToolAccountCompatible(native, "gpt-6.1-sol", tools))
	require.True(t, OpenAIResponsesToolAccountCompatible(cc, "gpt-6.1-sol", []byte(`{"tools":[]}`)))
	require.True(t, OpenAIResponsesToolAccountCompatible(cc, "gpt-5.6-terra", tools))
	alias := *cc
	alias.Credentials = map[string]any{"model_mapping": map[string]any{"my-model": "gpt-6.1-sol"}}
	require.False(t, OpenAIResponsesToolAccountCompatible(&alias, "my-model", tools))
	grok := &Account{Platform: PlatformGrok, Type: AccountTypeAPIKey}
	require.True(t, OpenAIResponsesToolAccountCompatible(grok, "gpt-6.1-sol", tools))
}

func TestOpenAIResponsesToolAccountCompatibleRuntimeTools(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Extra: map[string]any{"openai_responses_supported": false}}
	for _, body := range []string{
		`{"input":[{"type":"additional_tools","tools":[{"type":"function","name":"read_file","parameters":{"type":"object"}}]}]}`,
		`{"tools":[],"input":[{"type":"additional_tools","tools":[{"type":"function","name":"read_file","parameters":{"type":"object"}}]}]}`,
	} {
		require.False(t, OpenAIResponsesToolAccountCompatible(account, "gpt-6.1-sol", []byte(body)))
	}
}
