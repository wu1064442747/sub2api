package service

import (
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
