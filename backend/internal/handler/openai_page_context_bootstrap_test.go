package handler

import (
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"testing"
)

func TestNormalizeCodexPageContextBootstrap(t *testing.T) {
	for _, item := range []string{
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_1","output":"Companion Page snapshot part 1 of 1:\n{\"page_id\":null,\"document_view\":null}"}`,
	} {
		body := []byte(`{"model":"gpt-6.1-sol","input":[` + item + `,{"type":"function_call","call_id":"call_1"},{"type":"function_call_output","call_id":"call_1","output":"done"}]}`)
		got, changed := normalizeCodexPageContextBootstrap(body)
		require.True(t, changed)
		require.Equal(t, "message", gjson.GetBytes(got, "input.0.type").String())
		require.Equal(t, "user", gjson.GetBytes(got, "input.0.role").String())
		require.Equal(t, gjson.GetBytes(body, "input.0.output").String(), gjson.GetBytes(got, "input.0.content.0.text").String())
		require.Equal(t, "call_1", gjson.GetBytes(got, "input.2.call_id").String())
		again, changed := normalizeCodexPageContextBootstrap(got)
		require.False(t, changed)
		require.Equal(t, got, again)
	}
}

func TestNormalizeCodexPageContextBootstrapRejectsAmbiguity(t *testing.T) {
	for _, item := range []string{
		`{"type":"function_call_output","namespace":"other","name":"codex_apps_open_page","output":"{\"page_id\":null}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"other","output":"{\"page_id\":null}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","call_id":"call_1","output":"{\"page_id\":null}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","call_id":42,"output":"{\"page_id\":null}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"not page context"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_selected_page_part_1_of_1","output":"Companion Page snapshot part 2 of 2:\n{\"page_id\":null,\"document_view\":null}"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},{"type":"function_call_output","output":"unpaired"}`,
		`{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"},{"type":"function_call"}`,
	} {
		body := []byte(`{"input":[` + item + `]}`)
		got, changed := normalizeCodexPageContextBootstrap(body)
		require.False(t, changed, string(body))
		require.Equal(t, body, got)
	}
	body := []byte(`{"input":[],"input":[{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"}]}`)
	got, changed := normalizeCodexPageContextBootstrap(body)
	require.False(t, changed)
	require.Equal(t, body, got)
}

func TestNormalizeCodexPageContextWithOtherBootstrapMessages(t *testing.T) {
	page := `{"type":"function_call_output","namespace":"agent","name":"codex_apps_open_page","output":"{\"page_id\":null}"}`
	delegation := `{"type":"function_call_output","namespace":"codex_app","name":"create_thread","output":` + mustJSON(t, delegationEnvelope) + `}`
	body := []byte(`{"input":[` + page + `,` + delegation + `]}`)
	got, changed := normalizeCodexPageContextBootstrap(body)
	require.True(t, changed)
	require.Equal(t, "message", gjson.GetBytes(got, "input.0.type").String())
	require.Equal(t, "message", gjson.GetBytes(got, "input.1.type").String())
	automation := `{"type":"function_call_output","namespace":"codex_app","name":"automation_update","output":` + mustJSON(t, codexAutomationBootstrap("wiki", "never", "Review changes")) + `}`
	body = []byte(`{"input":[` + page + `,` + automation + `]}`)
	got, changed = normalizeCodexPageContextBootstrap(body)
	require.True(t, changed)
	require.Equal(t, "message", gjson.GetBytes(got, "input.1.type").String())
	body = []byte(`{"input":[` + page + `,` + automation + `,{"type":"function_call","call_id":"call_1"}]}`)
	got, changed = normalizeCodexPageContextBootstrap(body)
	require.False(t, changed, "automation must retain its bootstrap-only boundary")
	require.Equal(t, body, got)
}
