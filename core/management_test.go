package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

type deadlineAwareModelAgent struct {
	stubModelModeAgent
	mu          sync.Mutex
	hasDeadline bool
}

func (a *deadlineAwareModelAgent) AvailableModels(ctx context.Context) []ModelOption {
	a.mu.Lock()
	_, ok := ctx.Deadline()
	a.hasDeadline = ok
	a.mu.Unlock()
	return []ModelOption{{Name: "gpt-4.1"}}
}

func (a *deadlineAwareModelAgent) sawDeadline() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.hasDeadline
}

// testManagementServer creates a ManagementServer with a test engine and returns an httptest.Server.
func testManagementServer(t *testing.T, token string) (*ManagementServer, *httptest.Server, *Engine) {
	t.Helper()

	agent := &stubAgent{}
	sm := NewSessionManager("")
	e := NewEngine("test-project", agent, nil, "", LangEnglish)
	e.sessions = sm

	mgmt := NewManagementServer(0, token, nil)
	mgmt.RegisterEngine("test-project", e)

	mux := http.NewServeMux()
	prefix := "/api/v1"
	mux.HandleFunc(prefix+"/status", mgmt.wrap(mgmt.handleStatus))
	mux.HandleFunc(prefix+"/restart", mgmt.wrap(mgmt.handleRestart))
	mux.HandleFunc(prefix+"/reload", mgmt.wrap(mgmt.handleReload))
	mux.HandleFunc(prefix+"/config", mgmt.wrap(mgmt.handleConfig))
	mux.HandleFunc(prefix+"/settings", mgmt.wrap(mgmt.handleGlobalSettings))
	mux.HandleFunc(prefix+"/agents", mgmt.wrap(mgmt.handleAgents))
	mux.HandleFunc(prefix+"/projects", mgmt.wrap(mgmt.handleProjects))
	mux.HandleFunc(prefix+"/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	mux.HandleFunc(prefix+"/cron", mgmt.wrap(mgmt.handleCron))
	mux.HandleFunc(prefix+"/cron/", mgmt.wrap(mgmt.handleCronByID))
	mux.HandleFunc(prefix+"/providers", mgmt.wrap(mgmt.handleGlobalProviders))
	mux.HandleFunc(prefix+"/providers/", mgmt.wrap(mgmt.handleGlobalProviderRoutes))
	mux.HandleFunc(prefix+"/skills", mgmt.wrap(mgmt.handleSkills))
	mux.HandleFunc(prefix+"/skills/presets", mgmt.wrap(mgmt.handleSkillPresets))
	mux.HandleFunc(prefix+"/enterprise/overview", mgmt.wrap(mgmt.handleEnterpriseOverview))
	mux.HandleFunc(prefix+"/enterprise/tenants", mgmt.wrap(mgmt.handleEnterpriseTenants))
	mux.HandleFunc(prefix+"/enterprise/users", mgmt.wrap(mgmt.handleEnterpriseUsers))
	mux.HandleFunc(prefix+"/enterprise/spaces", mgmt.wrap(mgmt.handleEnterpriseSpaces))
	mux.HandleFunc(prefix+"/enterprise/skills", mgmt.wrap(mgmt.handleEnterpriseSkills))
	mux.HandleFunc(prefix+"/enterprise/bots", mgmt.wrap(mgmt.handleEnterpriseBots))
	mux.HandleFunc(prefix+"/enterprise/providers", mgmt.wrap(mgmt.handleEnterpriseProviders))
	mux.HandleFunc(prefix+"/enterprise/permissions", mgmt.wrap(mgmt.handleEnterprisePermissions))
	mux.HandleFunc(prefix+"/enterprise/roles", mgmt.wrap(mgmt.handleEnterpriseRoles))
	mux.HandleFunc(prefix+"/enterprise/role-bindings", mgmt.wrap(mgmt.handleEnterpriseRoleBindings))
	mux.HandleFunc(prefix+"/enterprise/effective-access", mgmt.wrap(mgmt.handleEnterpriseEffectiveAccess))
	mux.HandleFunc(prefix+"/enterprise/projects", mgmt.wrap(mgmt.handleEnterpriseProjects))
	mux.HandleFunc(prefix+"/enterprise/tasks", mgmt.wrap(mgmt.handleEnterpriseTasks))
	mux.HandleFunc(prefix+"/enterprise/settings", mgmt.wrap(mgmt.handleEnterpriseSettings))
	mux.HandleFunc(prefix+"/enterprise/imports", mgmt.wrap(mgmt.handleEnterpriseImports))
	mux.HandleFunc(prefix+"/enterprise/usage", mgmt.wrap(mgmt.handleEnterpriseUsage))
	mux.HandleFunc(prefix+"/enterprise/leaderboard", mgmt.wrap(mgmt.handleEnterpriseLeaderboard))
	mux.HandleFunc(prefix+"/bridge/adapters", mgmt.wrap(mgmt.handleBridgeAdapters))

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return mgmt, ts, e
}

type mgmtResponse struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

func mgmtGet(t *testing.T, url, token string) mgmtResponse {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	return r
}

func mgmtPost(t *testing.T, url, token string, body any) mgmtResponse {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode POST body: %v", err)
		}
	}
	req, _ := http.NewRequest("POST", url, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	return r
}

func mgmtPatch(t *testing.T, url, token string, body any) mgmtResponse {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode PATCH body: %v", err)
		}
	}
	req, _ := http.NewRequest("PATCH", url, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", url, err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode PATCH response: %v", err)
	}
	return r
}

func mgmtDelete(t *testing.T, url, token string) mgmtResponse {
	t.Helper()
	req, _ := http.NewRequest("DELETE", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", url, err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode DELETE response: %v", err)
	}
	return r
}

func TestMgmt_AuthRequired(t *testing.T) {
	_, ts, _ := testManagementServer(t, "secret-token")

	r := mgmtGet(t, ts.URL+"/api/v1/status", "")
	if r.OK {
		t.Fatal("expected auth failure without token")
	}
	if !strings.Contains(r.Error, "unauthorized") {
		t.Fatalf("expected unauthorized error, got: %s", r.Error)
	}

	r = mgmtGet(t, ts.URL+"/api/v1/status", "wrong-token")
	if r.OK {
		t.Fatal("expected auth failure with wrong token")
	}

	r = mgmtGet(t, ts.URL+"/api/v1/status", "secret-token")
	if !r.OK {
		t.Fatalf("expected success with correct token, got error: %s", r.Error)
	}
}

func TestMgmt_AuthQueryParam(t *testing.T) {
	_, ts, _ := testManagementServer(t, "qp-token")

	r := mgmtGet(t, ts.URL+"/api/v1/status?token=qp-token", "")
	if !r.OK {
		t.Fatalf("expected success with query param token, got: %s", r.Error)
	}
}

func TestMgmt_NoAuthRequired(t *testing.T) {
	_, ts, _ := testManagementServer(t, "")

	r := mgmtGet(t, ts.URL+"/api/v1/status", "")
	if !r.OK {
		t.Fatalf("expected success without token when no token configured, got: %s", r.Error)
	}
}

func TestMgmt_Status(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/status", "tok")
	if !r.OK {
		t.Fatalf("status failed: %s", r.Error)
	}

	var data map[string]any
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal status data: %v", err)
	}
	if data["projects_count"] != float64(1) {
		t.Fatalf("expected 1 project, got %v", data["projects_count"])
	}
}

func TestMgmt_StatusIncludesBridgeToken(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetBridgeServer(NewBridgeServer(9810, "bridge-secret", "/bridge/ws", nil))

	r := mgmtGet(t, ts.URL+"/api/v1/status", "tok")
	if !r.OK {
		t.Fatalf("status failed: %s", r.Error)
	}

	var data struct {
		Bridge struct {
			Enabled bool   `json:"enabled"`
			Port    int    `json:"port"`
			Path    string `json:"path"`
			Token   string `json:"token"`
		} `json:"bridge"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal status data: %v", err)
	}
	if !data.Bridge.Enabled {
		t.Fatal("expected bridge to be enabled")
	}
	if data.Bridge.Token != "bridge-secret" {
		t.Fatalf("expected bridge token, got %q", data.Bridge.Token)
	}
}

func TestMgmt_Projects(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/projects", "tok")
	if !r.OK {
		t.Fatalf("projects failed: %s", r.Error)
	}

	var data struct {
		Projects []map[string]any `json:"projects"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal projects data: %v", err)
	}
	if len(data.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(data.Projects))
	}
	if data.Projects[0]["name"] != "test-project" {
		t.Fatalf("expected test-project, got %v", data.Projects[0]["name"])
	}
}

func TestMgmt_ProjectDetail(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project", "tok")
	if !r.OK {
		t.Fatalf("project detail failed: %s", r.Error)
	}

	var data map[string]any
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal project detail: %v", err)
	}
	if data["name"] != "test-project" {
		t.Fatalf("expected test-project, got %v", data["name"])
	}

	r = mgmtGet(t, ts.URL+"/api/v1/projects/nonexistent", "tok")
	if r.OK {
		t.Fatal("expected 404 for nonexistent project")
	}
}

func TestMgmt_ProjectPatch(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project", "tok", map[string]any{
		"language": "zh",
	})
	if !r.OK {
		t.Fatalf("patch failed: %s", r.Error)
	}
}

