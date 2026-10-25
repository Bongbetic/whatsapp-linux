package store

import (
	"context"
	"database/sql"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// UpsertChat inserts or replaces a chat row.
func (d *DB) UpsertChat(ctx context.Context, c bridge.Chat) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO chats(jid,kind,title,avatar_path,unread,mentions,archived,pinned,muted_until,ephemeral_secs,last_msg_id,last_ts,draft)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(jid) DO UPDATE SET kind=excluded.kind,title=excluded.title,avatar_path=excluded.avatar_path,
		unread=excluded.unread,mentions=excluded.mentions,archived=excluded.archived,pinned=excluded.pinned,
		muted_until=excluded.muted_until,ephemeral_secs=excluded.ephemeral_secs,last_msg_id=excluded.last_msg_id,
		last_ts=excluded.last_ts,draft=excluded.draft`,
		c.JID.String(), c.Kind, c.Title, nullStr(c.AvatarPath), c.Unread, c.Mentions,
		boolInt(c.Archived), boolInt(c.Pinned), c.MutedUntil, c.EphemeralSecs,
		nullStr(c.LastMsgID), c.LastTs, c.Draft)
	return err
}

// GetChat fetches one chat by JID string.
func (d *DB) GetChat(ctx context.Context, jid string) (bridge.Chat, error) {
	var c bridge.Chat
	var j, avatar, lastID sql.NullString
	var unread, mentions, archived, pinned int
	var muted, eph int64
	var lastTs int64
	err := d.db.QueryRowContext(ctx, `SELECT jid,kind,title,avatar_path,unread,mentions,archived,pinned,muted_until,ephemeral_secs,last_msg_id,last_ts,draft
		FROM chats WHERE jid=?`, jid).Scan(&j, &c.Kind, &c.Title, &avatar, &unread, &mentions, &archived, &pinned, &muted, &eph, &lastID, &lastTs, &c.Draft)
	if err != nil {
		return c, err
	}
	c.JID = parseJID(j.String)
	c.AvatarPath, c.LastMsgID = avatar.String, lastID.String
	c.Unread, c.Mentions = unread, mentions
	c.Archived, c.Pinned = archived == 1, pinned == 1
	c.MutedUntil, c.EphemeralSecs, c.LastTs = muted, int(eph), lastTs
	return c, nil
}

// ListChats returns chats newest-first, keyset-paginated on
// (last_ts, jid); afterTs/afterJID select the next page
// (0, "" = first page).
func (d *DB) ListChats(ctx context.Context, limit int, afterTs int64, afterJID string, includeArchived bool) ([]bridge.Chat, error) {
	if afterTs == 0 {
		afterTs = 1<<63 - 1
	}
	q := `SELECT jid,kind,title,avatar_path,unread,mentions,archived,pinned,muted_until,ephemeral_secs,last_msg_id,last_ts,draft
		FROM chats WHERE (last_ts < ? OR (last_ts = ? AND jid > ?))`
	if !includeArchived {
		q += ` AND archived=0`
	}
	q += ` ORDER BY last_ts DESC, jid ASC LIMIT ?`
	rows, err := d.db.QueryContext(ctx, q, afterTs, afterTs, afterJID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bridge.Chat
	for rows.Next() {
		var c bridge.Chat
		var j, avatar, lastID sql.NullString
		var unread, mentions, archived, pinned int
		var muted int64
		var eph int64
		var lastTs int64
		if err := rows.Scan(&j, &c.Kind, &c.Title, &avatar, &unread, &mentions, &archived, &pinned, &muted, &eph, &lastID, &lastTs, &c.Draft); err != nil {
			return nil, err
		}
		c.JID = parseJID(j.String)
		c.AvatarPath, c.LastMsgID = avatar.String, lastID.String
		c.Unread, c.Mentions = unread, mentions
		c.Archived, c.Pinned = archived == 1, pinned == 1
		c.MutedUntil, c.EphemeralSecs, c.LastTs = muted, int(eph), lastTs
		out = append(out, c)
	}
	return out, rows.Err()
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
