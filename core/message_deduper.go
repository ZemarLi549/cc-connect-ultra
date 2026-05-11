package core

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultMessageDedupTTL = 2 * time.Minute

// MessageDeduper provides best-effort cross-process message de-duplication.
// It uses tiny lock files keyed by platform+messageID.
type MessageDeduper struct {
	dir string
	ttl time.Duration
}

func NewMessageDeduper(dir string, ttl time.Duration) *MessageDeduper {
	if ttl <= 0 {
		ttl = defaultMessageDedupTTL
	}
	if dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	return &MessageDeduper{
		dir: dir,
		ttl: ttl,
	}
}

func (d *MessageDeduper) ShouldProcess(platform, messageID string) bool {
	if d == nil {
		return true
	}
	platform = strings.TrimSpace(platform)
	messageID = strings.TrimSpace(messageID)
	if d.dir == "" || messageID == "" {
		return true
	}

	sum := sha256.Sum256([]byte(platform + ":" + messageID))
	p := filepath.Join(d.dir, hex.EncodeToString(sum[:])+".seen")
	now := time.Now()

	if info, err := os.Stat(p); err == nil {
		if now.Sub(info.ModTime()) <= d.ttl {
			return false
		}
		_ = os.Remove(p)
	}

	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			// Another goroutine/process won the race and created the marker first.
			return false
		}
		// Fail open on filesystem errors to avoid dropping valid messages.
		return true
	}
	_, _ = f.WriteString(now.Format(time.RFC3339Nano))
	_ = f.Close()
	return true
}
