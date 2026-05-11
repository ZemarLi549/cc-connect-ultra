package dify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ZemarLi549/cc-connect-ultra/core"
)

func init() {
	core.RegisterAgent("dify", New)
}

const defaultConversationLimit = 100

type Agent struct {
	project            string
	workDir            string
	baseURL            string
	apiKey             string
	appMode            string
	user               string
	queryInputKey      string
	workflowID         string
	autoGenerateName   bool
	timeout            time.Duration
	inputs             map[string]any
	providers          []core.ProviderConfig
	activeIdx          int
	sessionEnv         []string
	userFromSessionKey bool
	client             *http.Client
	mu                 sync.RWMutex
}

func New(opts map[string]any) (core.Agent, error) {
	workDir, _ := opts["work_dir"].(string)
	if strings.TrimSpace(workDir) == "" {
		workDir = "."
	}
	project, _ := opts["cc_project"].(string)
	user, _ := opts["user"].(string)
	if user == "" {
		if project != "" {
			user = "cc-connect:" + project
		} else {
			user = "cc-connect"
		}
	}

	baseURL, _ := opts["base_url"].(string)
	apiKey, _ := opts["api_key"].(string)
	appMode, _ := opts["app_mode"].(string)
	queryInputKey, _ := opts["query_input_key"].(string)
	workflowID, _ := opts["workflow_id"].(string)

	timeout := durationOpt(opts["timeout_secs"], 110*time.Second)

	return &Agent{
		project:            strings.TrimSpace(project),
		workDir:            workDir,
		baseURL:            normalizeBaseURL(baseURL),
		apiKey:             strings.TrimSpace(apiKey),
		appMode:            normalizeAppMode(appMode),
		user:               user,
		queryInputKey:      strings.TrimSpace(queryInputKey),
		workflowID:         strings.TrimSpace(workflowID),
		autoGenerateName:   boolOpt(opts["auto_generate_name"], true),
		timeout:            timeout,
		inputs:             cloneAnyMap(anyMapOpt(opts["inputs"])),
		activeIdx:          -1,
		userFromSessionKey: boolOpt(opts["user_from_session_key"], false),
		client:             &http.Client{Timeout: timeout},
	}, nil
}

func (a *Agent) Name() string { return "dify" }
func (a *Agent) Stop() error  { return nil }

func (a *Agent) SetWorkDir(dir string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.workDir = dir
	slog.Info("dify: work_dir changed", "work_dir", dir)
}

func (a *Agent) GetWorkDir() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.workDir
}

func (a *Agent) SetSessionEnv(env []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionEnv = append([]string(nil), env...)
}

func (a *Agent) SetProviders(providers []core.ProviderConfig) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.providers = providers
}

func (a *Agent) SetActiveProvider(name string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if name == "" {
		a.activeIdx = -1
		return true
	}
	for i, p := range a.providers {
		if p.Name == name {
			a.activeIdx = i
			return true
		}
	}
	return false
}

func (a *Agent) GetActiveProvider() *core.ProviderConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.activeIdx < 0 || a.activeIdx >= len(a.providers) {
		return nil
	}
	p := a.providers[a.activeIdx]
	return &p
}

func (a *Agent) ListProviders() []core.ProviderConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]core.ProviderConfig, len(a.providers))
	copy(out, a.providers)
	return out
}

