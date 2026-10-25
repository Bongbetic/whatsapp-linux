package store

import (
	"context"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// SetGroupMembers replaces a group's member list in one transaction.
func (d *DB) SetGroupMembers(ctx context.Context, groupJID string, members []bridge.GroupMember) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM group_members WHERE group_jid=?`, groupJID); err != nil {
		return err
	}
	for _, m := range members {
		if _, err := tx.ExecContext(ctx, `INSERT INTO group_members(group_jid,user_jid,role) VALUES(?,?,?)`,
			groupJID, m.User.String(), m.Role); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GroupMembers lists a group's members.
func (d *DB) GroupMembers(ctx context.Context, groupJID string) ([]bridge.GroupMember, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT user_jid,role FROM group_members WHERE group_jid=? ORDER BY user_jid`, groupJID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bridge.GroupMember
	for rows.Next() {
		var m bridge.GroupMember
		var u string
		if err := rows.Scan(&u, &m.Role); err != nil {
			return nil, err
		}
		m.User = parseJID(u)
		out = append(out, m)
	}
	return out, rows.Err()
}

// SavePoll stores a poll definition (WL-2.4).
func (d *DB) SavePoll(ctx context.Context, chatJID, msgID, question, optionsJSON string, selectable int) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO polls(chat_jid,msg_id,question,options_json,selectable) VALUES(?,?,?,?,?)
		ON CONFLICT(chat_jid,msg_id) DO UPDATE SET question=excluded.question,options_json=excluded.options_json,selectable=excluded.selectable`,
		chatJID, msgID, question, optionsJSON, selectable)
	return err
}

// SavePollVote records one voter's ballot.
func (d *DB) SavePollVote(ctx context.Context, chatJID, msgID, voterJID, optionsJSON string, ts int64) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO poll_votes(chat_jid,msg_id,voter_jid,options_json,ts) VALUES(?,?,?,?,?)
		ON CONFLICT(chat_jid,msg_id,voter_jid) DO UPDATE SET options_json=excluded.options_json,ts=excluded.ts`,
		chatJID, msgID, voterJID, optionsJSON, ts)
	return err
}

// LogCall appends a call-log row (WL-5.3).
func (d *DB) LogCall(ctx context.Context, id, chatJID, direction, kind string, startedTs, durationS int64, outcome string) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO calls(id,chat_jid,direction,kind,started_ts,duration_s,outcome) VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET duration_s=excluded.duration_s,outcome=excluded.outcome`,
		id, chatJID, direction, kind, startedTs, durationS, outcome)
	return err
}

// ListCalls returns the call log newest-first (WL-5.3).
func (d *DB) ListCalls(ctx context.Context, limit int) ([]bridge.CallLog, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT id,chat_jid,direction,kind,started_ts,duration_s,outcome FROM calls ORDER BY started_ts DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bridge.CallLog
	for rows.Next() {
		var c bridge.CallLog
		var chat string
		if err := rows.Scan(&c.ID, &chat, &c.Direction, &c.Kind, &c.StartedTs, &c.DurationS, &c.Outcome); err != nil {
			return nil, err
		}
		c.Chat = parseJID(chat)
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetDownload upserts a download-queue row (§8).
func (d *DB) SetDownload(ctx context.Context, chatJID, msgID, state string, done, total int64) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO downloads(chat_jid,msg_id,state,bytes_done,bytes_total) VALUES(?,?,?,?,?)
		ON CONFLICT(chat_jid,msg_id) DO UPDATE SET state=excluded.state,bytes_done=excluded.bytes_done,bytes_total=excluded.bytes_total`,
		chatJID, msgID, state, done, total)
	return err
}

// KVGet/KVPut read and write schema/meta keys (never settings, §6).
func (d *DB) KVGet(ctx context.Context, key string) (string, error) {
	var v string
	err := d.db.QueryRowContext(ctx, `SELECT value FROM kv WHERE key=?`, key).Scan(&v)
	return v, err
}

// KVPut stores a schema/meta key.
func (d *DB) KVPut(ctx context.Context, key, value string) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO kv(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}
