//go:build unit

package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type chatToolRoutingUpstream struct {
	service.HTTPUpstream
	t    *testing.T
	hits []int64
}

func (u *chatToolRoutingUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.hits = append(u.hits, accountID)
	body, err := io.ReadAll(req.Body)
	require.NoError(u.t, err)
	require.Equal(u.t, "/v1/responses", req.URL.Path)
	require.Contains(u.t, string(body), `"name":"get_test_value"`)
	events := "data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"function_call\",\"id\":\"fc_test\",\"call_id\":\"call_test\",\"name\":\"get_test_value\"}}\n\n" +
		"data: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"{}\"}\n\n"
	output := `[{"type":"function_call","id":"fc_test","call_id":"call_test","name":"get_test_value","arguments":"{}"}]`
	if len(u.hits) == 2 {
		require.Contains(u.t, string(body), `"type":"function_call_output"`)
		require.Contains(u.t, string(body), `"output":"test-value"`)
		events = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"received test-value\"}\n\n"
		output = `[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"received test-value"}]}]`
	}
	events += "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_test\",\"model\":\"gpt-6.2-sol\",\"status\":\"completed\",\"output\":" + output + ",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n"
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(events)),
	}, nil
}

func TestOpenAIChatCompletionsRoutesGPT62ToolsToResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const groupID int64 = 13
	accounts := []service.Account{
		{ID: 157, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 1,
			Credentials: map[string]any{"api_key": "test-chat-key"},
			Extra:       map[string]any{"openai_responses_supported": false}},
		{ID: 136, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: 2,
			Credentials: map[string]any{"api_key": "test-responses-key"},
			Extra:       map[string]any{"openai_responses_supported": true}},
	}
	repo := &grokCredentialHandlerRepo{accounts: accounts}
	upstream := &chatToolRoutingUpstream{t: t}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	billingCache := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingCache.Stop)
	var acquiredAccountIDs []int64
	cache := &concurrencyCacheMock{
		acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) { return true, nil },
		acquireAccountSlotFn: func(_ context.Context, accountID int64, _ int, _ string) (bool, error) {
			acquiredAccountIDs = append(acquiredAccountIDs, accountID)
			return true, nil
		},
	}
	concurrency := service.NewConcurrencyService(cache)
	gateway := service.NewOpenAIGatewayService(
		repo, nil, nil, nil, nil, nil, nil, cfg, nil, concurrency,
		service.NewBillingService(cfg, nil), nil, billingCache, upstream,
		&service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil,
	)
	handler := NewOpenAIGatewayHandler(gateway, concurrency, billingCache, &service.APIKeyService{}, nil, nil, nil, nil, cfg)
	keyGroupID := groupID
	apiKey := &service.APIKey{ID: 902, GroupID: &keyGroupID,
		User:  &service.User{ID: 903, Status: service.StatusActive},
		Group: &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}
	request := func(messages string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Set(string(middleware.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.User.ID, Concurrency: 1})
		body := `{"model":"gpt-6.2-sol","reasoning_effort":"medium","stream":true,"messages":` + messages +
			`,"tools":[{"type":"function","function":{"name":"get_test_value","parameters":{"type":"object","properties":{}}}}]}`
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
		c.Request.Header.Set("User-Agent", "WorkBuddy/5.11.3")
		handler.ChatCompletions(c)
		return recorder
	}
	first := request(`[{"role":"user","content":"Call get_test_value"}]`)
	require.Equal(t, http.StatusOK, first.Code, first.Body.String())
	require.Contains(t, first.Body.String(), `"finish_reason":"tool_calls"`)
	require.Contains(t, first.Body.String(), `"id":"call_test"`)
	second := request(`[{"role":"user","content":"Call get_test_value"},{"role":"assistant","tool_calls":[{"id":"call_test","type":"function","function":{"name":"get_test_value","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_test","content":"test-value"}]`)
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())
	require.Contains(t, second.Body.String(), "received test-value")
	require.Equal(t, []int64{136, 136}, upstream.hits)
	require.Equal(t, []int64{157, 136, 157, 136}, acquiredAccountIDs)
	require.Equal(t, int32(len(acquiredAccountIDs)), atomic.LoadInt32(&cache.releaseAccountCalled))
}