func (a *Agent) StartSession(ctx context.Context, sessionID string) (core.AgentSession, error) {
	a.mu.RLock()
	cfg := runtimeConfig{
		project:          a.project,
		workDir:          a.workDir,
		baseURL:          a.baseURL,
		apiKey:           a.apiKey,
		appMode:          a.appMode,
		user:             a.user,
		queryInputKey:    a.queryInputKey,
		workflowID:       a.workflowID,
		autoGenerateName: a.autoGenerateName,
		inputs:           cloneAnyMap(a.inputs),
		timeout:          a.timeout,
		client:           a.client,
	}
	if a.activeIdx >= 0 && a.activeIdx < len(a.providers) {
		p := a.providers[a.activeIdx]
		if strings.TrimSpace(p.BaseURL) != "" {
			cfg.baseURL = normalizeBaseURL(p.BaseURL)
		}
		if strings.TrimSpace(p.APIKey) != "" {
			cfg.apiKey = strings.TrimSpace(p.APIKey)
		}
	}
	sessionEnv := append([]string(nil), a.sessionEnv...)
	useSessionUser := a.userFromSessionKey
	a.mu.RUnlock()

	cfg.user = resolveRuntimeUser(cfg.user, sessionEnv, useSessionUser)
	return newSession(ctx, cfg, sessionID)
}

func (a *Agent) ListSessions(ctx context.Context) ([]core.AgentSessionInfo, error) {
	cfg, err := a.runtimeConfigForReadOnly()
	if err != nil {
		return nil, err
	}
	mode, err := fetchAppMode(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if !isConversationMode(mode) {
		return nil, nil
	}
	resp, err := doJSON[difyConversationListResponse](ctx, cfg.client, cfg, http.MethodGet, "/conversations", map[string]string{
		"user":    cfg.user,
		"limit":   fmt.Sprintf("%d", defaultConversationLimit),
		"sort_by": "-updated_at",
	}, nil)
	if err != nil {
		return nil, err
	}
	out := make([]core.AgentSessionInfo, 0, len(resp.Data))
	for _, item := range resp.Data {
		summary := strings.TrimSpace(item.Name)
		if summary == "" {
			summary = strings.TrimSpace(item.Introduction)
		}
		out = append(out, core.AgentSessionInfo{
			ID:         item.ID,
			Summary:    summary,
			ModifiedAt: unixOrNow(item.UpdatedAt),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModifiedAt.After(out[j].ModifiedAt) })
	return out, nil
}

func (a *Agent) DeleteSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	cfg, err := a.runtimeConfigForReadOnly()
	if err != nil {
		return err
	}
	mode, err := fetchAppMode(ctx, cfg)
	if err != nil {
		return err
	}
	if !isConversationMode(mode) {
		return fmt.Errorf("dify: delete session is only available for chat apps")
	}
	return doNoContent(ctx, cfg.client, cfg, http.MethodDelete, "/conversations/"+url.PathEscape(sessionID), nil, map[string]any{
		"user": cfg.user,
	})
}

func (a *Agent) GetSessionHistory(ctx context.Context, sessionID string, limit int) ([]core.HistoryEntry, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	cfg, err := a.runtimeConfigForReadOnly()
	if err != nil {
		return nil, err
	}
	mode, err := fetchAppMode(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if !isConversationMode(mode) {
		return nil, fmt.Errorf("dify: history is only available for chat apps")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	resp, err := doJSON[difyMessageListResponse](ctx, cfg.client, cfg, http.MethodGet, "/messages", map[string]string{
		"conversation_id": sessionID,
		"user":            cfg.user,
		"limit":           fmt.Sprintf("%d", limit),
	}, nil)
	if err != nil {
		return nil, err
	}
	data := append([]difyMessageRecord(nil), resp.Data...)
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
	var out []core.HistoryEntry
	for _, item := range data {
		ts := unixOrNow(item.CreatedAt)
		if q := strings.TrimSpace(item.Query); q != "" {
			out = append(out, core.HistoryEntry{Role: "user", Content: q, Timestamp: ts})
		}
		if ans := strings.TrimSpace(item.Answer); ans != "" {
			out = append(out, core.HistoryEntry{Role: "assistant", Content: ans, Timestamp: ts})
		}
	}
	return out, nil
}

func (a *Agent) runtimeConfigForReadOnly() (runtimeConfig, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.userFromSessionKey {
		return runtimeConfig{}, fmt.Errorf("dify: list/history/delete are unavailable when user_from_session_key = true")
	}
	cfg := runtimeConfig{
		project:          a.project,
		baseURL:          a.baseURL,
		apiKey:           a.apiKey,
		appMode:          a.appMode,
		user:             a.user,
		queryInputKey:    a.queryInputKey,
		workflowID:       a.workflowID,
		autoGenerateName: a.autoGenerateName,
		inputs:           cloneAnyMap(a.inputs),
		timeout:          a.timeout,
		client:           a.client,
	}
	if a.activeIdx >= 0 && a.activeIdx < len(a.providers) {
		p := a.providers[a.activeIdx]
		if strings.TrimSpace(p.BaseURL) != "" {
			cfg.baseURL = normalizeBaseURL(p.BaseURL)
		}
		if strings.TrimSpace(p.APIKey) != "" {
			cfg.apiKey = strings.TrimSpace(p.APIKey)
		}
	}
	if cfg.baseURL == "" || cfg.apiKey == "" {
		return runtimeConfig{}, requiredDifyConfigError(cfg.project)
	}
	return cfg, nil
}

type runtimeConfig struct {
	project          string
	workDir          string
	baseURL          string
	apiKey           string
	appMode          string
	user             string
	queryInputKey    string
	workflowID       string
	autoGenerateName bool
	inputs           map[string]any
	timeout          time.Duration
	client           *http.Client
}

type difyAppInfo struct {
	Mode string `json:"mode"`
}

type difyParametersResponse struct {
	UserInputForm []map[string]struct {
		Variable string `json:"variable"`
	} `json:"user_input_form"`
}

type difyConversationListResponse struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Introduction string `json:"introduction"`
		UpdatedAt    int64  `json:"updated_at"`
	} `json:"data"`
}

type difyMessageListResponse struct {
	Data []difyMessageRecord `json:"data"`
}

type difyMessageRecord struct {
	Query     string `json:"query"`
	Answer    string `json:"answer"`
	CreatedAt int64  `json:"created_at"`
}

type difyBlockingResponse struct {
	ConversationID string `json:"conversation_id"`
	Answer         string `json:"answer"`
	Metadata       struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	} `json:"metadata"`
}