func TestMgmt_ProjectPatch_DynamicOptionsPersistAndRequireRestart(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	var got ProjectSettingsUpdate
	mgmt.SetSaveProjectSettings(func(_ string, u ProjectSettingsUpdate) error {
		got = u
		return nil
	})

	r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project", "tok", map[string]any{
		"agent_options": map[string]any{
			"work_dir": "D:\\ai_study",
			"mode":     "default",
		},
		"platform_option_updates": []map[string]any{
			{
				"index": 0,
				"options": map[string]any{
					"app_id":      "cli_xxx",
					"allow_from":  "*",
					"group_reply": true,
				},
			},
		},
		"remove_platform_indexes": []int{2},
	})
	if !r.OK {
		t.Fatalf("patch failed: %s", r.Error)
	}

	var data map[string]any
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal response data: %v", err)
	}
	if data["restart_required"] != true {
		t.Fatalf("restart_required = %#v, want true", data["restart_required"])
	}
	if got.AgentOptions["mode"] != "default" {
		t.Fatalf("agent_options.mode = %#v, want default", got.AgentOptions["mode"])
	}
	if len(got.PlatformOptionUpdates) != 1 || got.PlatformOptionUpdates[0].Index != 0 {
		t.Fatalf("platform_option_updates = %#v", got.PlatformOptionUpdates)
	}
	if len(got.RemovePlatformIndexes) != 1 || got.RemovePlatformIndexes[0] != 2 {
		t.Fatalf("remove_platform_indexes = %#v, want [2]", got.RemovePlatformIndexes)
	}
}

func TestMgmt_Sessions(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")

	e.sessions.GetOrCreateActive("user1")

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/sessions", "tok")
	if !r.OK {
		t.Fatalf("sessions list failed: %s", r.Error)
	}

	// Create a session via API
	r = mgmtPost(t, ts.URL+"/api/v1/projects/test-project/sessions", "tok", map[string]string{
		"session_key": "user2",
		"name":        "work",
	})
	if !r.OK {
		t.Fatalf("create session failed: %s", r.Error)
	}
}

