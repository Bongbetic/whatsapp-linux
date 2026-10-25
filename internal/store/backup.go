package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// BackupTo checkpoints the WAL and copies app.db to dst (WL-6.2).
// Run before migrations on upgrade and before destructive ops.
func (d *DB) BackupTo(dst string) error {
	if _, _, _, err := d.checkpoint(); err != nil {
		return err
	}
	src, err := os.Open(d.path)
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

func (d *DB) checkpoint() (busy, log, checkpointed int, err error) {
	err = d.db.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`).Scan(&busy, &log, &checkpointed)
	return busy, log, checkpointed, err
}

// ExportChat writes one chat's history as txt or json (WL-6.2).
func (d *DB) ExportChat(ctx context.Context, chatJID, format string, w io.Writer) error {
	chat, err := d.GetChat(ctx, chatJID)
	if err != nil {
		return err
	}
	var all []bridge.Message
	var beforeTs int64
	var beforeID string
	for {
		page, err := d.ListMessages(ctx, chatJID, 500, beforeTs, beforeID)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		beforeTs, beforeID = page[len(page)-1].Ts.Unix(), page[len(page)-1].ID
		if len(page) < 500 {
			break
		}
	}
	// Oldest first for reading.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	switch format {
	case "txt":
		return exportTxt(chat, all, w)
	case "json":
		return exportJSON(chat, all, w)
	default:
		return fmt.Errorf("unknown export format %q", format)
	}
}

func exportTxt(chat bridge.Chat, msgs []bridge.Message, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "Chat: %s (%s)\nExported: %s\n\n", chat.Title, chat.JID, time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	for _, m := range msgs {
		who := m.Sender.User
		if m.IsFromMe {
			who = "Me"
		}
		body := m.Body
		if body == "" {
			body = "[" + m.Kind + "]"
		}
		if _, err := fmt.Fprintf(w, "[%s] %s: %s\n", m.Ts.Format("2006-01-02 15:04"), who, body); err != nil {
			return err
		}
	}
	return nil
}

func exportJSON(chat bridge.Chat, msgs []bridge.Message, w io.Writer) error {
	type msgOut struct {
		ID     string `json:"id"`
		Sender string `json:"sender"`
		FromMe bool   `json:"from_me"`
		Ts     int64  `json:"ts"`
		Kind   string `json:"kind"`
		Body   string `json:"body"`
		State  string `json:"state"`
	}
	out := struct {
		Chat     string   `json:"chat"`
		Title    string   `json:"title"`
		Messages []msgOut `json:"messages"`
	}{Chat: chat.JID.String(), Title: chat.Title}
	for _, m := range msgs {
		out.Messages = append(out.Messages, msgOut{m.ID, m.Sender.String(), m.IsFromMe, m.Ts.Unix(), m.Kind, m.Body, m.State})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
