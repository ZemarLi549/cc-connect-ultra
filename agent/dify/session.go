package dify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ZemarLi549/cc-connect-ultra/core"
)

type Session struct {
	cfg              runtimeConfig
	ctx              context.Context
	cancel           context.CancelFunc
	events           chan core.Event
	busy             atomic.Bool
	alive            atomic.Bool
	modeMu           sync.Mutex
	resolvedMode     string
	resolvedInputKey string
	conversationID   atomic.Value // string
}

func newSession(ctx context.Context, cfg runtimeConfig, sessionID string) (*Session, error) {
	sessionCtx, cancel := context.WithCancel(ctx)
	s := &Session{
		cfg:    cfg,
		ctx:    sessionCtx,
		cancel: cancel,
		events: make(chan core.Event, 32),
	}
	s.alive.Store(true)
	if sessionID != "" && sessionID != core.ContinueSession {
		s.conversationID.Store(sessionID)
	}
	return s, nil
}

func (s *Session) Send(prompt string, images []core.ImageAttachment, files []core.FileAttachment) error {
	if !s.alive.Load() {
		return fmt.Errorf("session is closed")
	}
	if !s.busy.CompareAndSwap(false, true) {
		return fmt.Errorf("session is busy")
	}
	go s.run(prompt, images, files)
	return nil
}

func (s *Session) run(prompt string, images []core.ImageAttachment, files []core.FileAttachment) {
	defer s.busy.Store(false)

	mode, queryInputKey, err := s.resolveMode()
	if err != nil {
		s.emitError(err)
		return
	}

	if mode == "agent-chat" {
		s.emitError(fmt.Errorf("dify: agent-chat apps are not supported in blocking mode yet; use advanced-chat/chat/workflow/completion apps instead"))
		return
	}

	if len(images) > 0 || len(files) > 0 {
		prompt += fmt.Sprintf("\n\n[cc-connect notice: omitted %d image(s) and %d file(s) because the dify agent currently sends text-only API requests.]", len(images), len(files))
	}

	switch mode {
	case "chat", "advanced-chat":
		resp, err := s.sendChat(prompt)
		if err != nil {
			s.emitError(err)
			return
		}
		s.emitResult(resp.Answer, resp.ConversationID, resp.Metadata.Usage.PromptTokens, resp.Metadata.Usage.CompletionTokens)
	case "completion":
		resp, err := s.sendCompletion(prompt, queryInputKey)
		if err != nil {
			s.emitError(err)
			return
		}
		s.emitResult(resp.Answer, "", resp.Metadata.Usage.PromptTokens, resp.Metadata.Usage.CompletionTokens)
	case "workflow":
		answer, tokens, err := s.sendWorkflow(prompt, queryInputKey)
		if err != nil {
			s.emitError(err)
			return
		}
		s.emitResult(answer, "", tokens, 0)
	default:
		s.emitError(fmt.Errorf("dify: unsupported app mode %q", mode))
	}
}

func (s *Session) resolveMode() (string, string, error) {
	s.modeMu.Lock()
	defer s.modeMu.Unlock()

	if s.resolvedMode == "" {
		mode, err := fetchAppMode(s.ctx, s.cfg)
		if err != nil {
			return "", "", err
		}
		s.resolvedMode = mode
	}

	if s.resolvedMode == "workflow" || s.resolvedMode == "completion" {
		if s.resolvedInputKey == "" {
			key := strings.TrimSpace(s.cfg.queryInputKey)
			if key == "" {
				autoKey, err := fetchQueryInputKey(s.ctx, s.cfg)
				if err != nil {
					return "", "", err
				}
				key = autoKey
			}
			s.resolvedInputKey = strings.TrimSpace(key)
		}
	}

	return s.resolvedMode, s.resolvedInputKey, nil
}

