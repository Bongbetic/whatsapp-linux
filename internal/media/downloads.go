package media

import (
	"context"
	"io"
	"sync"

	"github.com/Bongbetic/whatsapp-linux/internal/settings"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
)

// MaxConcurrentDownloads bounds parallel fetches (§8: max 3).
const MaxConcurrentDownloads = 3

// Fetch downloads one attachment to w; the Engine's DownloadMedia is
// the production implementation.
type Fetch func(ctx context.Context, chatJID, msgID string, w io.Writer) error

// Manager runs the download queue (§8, WL-3.3): at most 3 concurrent,
// state persisted in the downloads table, auto-download policy from
// settings evaluated on insert.
type Manager struct {
	db    *appstore.DB
	fetch Fetch
	sem   chan struct{}
	wg    sync.WaitGroup
}

// NewManager builds the queue over an open app.db.
func NewManager(db *appstore.DB, fetch Fetch) *Manager {
	return &Manager{db: db, fetch: fetch, sem: make(chan struct{}, MaxConcurrentDownloads)}
}

// ShouldAutoDownload applies the settings policy (WL-3.3).
func ShouldAutoDownload(s *settings.Store, kind string, size int64) bool {
	const big = 16 << 20 // >16 MB always manual
	if size > big {
		return false
	}
	switch kind {
	case "image":
		return s.GetBool(settings.KeyAutoPhotos)
	case "audio", "voice":
		return s.GetBool(settings.KeyAutoAudio)
	case "video":
		return s.GetBool(settings.KeyAutoVideo)
	case "doc":
		return s.GetBool(settings.KeyAutoDocs)
	}
	return false
}

// Enqueue records a queued download and starts it when a slot frees.
func (m *Manager) Enqueue(ctx context.Context, chatJID, msgID string, total int64, dst io.Writer) {
	_ = m.db.SetDownload(ctx, chatJID, msgID, "queued", 0, total)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.sem <- struct{}{}
		defer func() { <-m.sem }()
		_ = m.db.SetDownload(ctx, chatJID, msgID, "active", 0, total)
		if err := m.fetch(ctx, chatJID, msgID, dst); err != nil {
			_ = m.db.SetDownload(ctx, chatJID, msgID, "failed", 0, total)
			return
		}
		_ = m.db.SetDownload(ctx, chatJID, msgID, "done", total, total)
	}()
}

// Wait blocks until queued downloads finish (tests, shutdown flush).
func (m *Manager) Wait() { m.wg.Wait() }
