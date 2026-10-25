package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// SearchMessages runs a global FTS5 phrase query (WL-2.7), newest
// first. chatJID scopes to one chat when non-empty.
func (d *DB) SearchMessages(ctx context.Context, query, chatJID string, limit int) ([]bridge.Message, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	phrase := `"` + strings.ReplaceAll(q, `"`, `""`) + `"`
	stmt := `SELECT m.chat_jid,m.id,m.sender_jid,m.is_from_me,m.ts,m.kind,m.body,m.quoted_id,
		m.media_path,m.media_mime,m.media_size,m.thumb_path,m.state,m.edited,m.starred,m.view_once
		FROM messages_fts f JOIN messages m ON m.rowid=f.rowid
		WHERE messages_fts MATCH ?`
	args := []any{phrase}
	if chatJID != "" {
		stmt += ` AND m.chat_jid=?`
		args = append(args, chatJID)
	}
	stmt += ` ORDER BY m.ts DESC LIMIT ?`
	args = append(args, limit)
	rows, err := d.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bridge.Message
	for rows.Next() {
		var m bridge.Message
		var chat, sender string
		var fromMe, edited, starred, vo int
		var ts int64
		var size sql.NullInt64
		var quoted, mediaPath, mediaMIME, thumb sql.NullString
		if err := rows.Scan(&chat, &m.ID, &sender, &fromMe, &ts, &m.Kind, &m.Body, &quoted,
			&mediaPath, &mediaMIME, &size, &thumb, &m.State, &edited, &starred, &vo); err != nil {
			return nil, err
		}
		m.Chat, m.Sender = parseJID(chat), parseJID(sender)
		m.IsFromMe, m.Edited, m.Starred, m.ViewOnce = fromMe == 1, edited == 1, starred == 1, vo == 1
		m.Ts = time.Unix(ts, 0)
		m.QuotedID, m.MediaPath, m.MediaMIME, m.ThumbPath = quoted.String, mediaPath.String, mediaMIME.String, thumb.String
		m.MediaSize = size.Int64
		out = append(out, m)
	}
	return out, rows.Err()
}
