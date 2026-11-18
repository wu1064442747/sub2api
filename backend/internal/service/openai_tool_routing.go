package service

import (
	"context"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
)

// OpenAIResponsesToolAccountCompatible prevents the Responses-to-Chat fallback
// from rejecting tool requests after selection. Resolve account aliases exactly
// as forwarding does. Other models and providers retain their compatibility path.
func OpenAIResponsesToolAccountCompatible(account *Account, model string, body []byte) bool {
	if account == nil {
		return false
	}
	if account.Platform != PlatformOpenAI || account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityResponses) {
		return true
	}
	upstreamModel := normalizeOpenAIModelForUpstream(account, resolveOpenAIForwardModel(account, model, ""))
	if !openai.RequiresResponsesForToolCalls(upstreamModel) {
		return true
	}
	rawTools := gjson.GetBytes(body, "tools")
	var req apicompat.ResponsesRequest
	if rawTools.Exists() && json.Unmarshal([]byte(rawTools.Raw), &req.Tools) != nil {
		return true
	} // Forwarding owns malformed-request validation.
	if len(req.Tools) > 0 {
		return false
	}
	// Only additional_tools can add declarations when top-level tools is empty.
	// Extract those items rather than decoding a potentially huge conversation.
	req.Input = json.RawMessage(gjson.GetBytes(body, `input.#(type=="additional_tools")#`).Raw)
	tools, err := apicompat.EffectiveResponsesTools(&req)
	return err != nil || len(tools) == 0
}

// OpenAIChatToolAccountCompatible checks whether Chat Completions forwarding
// would reject GPT-6.2 Sol tool declarations on a native Chat Completions upstream.
func (s *OpenAIGatewayService) OpenAIChatToolAccountCompatible(ctx context.Context, account *Account, body []byte) bool {
	if account.Platform != PlatformOpenAI || !hasOpenAIChatToolDeclarations(body) {
		return true
	}
	model := resolveMappedUpstreamModel(account, body, "")
	if openai.GPT62ModelID(model) != "gpt-6.2-sol" {
		return true
	}
	inbound := APIProtocolChatCompletions
	if !gjson.GetBytes(body, "messages").Exists() && gjson.GetBytes(body, "input").Exists() {
		inbound = APIProtocolResponses
	}
	routingModel := upstreamRoutingModel(account, body, "")
	return s.resolveUpstreamProtocolFor(ctx, account, inbound, routingModel) != APIProtocolChatCompletions
}

func hasOpenAIChatToolDeclarations(body []byte) bool {
	return len(gjson.GetBytes(body, "tools").Array()) > 0 || len(gjson.GetBytes(body, "functions").Array()) > 0
}