func TestMgmt_SessionDetail(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")

	s := e.sessions.GetOrCreateActive("user1")
	s.AddHistory("user", "hello")
	s.AddHistory("assistant", "hi there")

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/sessions/"+s.ID, "tok")
	if !r.OK {
		t.Fatalf("session detail failed: %s", r.Error)
	}

	var data struct {
		History []map[string]any `json:"history"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal session detail: %v", err)
	}
	if len(data.History) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(data.History))
	}
}

func TestMgmt_SessionDelete(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")

	s := e.sessions.GetOrCreateActive("user1")
	sid := s.ID

	r := mgmtDelete(t, ts.URL+"/api/v1/projects/test-project/sessions/"+sid, "tok")
	if !r.OK {
		t.Fatalf("delete session failed: %s", r.Error)
	}

	r = mgmtGet(t, ts.URL+"/api/v1/projects/test-project/sessions/"+sid, "tok")
	if r.OK {
		t.Fatal("expected 404 after deletion")
	}
}

func TestMgmt_Config(t *testing.T) {
	srv, ts, _ := testManagementServer(t, "tok")

	// Write a temp TOML file and point the server at it
	tmp := t.TempDir()
	cfgPath := tmp + "/config.toml"
	if err := os.WriteFile(cfgPath, []byte("[display]\ntitle = \"test\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srv.SetConfigFilePath(cfgPath)

	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/config", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "title") {
		t.Fatalf("expected TOML content, got: %s", body)
	}
}

func TestMgmt_Reload(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")

	reloaded := false
	e.configReloadFunc = func() (*ConfigReloadResult, error) {
		reloaded = true
		return &ConfigReloadResult{}, nil
	}

	r := mgmtPost(t, ts.URL+"/api/v1/reload", "tok", nil)
	if !r.OK {
		t.Fatalf("reload failed: %s", r.Error)
	}
	if !reloaded {
		t.Fatal("expected config reload to be triggered")
	}
}

func TestMgmt_BridgeAdapters(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/bridge/adapters", "tok")
	if !r.OK {
		t.Fatalf("bridge adapters failed: %s", r.Error)
	}
}

func TestMgmt_HeartbeatNotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/heartbeat", "tok")
	if r.OK {
		var data map[string]any
		if err := json.Unmarshal(r.Data, &data); err != nil {
			t.Fatalf("unmarshal heartbeat data: %v", err)
		}
		// heartbeat scheduler is nil, so we expect service unavailable
	}
}

func TestMgmt_HeartbeatWithScheduler(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	hs := NewHeartbeatScheduler("")
	mgmt.SetHeartbeatScheduler(hs)

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/heartbeat", "tok")
	if !r.OK {
		t.Fatalf("heartbeat status failed: %s", r.Error)
	}

	var data map[string]any
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal heartbeat status: %v", err)
	}
	if data["enabled"] != false {
		t.Fatalf("expected heartbeat disabled, got %v", data["enabled"])
	}
}

func TestMgmt_CronNilScheduler(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/cron", "tok")
	if r.OK {
		t.Fatal("expected error when cron scheduler is nil")
	}
}

func TestMgmt_CronWithScheduler(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, err := NewCronStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	// List (empty)
	r := mgmtGet(t, ts.URL+"/api/v1/cron", "tok")
	if !r.OK {
		t.Fatalf("cron list failed: %s", r.Error)
	}

	// Add
	r = mgmtPost(t, ts.URL+"/api/v1/cron", "tok", map[string]any{
		"project":     "test-project",
		"session_key": "user1",
		"cron_expr":   "0 9 * * *",
		"prompt":      "hello",
		"description": "test cron",
	})
	if !r.OK {
		t.Fatalf("cron add failed: %s", r.Error)
	}

	var job CronJob
	if err := json.Unmarshal(r.Data, &job); err != nil {
		t.Fatalf("unmarshal cron job: %v", err)
	}
	if job.ID == "" {
		t.Fatal("expected cron job ID")
	}

	// List (should have 1)
	r = mgmtGet(t, ts.URL+"/api/v1/cron", "tok")
	if !r.OK {
		t.Fatalf("cron list failed: %s", r.Error)
	}

	// Delete
	r = mgmtDelete(t, ts.URL+"/api/v1/cron/"+job.ID, "tok")
	if !r.OK {
		t.Fatalf("cron delete failed: %s", r.Error)
	}

	// Delete nonexistent
	r = mgmtDelete(t, ts.URL+"/api/v1/cron/nonexistent", "tok")
	if r.OK {
		t.Fatal("expected 404 for nonexistent cron job")
	}
}

func TestMgmt_CORS(t *testing.T) {
	mgmt := NewManagementServer(0, "", []string{"http://localhost:3000"})
	mgmt.RegisterEngine("p", NewEngine("p", &stubAgent{}, nil, "", LangEnglish))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/status", mgmt.wrap(mgmt.handleStatus))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	req, _ := http.NewRequest("OPTIONS", ts.URL+"/api/v1/status", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("expected CORS origin header, got %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestMgmt_BridgeWebSocketPathProxiesToBridgeServer(t *testing.T) {
	mgmt := NewManagementServer(0, "", []string{"*"})
	mgmt.RegisterEngine("p", NewEngine("p", &stubAgent{}, nil, "", LangEnglish))
	mgmt.SetBridgeServer(NewBridgeServer(9810, "bridge-secret", "/bridge/ws", []string{"*"}))

	mux := http.NewServeMux()
	ts := httptest.NewServer(mgmt.buildHandler(mux))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/bridge/ws?token=bridge-secret", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected websocket upgrade, got %d", resp.StatusCode)
	}
}

func TestMgmt_BridgeWebSocketPathWorksWhenBridgeServerSetAfterHandlerBuild(t *testing.T) {
	mgmt := NewManagementServer(0, "", []string{"*"})
	mgmt.RegisterEngine("p", NewEngine("p", &stubAgent{}, nil, "", LangEnglish))

	mux := http.NewServeMux()
	ts := httptest.NewServer(mgmt.buildHandler(mux))
	defer ts.Close()

	mgmt.SetBridgeServer(NewBridgeServer(9810, "bridge-secret", "/bridge/ws", []string{"*"}))

	req, _ := http.NewRequest("GET", ts.URL+"/bridge/ws?token=bridge-secret", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected websocket upgrade after late bridge setup, got %d", resp.StatusCode)
	}
}

func TestMgmt_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/status", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	var r mgmtResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode method not allowed response: %v", err)
	}
	resp.Body.Close()
}

func TestMgmt_ProjectModel_UsesSwitchModelWithActiveProvider(t *testing.T) {
	agent := &stubModelModeAgent{
		model: "gpt-4.1-mini",
		providers: []ProviderConfig{
			{
				Name:   "openai",
				Model:  "gpt-4.1-mini",
				Models: []ModelOption{{Name: "gpt-4.1", Alias: "gpt"}, {Name: "gpt-4.1-mini", Alias: "mini"}},
			},
		},
		active: "openai",
	}
	e := NewEngine("test-project", agent, nil, "", LangEnglish)
	var savedProvider, savedModel string
	e.SetProviderModelSaveFunc(func(providerName, model string) error {
		savedProvider = providerName
		savedModel = model
		return nil
	})

	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("test-project", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/model", "tok", map[string]string{"model": "gpt-4.1"})
	if !r.OK {
		t.Fatalf("update model failed: %s", r.Error)
	}

	if got := agent.GetModel(); got != "gpt-4.1" {
		t.Fatalf("GetModel() = %q, want gpt-4.1", got)
	}
	if got := agent.GetActiveProvider(); got == nil || got.Model != "gpt-4.1" {
		t.Fatalf("active provider model = %#v, want gpt-4.1", got)
	}
	if savedProvider != "openai" || savedModel != "gpt-4.1" {
		t.Fatalf("saved provider/model = %q/%q, want openai/gpt-4.1", savedProvider, savedModel)
	}
}

func TestMgmt_ProjectModel_SavesModelWithoutActiveProvider(t *testing.T) {
	agent := &stubModelModeAgent{
		model: "gpt-4.1-mini",
		providers: []ProviderConfig{
			{
				Name:   "openai",
				Model:  "gpt-4.1-mini",
				Models: []ModelOption{{Name: "gpt-4.1", Alias: "gpt"}, {Name: "gpt-4.1-mini", Alias: "mini"}},
			},
		},
	}
	e := NewEngine("test-project", agent, nil, "", LangEnglish)
	var savedModel string
	var providerSaveCalled bool
	e.SetModelSaveFunc(func(model string) error {
		savedModel = model
		return nil
	})
	e.SetProviderModelSaveFunc(func(providerName, model string) error {
		providerSaveCalled = true
		return nil
	})

	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("test-project", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/model", "tok", map[string]string{"model": "gpt-4.1"})
	if !r.OK {
		t.Fatalf("update model failed: %s", r.Error)
	}

	if got := agent.GetModel(); got != "gpt-4.1" {
		t.Fatalf("GetModel() = %q, want gpt-4.1", got)
	}
	if savedModel != "gpt-4.1" {
		t.Fatalf("saved model = %q, want gpt-4.1", savedModel)
	}
	if providerSaveCalled {
		t.Fatal("provider save callback should not be called without active provider")
	}
}

func TestMgmt_ProjectModel_ReturnsErrorWhenModelSaveFails(t *testing.T) {
	agent := &stubModelModeAgent{
		model: "gpt-4.1-mini",
		providers: []ProviderConfig{
			{
				Name:   "openai",
				Model:  "gpt-4.1-mini",
				Models: []ModelOption{{Name: "gpt-4.1", Alias: "gpt"}, {Name: "gpt-4.1-mini", Alias: "mini"}},
			},
		},
	}
	e := NewEngine("test-project", agent, nil, "", LangEnglish)
	e.SetModelSaveFunc(func(model string) error {
		return errors.New("disk full")
	})

	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("test-project", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/model", "tok", map[string]string{"model": "gpt-4.1"})
	if r.OK {
		t.Fatal("update model unexpectedly succeeded")
	}
	if !strings.Contains(r.Error, "disk full") {
		t.Fatalf("error = %q, want save failure", r.Error)
	}
	if got := agent.GetModel(); got != "gpt-4.1-mini" {
		t.Fatalf("GetModel() = %q, want unchanged gpt-4.1-mini", got)
	}
}

func TestMgmt_ProjectModels_UsesTimeoutContext(t *testing.T) {
	agent := &deadlineAwareModelAgent{}
	e := NewEngine("test-project", agent, nil, "", LangEnglish)

	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("test-project", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/models", "tok")
	if !r.OK {
		t.Fatalf("project models failed: %s", r.Error)
	}
	if !agent.sawDeadline() {
		t.Fatal("AvailableModels context has no deadline; want timeout-bounded context")
	}
}

func TestMgmt_RemoveGlobalProvider_PurgesFromEngines(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{
			{Name: "prov-a", BaseURL: "https://a.example"},
			{Name: "prov-b", BaseURL: "https://b.example"},
		},
		active: "prov-b",
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)

	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)

	removed := ""
	mgmt.SetRemoveGlobalProvider(func(name string) error {
		removed = name
		return nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/providers/", mgmt.wrap(mgmt.handleGlobalProviderRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	req, _ := http.NewRequest("DELETE", ts.URL+"/api/v1/providers/prov-a", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	resp.Body.Close()

	if removed != "prov-a" {
		t.Fatalf("removeGlobalProvider called with %q, want prov-a", removed)
	}

	remaining := agent.ListProviders()
	if len(remaining) != 1 {
		t.Fatalf("remaining providers = %d, want 1", len(remaining))
	}
	if remaining[0].Name != "prov-b" {
		t.Fatalf("remaining provider = %q, want prov-b", remaining[0].Name)
	}
}

func TestResolveGlobalProviderForAgent(t *testing.T) {
	g := GlobalProviderInfo{
		Name:    "relay",
		APIKey:  "sk-123",
		BaseURL: "https://api.example.com/anthropic",
		Model:   "claude-sonnet-4",
		Endpoints: map[string]string{
			"codex": "https://api.example.com/v1",
		},
		AgentModels: map[string]string{
			"codex": "gpt-5.3-codex",
		},
		AgentModelLists: map[string][]GlobalModelEntry{
			"codex": {{Model: "gpt-5.3-codex"}, {Model: "gpt-5.4"}},
		},
		Models: []struct {
			Model string `json:"model"`
			Alias string `json:"alias,omitempty"`
		}{{Model: "claude-sonnet-4"}, {Model: "claude-opus-4"}},
	}

	// claudecode: should use top-level values
	cc := resolveGlobalProviderForAgent(g, "claudecode")
	if cc.BaseURL != "https://api.example.com/anthropic" {
		t.Errorf("claudecode BaseURL = %q", cc.BaseURL)
	}
	if cc.Model != "claude-sonnet-4" {
		t.Errorf("claudecode Model = %q", cc.Model)
	}
	if len(cc.Models) != 2 || cc.Models[0].Name != "claude-sonnet-4" {
		t.Errorf("claudecode Models = %v", cc.Models)
	}

	// codex: should use per-agent overrides
	cx := resolveGlobalProviderForAgent(g, "codex")
	if cx.BaseURL != "https://api.example.com/v1" {
		t.Errorf("codex BaseURL = %q", cx.BaseURL)
	}
	if cx.Model != "gpt-5.3-codex" {
		t.Errorf("codex Model = %q", cx.Model)
	}
	if len(cx.Models) != 2 || cx.Models[0].Name != "gpt-5.3-codex" {
		t.Errorf("codex Models = %v", cx.Models)
	}
}

func TestMgmt_AddPlatformToNewProject_DoesNotRequireEngine(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")

	var savedProject, savedPlatName, savedPlatType string
	mgmt.SetAddPlatformToProject(func(proj, platName, platType string, opts map[string]any, workDir, agentType string, agentOptions map[string]any) error {
		savedProject = proj
		savedPlatName = platName
		savedPlatType = platType
		return nil
	})

	// "brand-new-project" has no engine registered — this must NOT return 404.
	r := mgmtPost(t, ts.URL+"/api/v1/projects/brand-new-project/add-platform", "tok", map[string]any{
		"name":    "Ops Bot",
		"type":    "dingtalk",
		"options": map[string]any{"client_id": "abc", "client_secret": "def"},
	})
	if !r.OK {
		t.Fatalf("add-platform to new project failed: %s — should not require a running engine", r.Error)
	}
	if savedProject != "brand-new-project" {
		t.Fatalf("saved project = %q, want brand-new-project", savedProject)
	}
	if savedPlatName != "Ops Bot" {
		t.Fatalf("saved platform name = %q, want Ops Bot", savedPlatName)
	}
	if savedPlatType != "dingtalk" {
		t.Fatalf("saved platform type = %q, want dingtalk", savedPlatType)
	}
}

func TestMgmt_OtherRoutesStillRequireEngine(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/projects/nonexistent/sessions", "tok")
	if r.OK {
		t.Fatal("expected 404 for sessions on nonexistent project")
	}
	if !strings.Contains(r.Error, "project not found") {
		t.Fatalf("error = %q, want project not found", r.Error)
	}
}

func mgmtPut(t *testing.T, url, token string, body any) mgmtResponse {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode PUT body: %v", err)
		}
	}
	req, _ := http.NewRequest("PUT", url, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", url, err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("decode PUT response: %v", err)
	}
	return r
}

// ── Restart ──

func TestMgmt_Restart(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtPost(t, ts.URL+"/api/v1/restart", "tok", nil)
	if !r.OK {
		t.Fatalf("restart failed: %s", r.Error)
	}
	select {
	case req := <-RestartCh:
		_ = req
	default:
	}
}

func TestMgmt_Restart_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/restart", "tok")
	if r.OK {
		t.Fatal("expected GET on restart to fail")
	}
}

// ── Agents ──

func TestMgmt_Agents(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/agents", "tok")
	if !r.OK {
		t.Fatalf("agents failed: %s", r.Error)
	}
	var data struct {
		Agents    []string `json:"agents"`
		Platforms []string `json:"platforms"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal agents: %v", err)
	}
}

func TestMgmt_Agents_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/agents", "tok", nil)
	if r.OK {
		t.Fatal("expected POST on agents to fail")
	}
}

// ── Global Settings ──

func TestMgmt_GlobalSettings_Get(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetGetGlobalSettings(func() map[string]any {
		return map[string]any{"language": "zh", "auto_start": true}
	})

	r := mgmtGet(t, ts.URL+"/api/v1/settings", "tok")
	if !r.OK {
		t.Fatalf("get settings failed: %s", r.Error)
	}
	var data map[string]any
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data["language"] != "zh" {
		t.Fatalf("language = %v, want zh", data["language"])
	}
}

func TestMgmt_GlobalSettings_GetNotAvailable(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/settings", "tok")
	if r.OK {
		t.Fatal("expected error when getGlobalSettings is nil")
	}
}