func (s *Session) sendChat(prompt string) (difyBlockingResponse, error) {
	inputs := normalizeInputs(s.cfg.inputs)
	body := map[string]any{
		"inputs":             inputs,
		"query":              prompt,
		"response_mode":      "blocking",
		"user":               s.cfg.user,
		"auto_generate_name": s.cfg.autoGenerateName,
	}
	if sid := s.CurrentSessionID(); sid != "" {
		body["conversation_id"] = sid
	}
	if s.cfg.workflowID != "" {
		body["workflow_id"] = s.cfg.workflowID
	}
	resp, err := doJSON[difyBlockingResponse](s.ctx, s.cfg.client, s.cfg, "POST", "/chat-messages", nil, body)
	if err != nil {
		return difyBlockingResponse{}, err
	}
	if resp.ConversationID != "" {
		s.conversationID.Store(resp.ConversationID)
	}
	return resp, nil
}

func (s *Session) sendCompletion(prompt, queryInputKey string) (difyBlockingResponse, error) {
	inputs := normalizeInputs(s.cfg.inputs)
	if queryInputKey != "" {
		inputs[queryInputKey] = prompt
	}
	return doJSON[difyBlockingResponse](s.ctx, s.cfg.client, s.cfg, "POST", "/completion-messages", nil, map[string]any{
		"inputs":        inputs,
		"query":         prompt,
		"response_mode": "blocking",
		"user":          s.cfg.user,
	})
}

func (s *Session) sendWorkflow(prompt, queryInputKey string) (string, int, error) {
	inputs := normalizeInputs(s.cfg.inputs)
	if queryInputKey != "" {
		inputs[queryInputKey] = prompt
	} else if strings.TrimSpace(prompt) != "" {
		inputs["query"] = prompt
	}
	resp, err := doJSON[difyWorkflowResponse](s.ctx, s.cfg.client, s.cfg, "POST", "/workflows/run", nil, map[string]any{
		"inputs":        inputs,
		"response_mode": "blocking",
		"user":          s.cfg.user,
	})
	if err != nil {
		return "", 0, err
	}
	if resp.Data.Status != "" && resp.Data.Status != "succeeded" {
		return "", resp.Data.TotalTokens, fmt.Errorf("dify: workflow finished with status %q", resp.Data.Status)
	}
	if resp.Data.Error != nil {
		return "", resp.Data.TotalTokens, fmt.Errorf("dify: workflow error: %v", resp.Data.Error)
	}
	return workflowOutputsText(resp.Data.Outputs), resp.Data.TotalTokens, nil
}

func normalizeInputs(in map[string]any) map[string]any {
	out := cloneAnyMap(in)
	if out == nil {
		return map[string]any{}
	}
	return out
}

func workflowOutputsText(outputs map[string]any) string {
	if len(outputs) == 0 {
		return ""
	}
	for _, key := range []string{"result", "answer", "text", "output"} {
		if v, ok := outputs[key]; ok {
			if s := strings.TrimSpace(stringifyValue(v)); s != "" {
				return s
			}
		}
	}
	if len(outputs) == 1 {
		for _, v := range outputs {
			if s := strings.TrimSpace(stringifyValue(v)); s != "" {
				return s
			}
		}
	}
	data, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		return stringifyValue(outputs)
	}
	return string(data)
}

func stringifyValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		data, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		return string(data)
	}
}

func (s *Session) RespondPermission(_ string, _ core.PermissionResult) error { return nil }
func (s *Session) Events() <-chan core.Event                                 { return s.events }

func (s *Session) CurrentSessionID() string {
	if v, ok := s.conversationID.Load().(string); ok {
		return v
	}
	return ""
}

func (s *Session) Alive() bool { return s.alive.Load() }

func (s *Session) Close() error {
	if !s.alive.CompareAndSwap(true, false) {
		return nil
	}
	s.cancel()
	return nil
}

func (s *Session) emitResult(content, sessionID string, inputTokens, outputTokens int) {
	if sessionID != "" {
		s.conversationID.Store(sessionID)
	}
	s.emit(core.Event{
		Type:         core.EventResult,
		Content:      content,
		SessionID:    sessionID,
		Done:         true,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	})
}

func (s *Session) emitError(err error) {
	s.emit(core.Event{Type: core.EventError, Error: err, Done: true})
}

func (s *Session) emit(evt core.Event) {
	select {
	case s.events <- evt:
	case <-s.ctx.Done():
	}
}

var _ core.AgentSession = (*Session)(nil)
