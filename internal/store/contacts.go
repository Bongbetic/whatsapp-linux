package store

import (
	"context"
	"database/sql"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// UpsertContact inserts or replaces a contact row.
func (d *DB) UpsertContact(ctx context.Context, c bridge.Contact) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO contacts(jid,lid,phone,push_name,contact_name,about,avatar_path,blocked,is_business)
		VALUES(?,?,?,?,?,?,?,?,?)
		ON CONFLICT(jid) DO UPDATE SET lid=excluded.lid,phone=excluded.phone,push_name=excluded.push_name,
		contact_name=excluded.contact_name,about=excluded.about,avatar_path=excluded.avatar_path,
		blocked=excluded.blocked,is_business=excluded.is_business`,
		c.JID.String(), nullStr(c.LID), nullStr(c.Phone), c.PushName, c.ContactName,
		c.About, nullStr(c.AvatarPath), boolInt(c.Blocked), boolInt(c.IsBusiness))
	return err
}

// SetBlocked flips the blocked flag (WL-4.6).
func (d *DB) SetBlocked(ctx context.Context, jid string, blocked bool) error {
	_, err := d.db.ExecContext(ctx, `UPDATE contacts SET blocked=? WHERE jid=?`, boolInt(blocked), jid)
	return err
}

// ListContacts returns contacts ordered by display name.
func (d *DB) ListContacts(ctx context.Context) ([]bridge.Contact, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT jid,lid,phone,push_name,contact_name,about,avatar_path,blocked,is_business
		FROM contacts ORDER BY COALESCE(NULLIF(contact_name,''),push_name,jid) ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []bridge.Contact
	for rows.Next() {
		var c bridge.Contact
		var j string
		var lid, phone, avatar sql.NullString
		var blocked, biz int
		if err := rows.Scan(&j, &lid, &phone, &c.PushName, &c.ContactName, &c.About, &avatar, &blocked, &biz); err != nil {
			return nil, err
		}
		c.JID = parseJID(j)
		c.LID, c.Phone, c.AvatarPath = lid.String, phone.String, avatar.String
		c.Blocked, c.IsBusiness = blocked == 1, biz == 1
		out = append(out, c)
	}
	return out, rows.Err()
}
