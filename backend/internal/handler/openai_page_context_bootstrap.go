package handler

import (
	"encoding/json"
	"strings"
)

// Codex Desktop injects page context as unpaired tool outputs. These are user
// context, not results of model-issued calls. Only recognize the observed wire
// shapes; never manufacture a call_id for arbitrary outputs.
func normalizeCodexPageContextBootstrap(body []byte) ([]byte, bool) {
	if !mayContainCodexCallOutputCandidate(body, "codex_apps_open_page", "codex_apps_selected_page_part_1_of_1") {
		return body, false
	}
	candidate := func(item map[string]any) bool {
		return isCodexPageContextCandidate(item) || isCodexDelegationCandidate(item)
	}
	if normalized, changed := normalizeCodexCallOutputBootstrap(body, candidate, true); changed {
		return normalized, true
	}
	// Automation messages may coexist only in a bootstrap with no historical
	// tool context, preserving the original automation normalization boundary.
	return normalizeCodexCallOutputBootstrap(body, func(item map[string]any) bool {
		return candidate(item) || isCodexAutomationCandidate(item)
	}, false)
}

func isCodexPageContextCandidate(item map[string]any) bool {
	if stringField(item, "type") != "function_call_output" || stringField(item, "namespace") != "agent" {
		return false
	}
	output, ok := item["output"].(string)
	if !ok {
		return false
	}
	switch stringField(item, "name") {
	case "codex_apps_open_page":
	case "codex_apps_selected_page_part_1_of_1":
		const prefix = "Companion Page snapshot part 1 of 1:\n"
		if !strings.HasPrefix(output, prefix) {
			return false
		}
		output = strings.TrimPrefix(output, prefix)
	default:
		return false
	}
	if !hasUniqueJSONMembers([]byte(output)) {
		return false
	}
	var page map[string]json.RawMessage
	if json.Unmarshal([]byte(output), &page) != nil || page == nil {
		return false
	}
	id, exists := page["page_id"]
	if !exists {
		return false
	}
	if string(id) == "null" {
		return true
	}
	var value string
	return json.Unmarshal(id, &value) == nil && strings.TrimSpace(value) != ""
}
