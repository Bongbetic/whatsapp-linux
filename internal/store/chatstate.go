package store

import "context"

// SetChatArchived flips the archived flag (WL-1.5 remote sync).
func (d *DB) SetChatArchived(ctx context.Context, jid string, archived bool) error {
	_, err := d.db.ExecContext(ctx, `UPDATE chats SET archived=? WHERE jid=?`, boolInt(archived), jid)
	return err
}

// SetChatPinned flips the pinned flag (WL-1.5 remote sync).
func (d *DB) SetChatPinned(ctx context.Context, jid string, pinned bool) error {
	_, err := d.db.ExecContext(ctx, `UPDATE chats SET pinned=? WHERE jid=?`, boolInt(pinned), jid)
	return err
}

// SetChatMuted sets muted_until (0 = unmuted, WL-1.5 remote sync).
func (d *DB) SetChatMuted(ctx context.Context, jid string, until int64) error {
	_, err := d.db.ExecContext(ctx, `UPDATE chats SET muted_until=? WHERE jid=?`, until, jid)
	return err
}

// SetMessageStarred flips the star flag (WL-1.5 remote sync).
func (d *DB) SetMessageStarred(ctx context.Context, chatJID, id string, starred bool) error {
	_, err := d.db.ExecContext(ctx, `UPDATE messages SET starred=? WHERE chat_jid=? AND id=?`, boolInt(starred), chatJID, id)
	return err
}

// DeleteMessage removes one message (delete-for-me sync, WL-1.5).
func (d *DB) DeleteMessage(ctx context.Context, chatJID, id string) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid=? AND id=?`, chatJID, id)
	return err
}

// ClearChat removes a chat's messages, keeping the row (WL-1.5).
func (d *DB) ClearChat(ctx context.Context, jid string) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid=?`, jid)
	return err
}

// DeleteChat removes a chat and its messages (WL-1.5).
func (d *DB) DeleteChat(ctx context.Context, jid string) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM messages WHERE chat_jid=?`, jid); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM chats WHERE jid=?`, jid); err != nil {
		return err
	}
	return tx.Commit()
}
