package core

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

// ChatProjectRouter persists "which project should answer this chat" routing.
// Key format: "<platform>:<channelID>".
type ChatProjectRouter struct {
	mu                sync.Mutex
	routes            map[string]string
	storePath         string
	lastLoadedModTime time.Time
	lastLoadedSize    int64
}

func NewChatProjectRouter(storePath string) *ChatProjectRouter {
	r := &ChatProjectRouter{
		routes:    make(map[string]string),
		storePath: storePath,
	}
	if storePath != "" {
		r.load()
	}
	return r
}

func chatProjectRouteKey(platform, channelID string) string {
	platform = strings.TrimSpace(platform)
	channelID = strings.TrimSpace(channelID)
	if platform == "" || channelID == "" {
		return ""
	}
	return platform + ":" + channelID
}

func (r *ChatProjectRouter) Get(platform, channelID string) string {
	key := chatProjectRouteKey(platform, channelID)
	if key == "" {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshLocked()
	return strings.TrimSpace(r.routes[key])
}

// Claim selects project for the route if missing, then returns the effective project.
func (r *ChatProjectRouter) Claim(platform, channelID, project string) string {
	key := chatProjectRouteKey(platform, channelID)
	project = strings.TrimSpace(project)
	if key == "" || project == "" {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshLocked()
	if cur := strings.TrimSpace(r.routes[key]); cur != "" {
		return cur
	}
	r.routes[key] = project
	r.saveLocked()
	return project
}

func (r *ChatProjectRouter) Set(platform, channelID, project string) {
	key := chatProjectRouteKey(platform, channelID)
	project = strings.TrimSpace(project)
	if key == "" || project == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshLocked()
	r.routes[key] = project
	r.saveLocked()
}

func (r *ChatProjectRouter) Clear(platform, channelID string) {
	key := chatProjectRouteKey(platform, channelID)
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshLocked()
	delete(r.routes, key)
	r.saveLocked()
}

func (r *ChatProjectRouter) load() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refreshLocked()
}

func (r *ChatProjectRouter) refreshLocked() {
	if r.storePath == "" {
		return
	}
	info, err := os.Stat(r.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			r.routes = make(map[string]string)
			r.lastLoadedModTime = time.Time{}
			r.lastLoadedSize = 0
			return
		}
		slog.Error("chat project router: stat error", "error", err)
		return
	}
	if !r.lastLoadedModTime.IsZero() && info.ModTime().Equal(r.lastLoadedModTime) && info.Size() == r.lastLoadedSize {
		return
	}

	data, err := os.ReadFile(r.storePath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("chat project router: read error", "error", err)
		}
		return
	}
	loaded := make(map[string]string)
	if len(data) > 0 {
		if err := json.Unmarshal(data, &loaded); err != nil {
			slog.Error("chat project router: unmarshal error", "error", err)
			return
		}
	}
	r.routes = loaded
	r.lastLoadedModTime = info.ModTime()
	r.lastLoadedSize = info.Size()
}

func (r *ChatProjectRouter) saveLocked() {
	if r.storePath == "" {
		return
	}
	data, err := json.MarshalIndent(r.routes, "", "  ")
	if err != nil {
		slog.Error("chat project router: marshal error", "error", err)
		return
	}
	if err := AtomicWriteFile(r.storePath, data, 0o644); err != nil {
		slog.Error("chat project router: save error", "error", err)
		return
	}
	if info, err := os.Stat(r.storePath); err == nil {
		r.lastLoadedModTime = info.ModTime()
		r.lastLoadedSize = info.Size()
	}
}