func TestMgmt_GlobalSettings_Patch(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")

	saved := map[string]any{}
	mgmt.SetGetGlobalSettings(func() map[string]any { return saved })
	mgmt.SetSaveGlobalSettings(func(updates map[string]any) error {
		for k, v := range updates {
			saved[k] = v
		}
		return nil
	})

	r := mgmtPatch(t, ts.URL+"/api/v1/settings", "tok", map[string]any{"language": "ja"})
	if !r.OK {
		t.Fatalf("patch settings failed: %s", r.Error)
	}
	if saved["language"] != "ja" {
		t.Fatalf("saved language = %v, want ja", saved["language"])
	}
}

func TestMgmt_GlobalSettings_PatchSaveError(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetSaveGlobalSettings(func(updates map[string]any) error {
		return errors.New("write failed")
	})
	r := mgmtPatch(t, ts.URL+"/api/v1/settings", "tok", map[string]any{"x": 1})
	if r.OK {
		t.Fatal("expected save error")
	}
	if !strings.Contains(r.Error, "write failed") {
		t.Fatalf("error = %q, want write failed", r.Error)
	}
}

// ── Project Send ──

func TestMgmt_ProjectSend_EmptyMessage(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/send", "tok", map[string]string{
		"session_key": "user1",
		"message":     "",
	})
	if r.OK {
		t.Fatal("expected error for empty message")
	}
	if !strings.Contains(r.Error, "message is required") {
		t.Fatalf("error = %q, want message required", r.Error)
	}
}

func TestMgmt_ProjectSend_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/send", "tok")
	if r.OK {
		t.Fatal("expected GET on send to fail")
	}
}

// ── Project Providers ──

func TestMgmt_ProjectProviders_NoProviderSwitcher(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/providers", "tok")
	if r.OK {
		t.Fatal("expected error when agent doesn't support ProviderSwitcher")
	}
	if !strings.Contains(r.Error, "provider switching") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_ProjectProviders_ListAndAdd(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{{Name: "openai", BaseURL: "https://api.openai.com"}},
		active:    "openai",
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)

	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// GET list
	r := mgmtGet(t, ts.URL+"/api/v1/projects/proj/providers", "tok")
	if !r.OK {
		t.Fatalf("list providers failed: %s", r.Error)
	}
	var list struct {
		Providers      []map[string]any `json:"providers"`
		ActiveProvider string           `json:"active_provider"`
	}
	if err := json.Unmarshal(r.Data, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list.Providers) != 1 {
		t.Fatalf("providers count = %d, want 1", len(list.Providers))
	}
	if list.ActiveProvider != "openai" {
		t.Fatalf("active = %q, want openai", list.ActiveProvider)
	}

	// POST add
	r = mgmtPost(t, ts.URL+"/api/v1/projects/proj/providers", "tok", map[string]string{
		"name":     "anthropic",
		"api_key":  "sk-test",
		"base_url": "https://api.anthropic.com",
	})
	if !r.OK {
		t.Fatalf("add provider failed: %s", r.Error)
	}
	if len(agent.providers) != 2 {
		t.Fatalf("providers count = %d, want 2", len(agent.providers))
	}
}

func TestMgmt_ProjectProviders_AddMissingName(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{{Name: "openai"}},
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPost(t, ts.URL+"/api/v1/projects/proj/providers", "tok", map[string]string{"api_key": "sk"})
	if r.OK {
		t.Fatal("expected error for missing name")
	}
}

func TestMgmt_ProjectProviders_Activate(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{
			{Name: "openai", BaseURL: "https://openai.com"},
			{Name: "claude", BaseURL: "https://claude.ai"},
		},
		active: "openai",
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPost(t, ts.URL+"/api/v1/projects/proj/providers/claude/activate", "tok", nil)
	if !r.OK {
		t.Fatalf("activate failed: %s", r.Error)
	}
	if agent.active != "claude" {
		t.Fatalf("active = %q, want claude", agent.active)
	}
}

func TestMgmt_ProjectProviders_ActivateNotFound(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{{Name: "openai"}},
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPost(t, ts.URL+"/api/v1/projects/proj/providers/nope/activate", "tok", nil)
	if r.OK {
		t.Fatal("expected 404 for nonexistent provider")
	}
}

func TestMgmt_ProjectProviders_DeleteActive(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{
			{Name: "openai"},
			{Name: "claude"},
		},
		active: "openai",
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtDelete(t, ts.URL+"/api/v1/projects/proj/providers/openai", "tok")
	if r.OK {
		t.Fatal("expected error when deleting active provider")
	}
	if !strings.Contains(r.Error, "active provider") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_ProjectProviders_DeleteInactive(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{
			{Name: "openai"},
			{Name: "claude"},
		},
		active: "openai",
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtDelete(t, ts.URL+"/api/v1/projects/proj/providers/claude", "tok")
	if !r.OK {
		t.Fatalf("delete failed: %s", r.Error)
	}
	if len(agent.providers) != 1 {
		t.Fatalf("providers = %d, want 1", len(agent.providers))
	}
}

// ── Project Provider Refs ──

func TestMgmt_ProjectProviderRefs_GetEmpty(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "openai"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtGet(t, ts.URL+"/api/v1/projects/proj/provider-refs", "tok")
	if !r.OK {
		t.Fatalf("get provider-refs failed: %s", r.Error)
	}
	var data struct {
		ProviderRefs []string `json:"provider_refs"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.ProviderRefs) != 0 {
		t.Fatalf("expected empty refs, got %v", data.ProviderRefs)
	}
}

func TestMgmt_ProjectProviderRefs_PutNotConfigured(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "openai"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPut(t, ts.URL+"/api/v1/projects/proj/provider-refs", "tok", map[string]any{
		"provider_refs": []string{"shared-1"},
	})
	if r.OK {
		t.Fatal("expected error when saveProviderRefs is nil")
	}
}

// ── Project Users ──

func TestMgmt_ProjectUsers_Get(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/users", "tok")
	if !r.OK {
		t.Fatalf("get users failed: %s", r.Error)
	}
}

func TestMgmt_ProjectUsers_PatchInvalidJSON(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/v1/projects/test-project/users", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

// ── Project Delete ──

func TestMgmt_ProjectDelete_NotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/projects/test-project", "tok")
	if r.OK {
		t.Fatal("expected error when removeProject is nil")
	}
	if !strings.Contains(r.Error, "not configured") {
		t.Fatalf("error = %q", r.Error)
	}
}

// ── Global Providers ──

func TestMgmt_GlobalProviders_GetEmpty(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/providers", "tok")
	if !r.OK {
		t.Fatalf("get global providers failed: %s", r.Error)
	}
	var data struct {
		Providers []any `json:"providers"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.Providers) != 0 {
		t.Fatalf("expected empty, got %d", len(data.Providers))
	}
}

func TestMgmt_GlobalProviders_GetWithFunc(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetListGlobalProviders(func() ([]GlobalProviderInfo, error) {
		return []GlobalProviderInfo{{Name: "shared-relay"}}, nil
	})

	r := mgmtGet(t, ts.URL+"/api/v1/providers", "tok")
	if !r.OK {
		t.Fatalf("get providers failed: %s", r.Error)
	}
}

func TestMgmt_GlobalProviders_PostNotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/providers", "tok", map[string]string{"name": "new"})
	if r.OK {
		t.Fatal("expected error when addGlobalProvider is nil")
	}
}

func TestMgmt_GlobalProviders_PostMissingName(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetAddGlobalProvider(func(info GlobalProviderInfo) error { return nil })

	r := mgmtPost(t, ts.URL+"/api/v1/providers", "tok", map[string]string{"api_key": "sk"})
	if r.OK {
		t.Fatal("expected error for missing name")
	}
}

func TestMgmt_GlobalProviders_PostSuccess(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	var added string
	mgmt.SetAddGlobalProvider(func(info GlobalProviderInfo) error {
		added = info.Name
		return nil
	})

	r := mgmtPost(t, ts.URL+"/api/v1/providers", "tok", map[string]string{"name": "new-relay"})
	if !r.OK {
		t.Fatalf("add global provider failed: %s", r.Error)
	}
	if added != "new-relay" {
		t.Fatalf("added = %q, want new-relay", added)
	}
}

func TestMgmt_GlobalProviders_PostDuplicate(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetAddGlobalProvider(func(info GlobalProviderInfo) error {
		return errors.New("already exists: " + info.Name)
	})

	r := mgmtPost(t, ts.URL+"/api/v1/providers", "tok", map[string]string{"name": "dup"})
	if r.OK {
		t.Fatal("expected conflict error")
	}
}

func TestMgmt_GlobalProviders_UpdateNotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPut(t, ts.URL+"/api/v1/providers/some-provider", "tok", map[string]string{"name": "some-provider"})
	if r.OK {
		t.Fatal("expected error when updateGlobalProvider is nil")
	}
}

func TestMgmt_GlobalProviders_UpdateSuccess(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	var updated string
	mgmt.SetUpdateGlobalProvider(func(name string, info GlobalProviderInfo) error {
		updated = name
		return nil
	})

	r := mgmtPut(t, ts.URL+"/api/v1/providers/relay-1", "tok", map[string]string{"model": "gpt-5"})
	if !r.OK {
		t.Fatalf("update global provider failed: %s", r.Error)
	}
	if updated != "relay-1" {
		t.Fatalf("updated = %q, want relay-1", updated)
	}
}

