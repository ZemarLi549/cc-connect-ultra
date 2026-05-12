package dify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ZemarLi549/cc-connect-ultra/core"
)

func TestSessionChatFlowSendUsesConversationID(t *testing.T) {
	var firstBody map[string]any
	var secondBody map[string]any
	call := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/info":
			_ = json.NewEncoder(w).Encode(map[string]any{"mode": "advanced-chat"})
		case r.Method == http.MethodPost && r.URL.Path == "/chat-messages":
			call++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if call == 1 {
				firstBody = body
			} else {
				secondBody = body
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"answer":          "hello",
				"conversation_id": "conv-123",
				"metadata": map[string]any{
					"usage": map[string]any{
						"prompt_tokens":     10,
						"completion_tokens": 5,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	s, err := newSession(context.Background(), runtimeConfig{
		baseURL:          srv.URL,
		apiKey:           "app-key",
		user:             "cc-connect:test",
		autoGenerateName: true,
		client:           srv.Client(),
	}, "")
	if err != nil {
		t.Fatalf("newSession() error = %v", err)
	}
	defer s.Close()

	if err := s.Send("first", nil, nil); err != nil {
		t.Fatalf("Send(first) error = %v", err)
	}
	waitForEventResult(t, s.events)

	if got := s.CurrentSessionID(); got != "conv-123" {
		t.Fatalf("CurrentSessionID() = %q, want conv-123", got)
	}
	if _, ok := firstBody["conversation_id"]; ok {
		t.Fatalf("first request unexpectedly sent conversation_id: %#v", firstBody["conversation_id"])
	}
	if inputs, ok := firstBody["inputs"].(map[string]any); !ok {
		t.Fatalf("first request inputs type = %T, want object", firstBody["inputs"])
	} else if len(inputs) != 0 {
		t.Fatalf("first request inputs = %#v, want empty object", inputs)
	}

	if err := s.Send("second", nil, nil); err != nil {
		t.Fatalf("Send(second) error = %v", err)
	}
	waitForEventResult(t, s.events)

	if got, _ := secondBody["conversation_id"].(string); got != "conv-123" {
		t.Fatalf("second request conversation_id = %q, want conv-123", got)
	}
	if inputs, ok := secondBody["inputs"].(map[string]any); !ok {
		t.Fatalf("second request inputs type = %T, want object", secondBody["inputs"])
	} else if len(inputs) != 0 {
		t.Fatalf("second request inputs = %#v, want empty object", inputs)
	}
}

func TestSessionWorkflowSendMapsPromptToInput(t *testing.T) {
	var requestBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/info":
			_ = json.NewEncoder(w).Encode(map[string]any{"mode": "workflow"})
		case r.Method == http.MethodGet && r.URL.Path == "/parameters":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"user_input_form": []any{
					map[string]any{
						"text-input": map[string]any{
							"variable": "question",
						},
					},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/workflows/run":
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"status":       "succeeded",
					"total_tokens": 42,
					"outputs": map[string]any{
						"result": "workflow answer",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	s, err := newSession(context.Background(), runtimeConfig{
		baseURL: srv.URL,
		apiKey:  "app-key",
		user:    "cc-connect:test",
		client:  srv.Client(),
		inputs: map[string]any{
			"tenant": "ops",
		},
	}, "")
	if err != nil {
		t.Fatalf("newSession() error = %v", err)
	}
	defer s.Close()

	if err := s.Send("where is host-a", nil, nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	ev := waitForEventResult(t, s.events)
	if ev.Content != "workflow answer" {
		t.Fatalf("EventResult content = %q, want workflow answer", ev.Content)
	}
	inputs, _ := requestBody["inputs"].(map[string]any)
	if got, _ := inputs["question"].(string); got != "where is host-a" {
		t.Fatalf("workflow input question = %q, want original prompt", got)
	}
	if got, _ := inputs["tenant"].(string); got != "ops" {
		t.Fatalf("workflow input tenant = %q, want ops", got)
	}
}

func TestListSessionsReturnsConversationSummaries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/info":
			_ = json.NewEncoder(w).Encode(map[string]any{"mode": "chat"})
		case r.Method == http.MethodGet && r.URL.Path == "/conversations":
			if got := r.URL.Query().Get("user"); got != "cc-connect:test" {
				t.Fatalf("user query = %q, want cc-connect:test", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []any{
					map[string]any{
						"id":           "conv-2",
						"name":         "second",
						"updated_at":   1705407630,
						"introduction": "",
					},
					map[string]any{
						"id":           "conv-1",
						"name":         "",
						"introduction": "first intro",
						"updated_at":   1705407629,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	a, err := New(map[string]any{
		"base_url": srv.URL,
		"api_key":  "app-key",
		"user":     "cc-connect:test",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	agent := a.(*Agent)
	agent.client = srv.Client()

	sessions, err := agent.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("ListSessions() len = %d, want 2", len(sessions))
	}
	if sessions[0].ID != "conv-2" || sessions[0].Summary != "second" {
		t.Fatalf("sessions[0] = %+v, want conv-2/second", sessions[0])
	}
	if sessions[1].Summary != "first intro" {
		t.Fatalf("sessions[1].Summary = %q, want first intro", sessions[1].Summary)
	}
}

func TestResolveRuntimeUserFromSessionKey(t *testing.T) {
	got := resolveRuntimeUser("cc-connect:demo", []string{"CC_SESSION_KEY=telegram:1:2"}, true, false)
	if got != "cc-connect:demo:telegram:1:2" {
		t.Fatalf("resolveRuntimeUser() = %q", got)
	}
}

func TestResolveRuntimeUserFromSessionUserID(t *testing.T) {
	got := resolveRuntimeUser("cc-connect:demo", []string{"CC_PLATFORM_USER_ID=u_test_user"}, false, true)
	if got != "u_test_user" {
		t.Fatalf("resolveRuntimeUser() = %q", got)
	}
}

func TestResolveRuntimeUserFromSessionUserIDFallsBackToSessionKey(t *testing.T) {
	got := resolveRuntimeUser("cc-connect:demo", []string{"CC_SESSION_KEY=feishu:oc_test:ou_user_1"}, false, true)
	if got != "ou_user_1" {
		t.Fatalf("resolveRuntimeUser() = %q", got)
	}
}

func TestResolveRuntimeUserFromSessionUserIDIgnoresThreadKeys(t *testing.T) {
	got := resolveRuntimeUser("cc-connect:demo", []string{"CC_SESSION_KEY=feishu:oc_test:root:om_root"}, false, true)
	if got != "cc-connect:demo" {
		t.Fatalf("resolveRuntimeUser() = %q", got)
	}
}

func TestAgentStartSessionUsesSessionUserIDForDifyUser(t *testing.T) {
	var requestBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/info":
			_ = json.NewEncoder(w).Encode(map[string]any{"mode": "advanced-chat"})
		case r.Method == http.MethodPost && r.URL.Path == "/chat-messages":
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"answer":          "hello",
				"conversation_id": "conv-123",
				"metadata": map[string]any{
					"usage": map[string]any{
						"prompt_tokens":     3,
						"completion_tokens": 2,
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	a, err := New(map[string]any{
		"base_url":                  srv.URL,
		"api_key":                   "app-key",
		"user":                      "cc-connect:demo",
		"user_from_session_user_id": true,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	agent := a.(*Agent)
	agent.client = srv.Client()
	agent.SetSessionEnv([]string{"CC_SESSION_KEY=feishu:oc_test:ou_user_1", "CC_PLATFORM_USER_ID=u_test_user"})

	sess, err := agent.StartSession(context.Background(), "")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	defer sess.Close()

	if err := sess.Send("hello", nil, nil); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	waitForEventResult(t, sess.Events())

	if got, _ := requestBody["user"].(string); got != "u_test_user" {
		t.Fatalf("request user = %q, want u_test_user", got)
	}
}

func TestWorkflowOutputsTextFallsBackToJSON(t *testing.T) {
	out := workflowOutputsText(map[string]any{
		"score": 95,
		"items": []string{"a", "b"},
	})
	if !strings.Contains(out, "\"score\": 95") {
		t.Fatalf("workflowOutputsText() = %q, want JSON fallback", out)
	}
}

func waitForEventResult(t *testing.T, ch <-chan core.Event) core.Event {
	t.Helper()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("events channel closed before EventResult")
			}
			if ev.Type == core.EventError {
				t.Fatalf("received EventError: %v", ev.Error)
			}
			if ev.Type == core.EventResult {
				return ev
			}
		case <-timeout.C:
			t.Fatal("timed out waiting for EventResult")
		}
	}
}
