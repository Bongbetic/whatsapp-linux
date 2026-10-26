package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// UpsertMessage inserts or replaces a message row.
func (d *DB) UpsertMessage(ctx context.Context, m bridge.Message) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO messages(chat_jid,id,sender_jid,is_from_me,ts,kind,body,quoted_id,media_path,media_mime,media_size,media_sha256,thumb_path,waveform,state,edited,starred,view_once)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(chat_jid,id) DO UPDATE SET sender_jid=excluded.sender_jid,ts=excluded.ts,kind=excluded.kind,body=excluded.body,
		quoted_id=excluded.quoted_id,media_path=excluded.media_path,media_mime=excluded.media_mime,media_size=excluded.media_size,
		thumb_path=excluded.thumb_path,waveform=excluded.waveform,state=excluded.state,edited=excluded.edited,
		starred=excluded.starred,view_once=excluded.view_once`,
		m.Chat.String(), m.ID, m.Sender.String(), boolInt(m.IsFromMe), m.Ts.Unix(), m.Kind, m.Body,
		nullStr(m.QuotedID), nullStr(m.MediaPath), nullStr(m.MediaMIME), m.MediaSize, nil,
		nullStr(m.ThumbPath), m.Waveform, m.State, boolInt(m.Edited), boolInt(m.Starred), boolInt(m.ViewOnce))
	return err
}

// UpdateMessageState moves one message along sending→sent→… (§8).
func (d *DB) UpdateMessageState(ctx context.Context, chatJID, id, state string) error {
	_, err := d.db.ExecContext(ctx, `UPDATE messages SET state=? WHERE chat_jid=? AND id=?`, state, chatJID, id)
	return err
}

// ListMessages returns one chat's messages newest-first,
// keyset-paginated on (ts, id).
func (d *DB) ListMessages(ctx context.Context, chatJID string, limit int, beforeTs int64, beforeID string) ([]bridge.Message, error) {
	var rows *sql.Rows
	var err error
	if beforeTs == 0 {
		rows, err = d.db.QueryContext(ctx, `SELECT chat_jid,id,sender_jid,is_from_me,ts,kind,body,quoted_id,media_path,media_mime,media_size,thumb_path,waveform,state,edited,starred,view_once
			FROM messages WHERE chat_jid=? ORDER BY ts DESC, id DESC LIMIT ?`, chatJID, limit)
	} else {
		rows, err = d.db.QueryContext(ctx, `SELECT chat_jid,id,sender_jid,is_from_me,ts,kind,body,quoted_id,media_path,media_mime,media_size,thumb_path,waveform,state,edited,starred,view_once
			FROM messages WHERE chat_jid=? AND (ts < ? OR (ts = ? AND id < ?)) ORDER BY ts DESC, id DESC LIMIT ?`,
			chatJID, beforeTs, beforeTs, beforeID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bridge.Message
	for rows.Next() {
		var m bridge.Message
		var chat, sender, quoted, mediaPath, mediaMIME, thumb sql.NullString
		var fromMe, edited, starred, vo int
		var ts, size sql.NullInt64
		if err := rows.Scan(&chat, &m.ID, &sender, &fromMe, &ts, &m.Kind, &m.Body, &quoted, &mediaPath, &mediaMIME, &size, &thumb, &m.Waveform, &m.State, &edited, &starred, &vo); err != nil {
			return nil, err
		}
		m.Chat, m.Sender = parseJID(chat.String), parseJID(sender.String)
		m.IsFromMe, m.Edited, m.Starred, m.ViewOnce = fromMe == 1, edited == 1, starred == 1, vo == 1
		m.Ts = time.Unix(ts.Int64, 0)
		m.QuotedID, m.MediaPath, m.MediaMIME, m.ThumbPath = quoted.String, mediaPath.String, mediaMIME.String, thumb.String
		m.MediaSize = size.Int64
		out = append(out, m)
	}
	return out, rows.Err()
}

// parseJID splits "user@server".
func parseJID(s string) bridge.JID {
	if i := strings.IndexByte(s, '@'); i >= 0 {
		return bridge.JID{User: s[:i], Server: s[i+1:]}
	}
	return bridge.JID{User: s}
}

// LastMessage returns a chat's newest message (for list previews).
func (d *DB) LastMessage(ctx context.Context, chatJID string) (bridge.Message, error) {
	var m bridge.Message
	var chat, sender, quoted, mediaPath, mediaMIME, thumb sql.NullString
	var fromMe, edited, starred, vo int
	var ts, size sql.NullInt64
	err := d.db.QueryRowContext(ctx, `SELECT chat_jid,id,sender_jid,is_from_me,ts,kind,body,quoted_id,media_path,media_mime,media_size,thumb_path,waveform,state,edited,starred,view_once
		FROM messages WHERE chat_jid=? ORDER BY ts DESC, id DESC LIMIT 1`, chatJID).Scan(
		&chat, &m.ID, &sender, &fromMe, &ts, &m.Kind, &m.Body, &quoted, &mediaPath, &mediaMIME, &size, &thumb, &m.Waveform, &m.State, &edited, &starred, &vo)
	if err != nil {
		return m, err
	}
	m.Chat, m.Sender = parseJID(chat.String), parseJID(sender.String)
	m.IsFromMe, m.Edited, m.Starred, m.ViewOnce = fromMe == 1, edited == 1, starred == 1, vo == 1
	m.Ts = time.Unix(ts.Int64, 0)
	m.QuotedID, m.MediaPath, m.MediaMIME, m.ThumbPath = quoted.String, mediaPath.String, mediaMIME.String, thumb.String
	m.MediaSize = size.Int64
	return m, nil
}