func TestMgmt_GlobalProviders_DeleteNotFound(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetRemoveGlobalProvider(func(name string) error {
		return errors.New("not found: " + name)
	})

	r := mgmtDelete(t, ts.URL+"/api/v1/providers/nope", "tok")
	if r.OK {
		t.Fatal("expected 404")
	}
}

// ── Provider Presets ──

func TestMgmt_ProviderPresets_NilFunc(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/providers/presets", "tok")
	if !r.OK {
		t.Fatalf("presets failed: %s", r.Error)
	}
}

func TestMgmt_ProviderPresets_WithFunc(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetFetchPresets(func() (*ProviderPresetsResponse, error) {
		return &ProviderPresetsResponse{Version: 2}, nil
	})

	r := mgmtGet(t, ts.URL+"/api/v1/providers/presets", "tok")
	if !r.OK {
		t.Fatalf("presets failed: %s", r.Error)
	}
}

func TestMgmt_ProviderPresets_Error(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetFetchPresets(func() (*ProviderPresetsResponse, error) {
		return nil, errors.New("network error")
	})

	r := mgmtGet(t, ts.URL+"/api/v1/providers/presets", "tok")
	if r.OK {
		t.Fatal("expected error")
	}
}

// ── Skills ──

func TestMgmt_Skills(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtGet(t, ts.URL+"/api/v1/skills", "tok")
	if !r.OK {
		t.Fatalf("skills failed: %s", r.Error)
	}
	var data struct {
		Projects []projectSkills `json:"projects"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(data.Projects))
	}
}

func TestMgmt_Skills_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/skills", "tok", nil)
	if r.OK {
		t.Fatal("expected POST on skills to fail")
	}
}

func TestMgmt_SkillPresets_NilFunc(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/skills/presets", "tok")
	if !r.OK {
		t.Fatalf("skill presets failed: %s", r.Error)
	}
}

func TestMgmt_SkillPresets_WithFunc(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetFetchSkillPresets(func() (*SkillPresetsResponse, error) {
		return &SkillPresetsResponse{Version: 1}, nil
	})
	r := mgmtGet(t, ts.URL+"/api/v1/skills/presets", "tok")
	if !r.OK {
		t.Fatalf("skill presets failed: %s", r.Error)
	}
}

func TestMgmt_SkillPresets_Error(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetFetchSkillPresets(func() (*SkillPresetsResponse, error) {
		return nil, errors.New("fetch failed")
	})
	r := mgmtGet(t, ts.URL+"/api/v1/skills/presets", "tok")
	if r.OK {
		t.Fatal("expected error")
	}
}

func TestMgmt_Enterprise_NotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/enterprise/overview", "tok")
	if r.OK {
		t.Fatal("expected enterprise overview to fail without store")
	}
}

func TestMgmt_Enterprise_CRUDAndLeaderboard(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetEnterpriseStore(NewEnterpriseStore(""))

	tenantResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/tenants", "tok", map[string]any{
		"name": "Acme AI",
		"slug": "acme-ai",
	})
	if !tenantResp.OK {
		t.Fatalf("create tenant failed: %s", tenantResp.Error)
	}
	var tenant EnterpriseTenant
	if err := json.Unmarshal(tenantResp.Data, &tenant); err != nil {
		t.Fatalf("unmarshal tenant: %v", err)
	}

	userResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/users", "tok", map[string]any{
		"tenant_id":    tenant.ID,
		"display_name": "Alice",
		"email":        "alice@example.com",
		"role":         "member",
	})
	if !userResp.OK {
		t.Fatalf("create user failed: %s", userResp.Error)
	}
	var user EnterpriseUser
	if err := json.Unmarshal(userResp.Data, &user); err != nil {
		t.Fatalf("unmarshal user: %v", err)
	}

	spaceResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/spaces", "tok", map[string]any{
		"tenant_id":     tenant.ID,
		"owner_user_id": user.ID,
		"name":          "Alice Workspace",
		"workspace_dir": "D:\\tenant-data\\acme\\alice",
		"project_name":  "claude-enterprise",
	})
	if !spaceResp.OK {
		t.Fatalf("create space failed: %s", spaceResp.Error)
	}
	var space EnterpriseSpace
	if err := json.Unmarshal(spaceResp.Data, &space); err != nil {
		t.Fatalf("unmarshal space: %v", err)
	}

	providerResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/providers", "tok", map[string]any{
		"name":          "deepseek",
		"display_name":  "DeepSeek",
		"provider_type": "openai-compatible",
		"default_model": "deepseek-chat",
		"models":        []string{"deepseek-chat", "deepseek-reasoner"},
	})
	if !providerResp.OK {
		t.Fatalf("create provider failed: %s", providerResp.Error)
	}

	skillResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/skills", "tok", map[string]any{
		"tenant_id":     tenant.ID,
		"owner_user_id": user.ID,
		"name":          "weekly-report",
		"display_name":  "Weekly Report",
		"description":   "Generate a team weekly report",
		"scope":         "tenant",
		"status":        "published",
		"version":       "1.0.0",
	})
	if !skillResp.OK {
		t.Fatalf("create skill failed: %s", skillResp.Error)
	}

	usageResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/usage", "tok", map[string]any{
		"tenant_id":         tenant.ID,
		"user_id":           user.ID,
		"space_id":          space.ID,
		"provider_name":     "deepseek",
		"model_name":        "deepseek-chat",
		"prompt_tokens":     120,
		"completion_tokens": 80,
		"total_tokens":      200,
		"cost_micros":       3500,
		"request_kind":      "chat",
	})
	if !usageResp.OK {
		t.Fatalf("create usage failed: %s", usageResp.Error)
	}

	overviewResp := mgmtGet(t, ts.URL+"/api/v1/enterprise/overview", "tok")
	if !overviewResp.OK {
		t.Fatalf("enterprise overview failed: %s", overviewResp.Error)
	}
	var overview EnterpriseOverview
	if err := json.Unmarshal(overviewResp.Data, &overview); err != nil {
		t.Fatalf("unmarshal overview: %v", err)
	}
	if overview.TenantsCount != 1 || overview.UsersCount != 1 || overview.SpacesCount != 1 {
		t.Fatalf("unexpected overview counts: %+v", overview)
	}
	if overview.TotalTokens != 200 {
		t.Fatalf("total tokens = %d, want 200", overview.TotalTokens)
	}

	leaderboardResp := mgmtGet(t, ts.URL+"/api/v1/enterprise/leaderboard?group_by=user", "tok")
	if !leaderboardResp.OK {
		t.Fatalf("enterprise leaderboard failed: %s", leaderboardResp.Error)
	}
	var leaderboard struct {
		GroupBy     string                       `json:"group_by"`
		Leaderboard []EnterpriseLeaderboardEntry `json:"leaderboard"`
	}
	if err := json.Unmarshal(leaderboardResp.Data, &leaderboard); err != nil {
		t.Fatalf("unmarshal leaderboard: %v", err)
	}
	if len(leaderboard.Leaderboard) != 1 {
		t.Fatalf("leaderboard rows = %d, want 1", len(leaderboard.Leaderboard))
	}
	if leaderboard.Leaderboard[0].Tokens != 200 {
		t.Fatalf("leaderboard tokens = %d, want 200", leaderboard.Leaderboard[0].Tokens)
	}
}

// ── Cron PATCH (update job) ──

func TestMgmt_Enterprise_RBACAndProjects(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetEnterpriseStore(NewEnterpriseStore(""))

	tenantResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/tenants", "tok", map[string]any{
		"name": "Ops Tenant",
	})
	if !tenantResp.OK {
		t.Fatalf("create tenant failed: %s", tenantResp.Error)
	}
	var tenant EnterpriseTenant
	if err := json.Unmarshal(tenantResp.Data, &tenant); err != nil {
		t.Fatalf("unmarshal tenant: %v", err)
	}

	userResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/users", "tok", map[string]any{
		"tenant_id":    tenant.ID,
		"display_name": "Bob",
	})
	if !userResp.OK {
		t.Fatalf("create user failed: %s", userResp.Error)
	}
	var user EnterpriseUser
	if err := json.Unmarshal(userResp.Data, &user); err != nil {
		t.Fatalf("unmarshal user: %v", err)
	}

	roleResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/roles", "tok", map[string]any{
		"tenant_id":      tenant.ID,
		"name":           "Space Admin",
		"scope":          "tenant",
		"status":         "active",
		"permission_ids": []string{"space:view", "space:manage", "project:manage", "role:view"},
	})
	if !roleResp.OK {
		t.Fatalf("create role failed: %s", roleResp.Error)
	}
	var role EnterpriseRole
	if err := json.Unmarshal(roleResp.Data, &role); err != nil {
		t.Fatalf("unmarshal role: %v", err)
	}

	projectResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/projects", "tok", map[string]any{
		"tenant_id":     tenant.ID,
		"owner_user_id": user.ID,
		"name":          "ops-runtime",
		"workspace_dir": "D:\\ops\\runtime",
		"agent_type":    "claudecode",
		"agent_options": map[string]any{"work_dir": "D:\\ops\\runtime", "mode": "default"},
		"platforms": []map[string]any{
			{"type": "feishu"},
		},
	})
	if !projectResp.OK {
		t.Fatalf("create project failed: %s", projectResp.Error)
	}
	var project EnterpriseProjectProfile
	if err := json.Unmarshal(projectResp.Data, &project); err != nil {
		t.Fatalf("unmarshal project: %v", err)
	}

	bindingResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/role-bindings", "tok", map[string]any{
		"tenant_id":  tenant.ID,
		"role_id":    role.ID,
		"user_id":    user.ID,
		"project_id": project.ID,
		"scope":      "project",
		"status":     "active",
	})
	if !bindingResp.OK {
		t.Fatalf("create binding failed: %s", bindingResp.Error)
	}

	permResp := mgmtGet(t, ts.URL+"/api/v1/enterprise/permissions", "tok")
	if !permResp.OK {
		t.Fatalf("list permissions failed: %s", permResp.Error)
	}
	var perms struct {
		Permissions []EnterprisePermission `json:"permissions"`
	}
	if err := json.Unmarshal(permResp.Data, &perms); err != nil {
		t.Fatalf("unmarshal permissions: %v", err)
	}
	if len(perms.Permissions) == 0 {
		t.Fatal("expected builtin permissions")
	}

	accessResp := mgmtGet(t, ts.URL+"/api/v1/enterprise/effective-access?tenant_id="+tenant.ID+"&user_id="+user.ID+"&project_id="+project.ID, "tok")
	if !accessResp.OK {
		t.Fatalf("resolve access failed: %s", accessResp.Error)
	}
	var access EnterpriseAccessProfile
	if err := json.Unmarshal(accessResp.Data, &access); err != nil {
		t.Fatalf("unmarshal access: %v", err)
	}
	if len(access.RoleIDs) != 1 {
		t.Fatalf("role ids = %v, want 1", access.RoleIDs)
	}
	if len(access.PermissionIDs) == 0 {
		t.Fatal("expected resolved permissions")
	}

	projectListResp := mgmtGet(t, ts.URL+"/api/v1/enterprise/projects?tenant_id="+tenant.ID, "tok")
	if !projectListResp.OK {
		t.Fatalf("list projects failed: %s", projectListResp.Error)
	}
	var projects struct {
		Projects []EnterpriseProjectProfile `json:"projects"`
	}
	if err := json.Unmarshal(projectListResp.Data, &projects); err != nil {
		t.Fatalf("unmarshal projects: %v", err)
	}
	if len(projects.Projects) != 1 {
		t.Fatalf("projects = %d, want 1", len(projects.Projects))
	}
}

func TestMgmt_Enterprise_Tasks(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetEnterpriseStore(NewEnterpriseStore(""))

	tenantResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/tenants", "tok", map[string]any{
		"name": "Task Tenant",
	})
	if !tenantResp.OK {
		t.Fatalf("create tenant failed: %s", tenantResp.Error)
	}
	var tenant EnterpriseTenant
	if err := json.Unmarshal(tenantResp.Data, &tenant); err != nil {
		t.Fatalf("unmarshal tenant: %v", err)
	}

	userResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/users", "tok", map[string]any{
		"tenant_id":    tenant.ID,
		"display_name": "Task Owner",
	})
	if !userResp.OK {
		t.Fatalf("create user failed: %s", userResp.Error)
	}
	var user EnterpriseUser
	if err := json.Unmarshal(userResp.Data, &user); err != nil {
		t.Fatalf("unmarshal user: %v", err)
	}

	taskResp := mgmtPost(t, ts.URL+"/api/v1/enterprise/tasks", "tok", map[string]any{
		"tenant_id":        tenant.ID,
		"owner_user_id":    user.ID,
		"assignee_user_id": user.ID,
		"title":            "Follow up production alert",
		"description":      "Check alert noise and report status",
		"task_type":        "task",
		"priority":         "high",
		"status":           "todo",
		"tags":             []string{"alert", "ops"},
	})
	if !taskResp.OK {
		t.Fatalf("create task failed: %s", taskResp.Error)
	}
	var task EnterpriseTask
	if err := json.Unmarshal(taskResp.Data, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if task.ID == "" {
		t.Fatal("expected task id")
	}

	taskListResp := mgmtGet(t, ts.URL+"/api/v1/enterprise/tasks?tenant_id="+tenant.ID+"&priority=high", "tok")
	if !taskListResp.OK {
		t.Fatalf("list tasks failed: %s", taskListResp.Error)
	}
	var tasks struct {
		Tasks []EnterpriseTask `json:"tasks"`
	}
	if err := json.Unmarshal(taskListResp.Data, &tasks); err != nil {
		t.Fatalf("unmarshal tasks: %v", err)
	}
	if len(tasks.Tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(tasks.Tasks))
	}
	if tasks.Tasks[0].Title != "Follow up production alert" {
		t.Fatalf("task title = %q", tasks.Tasks[0].Title)
	}
}

func TestMgmt_CronPatch(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, err := NewCronStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	// Add a job
	r := mgmtPost(t, ts.URL+"/api/v1/cron", "tok", map[string]any{
		"project":   "test-project",
		"cron_expr": "0 9 * * *",
		"prompt":    "hello",
	})
	if !r.OK {
		t.Fatalf("cron add failed: %s", r.Error)
	}
	var job CronJob
	json.Unmarshal(r.Data, &job)

	// Patch it
	r = mgmtPatch(t, ts.URL+"/api/v1/cron/"+job.ID, "tok", map[string]any{
		"enabled": false,
	})
	if !r.OK {
		t.Fatalf("cron patch failed: %s", r.Error)
	}

	var updated CronJob
	json.Unmarshal(r.Data, &updated)
	if updated.Enabled {
		t.Fatal("expected enabled=false after patch")
	}
}

func TestMgmt_CronPatch_NonexistentJob(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, err := NewCronStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	r := mgmtPatch(t, ts.URL+"/api/v1/cron/nonexistent", "tok", map[string]any{"enabled": false})
	if r.OK {
		t.Fatal("expected error for nonexistent cron job")
	}
}

// ── Project routes: unknown sub-path ──

func TestMgmt_ProjectRoutes_UnknownSubpath(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/unknown-route", "tok")
	if r.OK {
		t.Fatal("expected 404 for unknown sub-path")
	}
}

// ── Session create missing session_key ──

func TestMgmt_SessionCreate_MissingKey(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/sessions", "tok", map[string]string{
		"name": "work",
	})
	if r.OK {
		t.Fatal("expected error for missing session_key")
	}
}

// ── Reload failure ──

func TestMgmt_Reload_Failure(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")
	e.configReloadFunc = func() (*ConfigReloadResult, error) {
		return nil, errors.New("parse error")
	}
	r := mgmtPost(t, ts.URL+"/api/v1/reload", "tok", nil)
	if r.OK {
		t.Fatal("expected reload failure")
	}
	if !strings.Contains(r.Error, "parse error") {
		t.Fatalf("error = %q, want parse error", r.Error)
	}
}

// ── Config PUT (save) ──

func TestMgmt_Config_Save(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	tmp := t.TempDir()
	cfgPath := tmp + "/config.toml"
	os.WriteFile(cfgPath, []byte("[display]\ntitle = \"old\"\n"), 0644)

	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/config", strings.NewReader("# new config\n"))
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// Without SetConfigFilePath, save should fail
	if resp.StatusCode == 200 {
		t.Fatal("expected error without config file path set")
	}
}

// ── CC-Switch providers ──

func TestMgmt_CCSwitchProviders_NotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/providers/cc-switch", "tok")
	if !r.OK {
		t.Fatalf("cc-switch get failed: %s", r.Error)
	}
	var data map[string]any
	json.Unmarshal(r.Data, &data)
	if data["available"] != false {
		t.Fatalf("expected available=false, got %v", data["available"])
	}
}

// ────────────────────────────────────────────────────────────────
// Edge cases & boundary tests below
// ────────────────────────────────────────────────────────────────

// ── Restart edge cases ──

func TestMgmt_Restart_AlreadyInProgress(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	// Fill the buffered channel (cap=1) so the next restart is rejected.
	RestartCh <- RestartRequest{}

	r := mgmtPost(t, ts.URL+"/api/v1/restart", "tok", nil)
	if r.OK {
		t.Fatal("expected conflict when restart channel is full")
	}
	if !strings.Contains(r.Error, "already in progress") {
		t.Fatalf("error = %q, want 'already in progress'", r.Error)
	}

	// Drain so other tests aren't affected.
	<-RestartCh
}

func TestMgmt_Restart_WithSessionKey(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtPost(t, ts.URL+"/api/v1/restart", "tok", map[string]string{
		"session_key": "user1",
		"platform":    "feishu",
	})
	if !r.OK {
		t.Fatalf("restart with session_key failed: %s", r.Error)
	}
	req := <-RestartCh
	if req.SessionKey != "user1" || req.Platform != "feishu" {
		t.Fatalf("restart request = %+v, want session_key=user1 platform=feishu", req)
	}
}

// ── Config edge cases ──

func TestMgmt_Config_NoPathSet(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/config", "tok")
	if r.OK {
		t.Fatal("expected error when configFilePath is empty")
	}
	if !strings.Contains(r.Error, "not set") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_Config_FileNotFound(t *testing.T) {
	srv, ts, _ := testManagementServer(t, "tok")
	srv.SetConfigFilePath("/nonexistent/path/config.toml")

	r := mgmtGet(t, ts.URL+"/api/v1/config", "tok")
	if r.OK {
		t.Fatal("expected error for missing config file")
	}
}

func TestMgmt_Config_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/config", "tok", nil)
	if r.OK {
		t.Fatal("expected POST on config to fail")
	}
}

// ── Settings edge cases ──

func TestMgmt_GlobalSettings_PatchInvalidJSON(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetSaveGlobalSettings(func(updates map[string]any) error { return nil })

	req, _ := http.NewRequest("PATCH", ts.URL+"/api/v1/settings", strings.NewReader("{bad json"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON body")
	}
	if !strings.Contains(r.Error, "invalid JSON") {
		t.Fatalf("error = %q, want 'invalid JSON'", r.Error)
	}
}

func TestMgmt_GlobalSettings_PatchNotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPatch(t, ts.URL+"/api/v1/settings", "tok", map[string]any{"x": 1})
	if r.OK {
		t.Fatal("expected error when saveGlobalSettings is nil")
	}
}

func TestMgmt_GlobalSettings_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/settings", "tok")
	if r.OK {
		t.Fatal("expected DELETE on settings to fail")
	}
}

// ── Send edge cases ──

func TestMgmt_ProjectSend_InvalidJSON(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/projects/test-project/send", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_ProjectSend_NonexistentProject(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/projects/nonexistent/send", "tok", map[string]string{
		"message": "hello",
	})
	if r.OK {
		t.Fatal("expected 404 for send to nonexistent project")
	}
}

// ── Project Detail PATCH edge cases ──

func TestMgmt_ProjectPatch_InvalidJSON(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	req, _ := http.NewRequest("PATCH", ts.URL+"/api/v1/projects/test-project", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON body")
	}
}

func TestMgmt_ProjectPatch_UnknownAgentType(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	agentType := "totally-unknown-agent"
	r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project", "tok", map[string]any{
		"agent_type": agentType,
	})
	if r.OK {
		t.Fatal("expected error for unknown agent type")
	}
	if !strings.Contains(r.Error, "unknown agent type") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_ProjectPatch_DisabledCommands(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")
	r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project", "tok", map[string]any{
		"disabled_commands": []string{"new", "delete"},
	})
	if !r.OK {
		t.Fatalf("patch disabled_commands failed: %s", r.Error)
	}
	got := e.GetDisabledCommands()
	if len(got) != 2 || got[0] != "new" || got[1] != "delete" {
		t.Fatalf("disabled_commands = %v, want [new delete]", got)
	}
}

func TestMgmt_ProjectPatch_AdminFrom(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")
	r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project", "tok", map[string]any{
		"admin_from": "user123",
	})
	if !r.OK {
		t.Fatalf("patch admin_from failed: %s", r.Error)
	}
	e.userRolesMu.RLock()
	got := e.adminFrom
	e.userRolesMu.RUnlock()
	if got != "user123" {
		t.Fatalf("adminFrom = %q, want user123", got)
	}
}

func TestMgmt_ProjectPatch_Language(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")
	for _, lang := range []string{"zh", "ja", "es", "zh-TW", "en"} {
		r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project", "tok", map[string]any{
			"language": lang,
		})
		if !r.OK {
			t.Fatalf("patch language=%s failed: %s", lang, r.Error)
		}
	}
	if e.i18n.CurrentLang() != LangEnglish {
		t.Fatalf("final lang = %s, want en", e.i18n.CurrentLang())
	}
}

func TestMgmt_ProjectDetail_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project", "tok", nil)
	if r.OK {
		t.Fatal("expected POST on project detail to fail")
	}
}

// ── Project Delete edge cases ──

func TestMgmt_ProjectDelete_Success(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	var removed string
	mgmt.SetRemoveProject(func(name string) error {
		removed = name
		return nil
	})
	r := mgmtDelete(t, ts.URL+"/api/v1/projects/test-project", "tok")
	if !r.OK {
		t.Fatalf("delete project failed: %s", r.Error)
	}
	if removed != "test-project" {
		t.Fatalf("removed = %q, want test-project", removed)
	}
	var data map[string]any
	json.Unmarshal(r.Data, &data)
	if data["restart_required"] != true {
		t.Fatal("expected restart_required=true")
	}
}

func TestMgmt_ProjectDelete_Error(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetRemoveProject(func(name string) error {
		return errors.New("cannot remove last project")
	})
	r := mgmtDelete(t, ts.URL+"/api/v1/projects/test-project", "tok")
	if r.OK {
		t.Fatal("expected error from removeProject")
	}
	if !strings.Contains(r.Error, "cannot remove last project") {
		t.Fatalf("error = %q", r.Error)
	}
}

// ── Session switch edge cases ──

func TestMgmt_SessionSwitch_Success(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")

	s1 := e.sessions.GetOrCreateActive("user1")
	s2 := e.sessions.NewSession("user1", "second session")

	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/sessions/switch", "tok", map[string]string{
		"session_key": "user1",
		"session_id":  s2.ID,
	})
	if !r.OK {
		t.Fatalf("switch failed: %s", r.Error)
	}
	current := e.sessions.GetOrCreateActive("user1")
	if current.ID == s1.ID {
		t.Fatal("session did not switch")
	}
}

func TestMgmt_SessionSwitch_MissingFields(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")

	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/sessions/switch", "tok", map[string]string{
		"session_key": "user1",
	})
	if r.OK {
		t.Fatal("expected error for missing session_id")
	}
	if !strings.Contains(r.Error, "required") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_SessionSwitch_InvalidSessionID(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")
	e.sessions.GetOrCreateActive("user1")

	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/sessions/switch", "tok", map[string]string{
		"session_key": "user1",
		"session_id":  "nonexistent-id",
	})
	if r.OK {
		t.Fatal("expected error for nonexistent session_id")
	}
}

func TestMgmt_SessionSwitch_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/sessions/switch", "tok")
	if r.OK {
		t.Fatal("expected GET on session switch to fail")
	}
}

func TestMgmt_SessionSwitch_InvalidJSON(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/projects/test-project/sessions/switch", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

// ── Session detail edge cases ──

func TestMgmt_SessionDetail_NotFound(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/sessions/nonexistent-id", "tok")
	if r.OK {
		t.Fatal("expected 404 for nonexistent session")
	}
}

func TestMgmt_SessionDetail_DeleteNotFound(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/projects/test-project/sessions/nonexistent-id", "tok")
	if r.OK {
		t.Fatal("expected 404 for deleting nonexistent session")
	}
}

func TestMgmt_SessionDetail_MethodNotAllowed(t *testing.T) {
	_, ts, e := testManagementServer(t, "tok")
	s := e.sessions.GetOrCreateActive("user1")
	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/sessions/"+s.ID, "tok", nil)
	if r.OK {
		t.Fatal("expected POST on session detail to fail")
	}
}

func TestMgmt_SessionCreate_InvalidJSON(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/projects/test-project/sessions", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_Sessions_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/projects/test-project/sessions", "tok")
	if r.OK {
		t.Fatal("expected DELETE on sessions list to fail")
	}
}

// ── Provider edge cases ──

func TestMgmt_ProjectProviders_PostInvalidJSON(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "a"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/projects/proj/providers", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_ProjectProviders_DeleteNotFound(t *testing.T) {
	agent := &stubProviderAgent{
		providers: []ProviderConfig{{Name: "openai"}},
		active:    "openai",
	}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtDelete(t, ts.URL+"/api/v1/projects/proj/providers/nonexistent", "tok")
	if r.OK {
		t.Fatal("expected 404 for nonexistent provider")
	}
}

func TestMgmt_ProjectProviders_MethodNotAllowed(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "a"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPut(t, ts.URL+"/api/v1/projects/proj/providers", "tok", nil)
	if r.OK {
		t.Fatal("expected PUT on providers list to fail")
	}
}

// ── Provider Refs edge cases ──

func TestMgmt_ProjectProviderRefs_PutInvalidJSON(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "a"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mgmt.SetSaveProviderRefs(func(proj string, refs []string) error { return nil })
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/projects/proj/provider-refs", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_ProjectProviderRefs_PutSaveError(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "a"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mgmt.SetSaveProviderRefs(func(proj string, refs []string) error {
		return errors.New("disk full")
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPut(t, ts.URL+"/api/v1/projects/proj/provider-refs", "tok", map[string]any{
		"provider_refs": []string{"shared"},
	})
	if r.OK {
		t.Fatal("expected error from save")
	}
	if !strings.Contains(r.Error, "disk full") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_ProjectProviderRefs_PutSuccess(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "a"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	var savedRefs []string
	mgmt.SetSaveProviderRefs(func(proj string, refs []string) error {
		savedRefs = refs
		return nil
	})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtPut(t, ts.URL+"/api/v1/projects/proj/provider-refs", "tok", map[string]any{
		"provider_refs": []string{"shared-1", "shared-2"},
	})
	if !r.OK {
		t.Fatalf("put provider-refs failed: %s", r.Error)
	}
	if len(savedRefs) != 2 {
		t.Fatalf("savedRefs = %v", savedRefs)
	}
}

func TestMgmt_ProjectProviderRefs_MethodNotAllowed(t *testing.T) {
	agent := &stubProviderAgent{providers: []ProviderConfig{{Name: "a"}}}
	e := NewEngine("proj", agent, nil, "", LangEnglish)
	mgmt := NewManagementServer(0, "tok", nil)
	mgmt.RegisterEngine("proj", e)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/", mgmt.wrap(mgmt.handleProjectRoutes))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	r := mgmtDelete(t, ts.URL+"/api/v1/projects/proj/provider-refs", "tok")
	if r.OK {
		t.Fatal("expected DELETE on provider-refs to fail")
	}
}

// ── Users edge cases ──

func TestMgmt_ProjectUsers_PatchValid(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project/users", "tok", map[string]any{
		"default_role": "member",
		"roles": map[string]any{
			"member": map[string]any{
				"user_ids": []string{"uid-member-1"},
			},
			"admin": map[string]any{
				"user_ids": []string{"uid-admin"},
			},
		},
	})
	if !r.OK {
		t.Fatalf("patch users failed: %s", r.Error)
	}
}

func TestMgmt_ProjectUsers_PatchInvalidRoleConfig(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPatch(t, ts.URL+"/api/v1/projects/test-project/users", "tok", map[string]any{
		"default_role": "nonexistent",
		"roles": map[string]any{
			"admin": map[string]any{
				"user_ids": []string{"uid-admin"},
			},
		},
	})
	if r.OK {
		t.Fatal("expected error when default_role doesn't match any defined role")
	}
	if !strings.Contains(r.Error, "invalid users config") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_ProjectUsers_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/projects/test-project/users", "tok")
	if r.OK {
		t.Fatal("expected DELETE on users to fail")
	}
}

// ── Global Providers edge cases ──

func TestMgmt_GlobalProviders_GetError(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetListGlobalProviders(func() ([]GlobalProviderInfo, error) {
		return nil, errors.New("db connection lost")
	})
	r := mgmtGet(t, ts.URL+"/api/v1/providers", "tok")
	if r.OK {
		t.Fatal("expected error from list")
	}
	if !strings.Contains(r.Error, "db connection lost") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_GlobalProviders_PostInvalidJSON(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetAddGlobalProvider(func(info GlobalProviderInfo) error { return nil })

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/providers", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_GlobalProviders_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/providers", "tok")
	if r.OK {
		t.Fatal("expected DELETE on /providers to fail")
	}
}

func TestMgmt_GlobalProviders_UpdateNotFound(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetUpdateGlobalProvider(func(name string, info GlobalProviderInfo) error {
		return errors.New("not found: " + name)
	})
	r := mgmtPut(t, ts.URL+"/api/v1/providers/nope", "tok", map[string]string{"model": "x"})
	if r.OK {
		t.Fatal("expected 404")
	}
}

func TestMgmt_GlobalProviders_UpdateInvalidJSON(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetUpdateGlobalProvider(func(name string, info GlobalProviderInfo) error { return nil })

	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/providers/test", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_GlobalProviders_DeleteNotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/providers/anything", "tok")
	if r.OK {
		t.Fatal("expected error when removeGlobalProvider is nil")
	}
}

func TestMgmt_GlobalProviders_DeleteSuccess(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	var deleted string
	mgmt.SetRemoveGlobalProvider(func(name string) error {
		deleted = name
		return nil
	})
	r := mgmtDelete(t, ts.URL+"/api/v1/providers/relay-old", "tok")
	if !r.OK {
		t.Fatalf("delete global provider failed: %s", r.Error)
	}
	if deleted != "relay-old" {
		t.Fatalf("deleted = %q, want relay-old", deleted)
	}
}

func TestMgmt_GlobalProviders_RouteMethodNotAllowed(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetUpdateGlobalProvider(func(name string, info GlobalProviderInfo) error { return nil })

	r := mgmtPost(t, ts.URL+"/api/v1/providers/test-prov", "tok", nil)
	if r.OK {
		t.Fatal("expected POST on /providers/{name} to fail")
	}
}

// ── Heartbeat edge cases ──

func TestMgmt_Heartbeat_PauseResumeRun(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	hs := NewHeartbeatScheduler("")
	mgmt.SetHeartbeatScheduler(hs)

	// pause/resume/run on unconfigured project → 404
	for _, action := range []string{"pause", "resume", "run"} {
		r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/heartbeat/"+action, "tok", nil)
		if r.OK {
			t.Fatalf("expected 404 for heartbeat %s on unconfigured project", action)
		}
	}
}

func TestMgmt_Heartbeat_IntervalTooSmall(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	hs := NewHeartbeatScheduler("")
	mgmt.SetHeartbeatScheduler(hs)

	r := mgmtPost(t, ts.URL+"/api/v1/projects/test-project/heartbeat/interval", "tok",
		map[string]any{"minutes": 0})
	if r.OK {
		t.Fatal("expected error for minutes < 1")
	}
	if !strings.Contains(r.Error, "minutes must be >= 1") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_Heartbeat_IntervalInvalidJSON(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	hs := NewHeartbeatScheduler("")
	mgmt.SetHeartbeatScheduler(hs)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/projects/test-project/heartbeat/interval", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_Heartbeat_UnknownAction(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	hs := NewHeartbeatScheduler("")
	mgmt.SetHeartbeatScheduler(hs)

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/heartbeat/unknown", "tok")
	if r.OK {
		t.Fatal("expected 404 for unknown heartbeat action")
	}
}

func TestMgmt_Heartbeat_MethodNotAllowed(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	hs := NewHeartbeatScheduler("")
	mgmt.SetHeartbeatScheduler(hs)

	r := mgmtGet(t, ts.URL+"/api/v1/projects/test-project/heartbeat/pause", "tok")
	if r.OK {
		t.Fatal("expected GET on pause to fail")
	}
}

// ── Cron edge cases ──

func TestMgmt_Cron_PostMissingCronExpr(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	r := mgmtPost(t, ts.URL+"/api/v1/cron", "tok", map[string]any{
		"project": "test-project",
		"prompt":  "hello",
	})
	if r.OK {
		t.Fatal("expected error for missing cron_expr")
	}
	if !strings.Contains(r.Error, "cron_expr is required") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_Cron_PostMissingPromptAndExec(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	r := mgmtPost(t, ts.URL+"/api/v1/cron", "tok", map[string]any{
		"project":   "test-project",
		"cron_expr": "0 9 * * *",
	})
	if r.OK {
		t.Fatal("expected error for missing prompt and exec")
	}
}

func TestMgmt_Cron_PostPromptAndExecMutuallyExclusive(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	r := mgmtPost(t, ts.URL+"/api/v1/cron", "tok", map[string]any{
		"project":   "test-project",
		"cron_expr": "0 9 * * *",
		"prompt":    "hello",
		"exec":      "ls -la",
	})
	if r.OK {
		t.Fatal("expected error when both prompt and exec are set")
	}
	if !strings.Contains(r.Error, "mutually exclusive") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_Cron_PostInvalidJSON(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/cron", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_Cron_MethodNotAllowed(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	r := mgmtDelete(t, ts.URL+"/api/v1/cron", "tok")
	if r.OK {
		t.Fatal("expected DELETE on /cron to fail")
	}
}

func TestMgmt_CronByID_MethodNotAllowed(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	r := mgmtGet(t, ts.URL+"/api/v1/cron/some-id", "tok")
	if r.OK {
		t.Fatal("expected GET on /cron/{id} to fail")
	}
}

func TestMgmt_CronPatch_InvalidJSON(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	req, _ := http.NewRequest("PATCH", ts.URL+"/api/v1/cron/some-id", strings.NewReader("{bad"))
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var r mgmtResponse
	json.NewDecoder(resp.Body).Decode(&r)
	if r.OK {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMgmt_CronByID_EmptyID(t *testing.T) {
	mgmt, ts, e := testManagementServer(t, "tok")
	store, _ := NewCronStore(t.TempDir())
	cs := NewCronScheduler(store)
	cs.RegisterEngine("test-project", e)
	mgmt.SetCronScheduler(cs)

	r := mgmtDelete(t, ts.URL+"/api/v1/cron/", "tok")
	if r.OK {
		t.Fatal("expected error for empty cron id")
	}
}

// ── Project routes: empty project name ──

func TestMgmt_ProjectRoutes_EmptyProjectName(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/projects/", "tok")
	// /projects/ with empty trailing slash is dispatched to handleProjectRoutes
	// which returns "project name required" error.
	if r.OK {
		t.Fatal("expected error for empty project name in project routes")
	}
}

// ── Reload edge cases ──

func TestMgmt_Reload_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtGet(t, ts.URL+"/api/v1/reload", "tok")
	if r.OK {
		t.Fatal("expected GET on reload to fail")
	}
}

func TestMgmt_Reload_NoReloadFunc(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/reload", "tok", nil)
	if !r.OK {
		t.Fatalf("reload with nil reloadFunc should succeed: %s", r.Error)
	}
}

// ── CC-Switch edge cases ──

func TestMgmt_CCSwitchProviders_PostNotConfigured(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtPost(t, ts.URL+"/api/v1/providers/cc-switch", "tok", map[string]any{
		"names": []string{"relay-1"},
	})
	if r.OK {
		t.Fatal("expected error when cc-switch not configured")
	}
}

func TestMgmt_CCSwitchProviders_PostMissingNames(t *testing.T) {
	mgmt, ts, _ := testManagementServer(t, "tok")
	mgmt.SetListCCSwitchProviders(func() ([]CCSwitchProviderInfo, error) {
		return nil, nil
	})
	mgmt.SetAddGlobalProvider(func(info GlobalProviderInfo) error { return nil })

	r := mgmtPost(t, ts.URL+"/api/v1/providers/cc-switch", "tok", map[string]any{
		"names": []string{},
	})
	if r.OK {
		t.Fatal("expected error for empty names")
	}
	if !strings.Contains(r.Error, "names is required") {
		t.Fatalf("error = %q", r.Error)
	}
}

func TestMgmt_CCSwitchProviders_MethodNotAllowed(t *testing.T) {
	_, ts, _ := testManagementServer(t, "tok")
	r := mgmtDelete(t, ts.URL+"/api/v1/providers/cc-switch", "tok")
	if r.OK {
		t.Fatal("expected DELETE on cc-switch to fail")
	}
}
