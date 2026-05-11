package core

import (
	"path/filepath"
	"testing"
)

func TestChatProjectRouter_PersistAndClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat_project_routes.json")
	r := NewChatProjectRouter(path)

	if got := r.Get("feishu", "chat-a"); got != "" {
		t.Fatalf("Get() = %q, want empty", got)
	}
	if got := r.Claim("feishu", "chat-a", "my-claude"); got != "my-claude" {
		t.Fatalf("Claim() = %q, want my-claude", got)
	}
	// Claim should not override existing route.
	if got := r.Claim("feishu", "chat-a", "my-dify"); got != "my-claude" {
		t.Fatalf("second Claim() = %q, want my-claude", got)
	}

	reloaded := NewChatProjectRouter(path)
	if got := reloaded.Get("feishu", "chat-a"); got != "my-claude" {
		t.Fatalf("reloaded Get() = %q, want my-claude", got)
	}

	reloaded.Set("feishu", "chat-a", "my-dify")
	if got := reloaded.Get("feishu", "chat-a"); got != "my-dify" {
		t.Fatalf("Set/Get = %q, want my-dify", got)
	}

	reloaded.Clear("feishu", "chat-a")
	if got := reloaded.Get("feishu", "chat-a"); got != "" {
		t.Fatalf("Get after Clear() = %q, want empty", got)
	}
}