type difyWorkflowResponse struct {
	Data struct {
		Status      string         `json:"status"`
		Outputs     map[string]any `json:"outputs"`
		Error       any            `json:"error"`
		TotalTokens int            `json:"total_tokens"`
	} `json:"data"`
}

type difyAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func fetchAppMode(ctx context.Context, cfg runtimeConfig) (string, error) {
	if cfg.appMode != "" {
		return cfg.appMode, nil
	}
	info, err := doJSON[difyAppInfo](ctx, cfg.client, cfg, http.MethodGet, "/info", nil, nil)
	if err != nil {
		return "", err
	}
	mode := normalizeAppMode(info.Mode)
	if mode == "" {
		return "", fmt.Errorf("dify: app mode missing in /info response")
	}
	return mode, nil
}

func fetchQueryInputKey(ctx context.Context, cfg runtimeConfig) (string, error) {
	if cfg.queryInputKey != "" {
		return cfg.queryInputKey, nil
	}
	params, err := doJSON[difyParametersResponse](ctx, cfg.client, cfg, http.MethodGet, "/parameters", nil, nil)
	if err != nil {
		return "", err
	}
	var vars []string
	for _, entry := range params.UserInputForm {
		for _, item := range entry {
			if v := strings.TrimSpace(item.Variable); v != "" {
				vars = append(vars, v)
			}
		}
	}
	if len(vars) == 1 {
		return vars[0], nil
	}
	return "", nil
}

