package core

import (
	"path/filepath"
	"testing"
)

func TestEngineChatProjectRouting_SingleResponderAndSwitch(t *testing.T) {
	router := NewChatProjectRouter(filepath.Join(t.TempDir(), "chat_project_routes.json"))

	p1 := &stubPlatformEngine{n: "feishu"}
	p2 := &stubPlatformEngine{n: "feishu"}
	e1 := NewEngine("my-claude", &stubAgent{}, []Platform{p1}, "", LangEnglish)
	e2 := NewEngine("my-dify", &stubAgent{}, []Platform{p2}, "", LangEnglish)
	e1.SetChatProjectRouter(router)
	e2.SetChatProjectRouter(router)

	rm := NewRelayManager("")
	rm.RegisterEngine("my-claude", e1)
	rm.RegisterEngine("my-dify", e2)
	e1.SetRelayManager(rm)
	e2.SetRelayManager(rm)

	msg := &Message{
		SessionKey: "feishu:chat-1:user-1",
		Platform:   "feishu",
		Content:    "/status",
		ReplyCtx:   "ctx",
	}
	e1.ReceiveMessage(p1, msg)
	e2.ReceiveMessage(p2, msg)

	if got := len(p1.getSent()); got == 0 {
		t.Fatal("expected first project to reply")
	}
	if got := len(p2.getSent()); got != 0 {
		t.Fatalf("expected second project to be filtered, got %d replies", got)
	}
	if got := router.Get("feishu", "chat-1"); got != "my-claude" {
		t.Fatalf("initial route = %q, want my-claude", got)
	}

	p1.clearSent()
	p2.clearSent()

	switchMsg := &Message{
		SessionKey: "feishu:chat-1:user-1",
		Platform:   "feishu",
		Content:    "/project switch my-dify",
		ReplyCtx:   "ctx",
	}
	e1.ReceiveMessage(p1, switchMsg)
	e2.ReceiveMessage(p2, switchMsg)

	if got := router.Get("feishu", "chat-1"); got != "my-dify" {
		t.Fatalf("route after switch = %q, want my-dify", got)
	}

	p1.clearSent()
	p2.clearSent()

	e1.ReceiveMessage(p1, msg)
	e2.ReceiveMessage(p2, msg)

	if got := len(p1.getSent()); got != 0 {
		t.Fatalf("expected first project to be filtered after switch, got %d replies", got)
	}
	if got := len(p2.getSent()); got == 0 {
		t.Fatal("expected second project to reply after switch")
	}
}
