package core

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMessageDeduper_Basic(t *testing.T) {
	d := NewMessageDeduper(filepath.Join(t.TempDir(), "dedupe"), 20*time.Millisecond)

	if !d.ShouldProcess("feishu", "m-1") {
		t.Fatal("first message should pass")
	}
	if d.ShouldProcess("feishu", "m-1") {
		t.Fatal("duplicate message should be blocked")
	}

	time.Sleep(25 * time.Millisecond)
	if !d.ShouldProcess("feishu", "m-1") {
		t.Fatal("expired duplicate window should pass again")
	}
}

func TestMessageDeduper_EmptyMessageID(t *testing.T) {
	d := NewMessageDeduper(filepath.Join(t.TempDir(), "dedupe"), time.Minute)
	if !d.ShouldProcess("feishu", "") {
		t.Fatal("empty message id should pass")
	}
	if !d.ShouldProcess("feishu", "") {
		t.Fatal("empty message id should always pass")
	}
}

func TestMessageDeduper_ConcurrentDuplicate(t *testing.T) {
	d := NewMessageDeduper(filepath.Join(t.TempDir(), "dedupe"), time.Minute)

	const workers = 24
	var passed int32
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			if d.ShouldProcess("feishu", "m-concurrent") {
				atomic.AddInt32(&passed, 1)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&passed); got != 1 {
		t.Fatalf("concurrent duplicate pass count = %d, want 1", got)
	}
}