func doJSON[T any](ctx context.Context, client *http.Client, cfg runtimeConfig, method, path string, query map[string]string, body any) (T, error) {
	var zero T
	if strings.TrimSpace(cfg.baseURL) == "" || strings.TrimSpace(cfg.apiKey) == "" {
		return zero, requiredDifyConfigError(cfg.project)
	}
	reqURL, err := buildURL(cfg.baseURL, path, query)
	if err != nil {
		return zero, err
	}
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("dify: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return zero, fmt.Errorf("dify: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("dify: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, decodeAPIError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(&zero); err != nil {
		return zero, fmt.Errorf("dify: decode response: %w", err)
	}
	return zero, nil
}

func doNoContent(ctx context.Context, client *http.Client, cfg runtimeConfig, method, path string, query map[string]string, body any) error {
	if strings.TrimSpace(cfg.baseURL) == "" || strings.TrimSpace(cfg.apiKey) == "" {
		return requiredDifyConfigError(cfg.project)
	}
	reqURL, err := buildURL(cfg.baseURL, path, query)
	if err != nil {
		return err
	}
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("dify: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("dify: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("dify: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}
	return nil
}

func decodeAPIError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	var apiErr difyAPIError
	if err := json.Unmarshal(data, &apiErr); err == nil && (apiErr.Message != "" || apiErr.Code != "") {
		if apiErr.Code != "" {
			return fmt.Errorf("dify: %s (%s)", apiErr.Message, apiErr.Code)
		}
		return fmt.Errorf("dify: %s", apiErr.Message)
	}
	msg := strings.TrimSpace(string(data))
	if msg == "" {
		msg = resp.Status
	}
	return fmt.Errorf("dify: %s", msg)
}

func requiredDifyConfigError(projectName string) error {
	pn := strings.TrimSpace(projectName)
	if pn == "" {
		return fmt.Errorf("dify: base_url and api_key are required; if config was updated, restart service and create a new session")
	}
	return fmt.Errorf("dify: base_url and api_key are required (project=%s); if config was updated, restart service and create a new session", pn)
}

func buildURL(baseURL, path string, query map[string]string) (string, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return "", fmt.Errorf("dify: parse URL: %w", err)
	}
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			if strings.TrimSpace(v) != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func resolveRuntimeUser(base string, env []string, fromSession bool) string {
	if !fromSession {
		return base
	}
	for _, item := range env {
		k, v, ok := strings.Cut(item, "=")
		if ok && k == "CC_SESSION_KEY" && strings.TrimSpace(v) != "" {
			return base + ":" + v
		}
	}
	return base
}

func isConversationMode(mode string) bool {
	switch normalizeAppMode(mode) {
	case "chat", "advanced-chat":
		return true
	default:
		return false
	}
}

func normalizeAppMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "chat":
		return "chat"
	case "advanced-chat", "advanced_chat", "chatflow":
		return "advanced-chat"
	case "agent-chat", "agent_chat":
		return "agent-chat"
	case "completion":
		return "completion"
	case "workflow":
		return "workflow"
	default:
		return ""
	}
}

func normalizeBaseURL(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func unixOrNow(ts int64) time.Time {
	if ts <= 0 {
		return time.Now()
	}
	return time.Unix(ts, 0)
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func anyMapOpt(v any) map[string]any {
	switch m := v.(type) {
	case nil:
		return nil
	case map[string]any:
		return m
	case map[string]string:
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	default:
		return nil
	}
}

func boolOpt(v any, fallback bool) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		switch strings.ToLower(strings.TrimSpace(b)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

func durationOpt(v any, fallback time.Duration) time.Duration {
	switch n := v.(type) {
	case int:
		if n > 0 {
			return time.Duration(n) * time.Second
		}
	case int64:
		if n > 0 {
			return time.Duration(n) * time.Second
		}
	case float64:
		if n > 0 {
			return time.Duration(n * float64(time.Second))
		}
	}
	return fallback
}

var (
	_ core.WorkDirSwitcher    = (*Agent)(nil)
	_ core.SessionEnvInjector = (*Agent)(nil)
	_ core.ProviderSwitcher   = (*Agent)(nil)
	_ core.SessionDeleter     = (*Agent)(nil)
	_ core.HistoryProvider    = (*Agent)(nil)
)
