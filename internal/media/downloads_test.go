package media

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Bongbetic/whatsapp-linux/internal/settings"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
)

func TestShouldAutoDownload(t *testing.T) {
	s := settings.New() // photos+audio on, video+docs off (§10)
	if !ShouldAutoDownload(s, "image", 100) || !ShouldAutoDownload(s, "voice", 100) {
		t.Fatal("photos/voice default on")
	}
	if ShouldAutoDownload(s, "video", 100) || ShouldAutoDownload(s, "doc", 100) {
		t.Fatal("video/docs default off")
	}
	if ShouldAutoDownload(s, "image", 17<<20) {
		t.Fatal(">16MB always manual")
	}
}

func TestManagerQueue(t *testing.T) {
	db, err := appstore.Open(t.TempDir() + "/app.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var live, peak int64
	m := NewManager(db, func(ctx context.Context, chat, msg string, w io.Writer) error {
		n := atomic.AddInt64(&live, 1)
		for {
			p := atomic.LoadInt64(&peak)
			if n <= p || atomic.CompareAndSwapInt64(&peak, p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt64(&live, -1)
		_, _ = w.Write([]byte("bytes"))
		return nil
	})
	ctx := context.Background()
	var bufs [5]bytes.Buffer
	for i := range bufs {
		m.Enqueue(ctx, "c", string(rune('a'+i)), 5, &bufs[i])
	}
	m.Wait()
	if p := atomic.LoadInt64(&peak); p > MaxConcurrentDownloads {
		t.Fatalf("peak concurrency %d > %d", p, MaxConcurrentDownloads)
	}
	for i := range bufs {
		if bufs[i].String() != "bytes" {
			t.Fatalf("job %d missing output", i)
		}
	}
}
