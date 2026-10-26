package core

import (
	"context"
	"database/sql"

	"go.mau.fi/whatsmeow/types"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
	appstore "github.com/Bongbetic/whatsapp-linux/internal/store"
)

// Contact learning (live-account fix): the device store already syncs
// push names, business names and phone-book contact names from the
// phone via app-state. The engine previously ignored all of them, so
// our contacts table stayed empty and chat titles fell back to raw
// JIDs/phone numbers. These helpers merge every signal into app.db.

// displayName picks the best visible name: phone-book name first,
// then push name, then the stored fallback (JID or phone number).
func displayName(contactName, pushName, fallback string) string {
	if contactName != "" {
		return contactName
	}
	if pushName != "" {
		return pushName
	}
	return fallback
}

// DisplayName resolves the visible name for a DM contact row.
func DisplayName(c bridge.Contact, fallback string) string {
	return displayName(c.ContactName, c.PushName, fallback)
}

// notePushName records a sender's push name, preserving any
// phone-book contact name already stored.
func notePushName(ctx context.Context, db *appstore.DB, jid bridge.JID, push string) {
	if db == nil || push == "" {
		return
	}
	c, err := db.GetContact(ctx, jid.String())
	if err != nil && err != sql.ErrNoRows {
		return
	}
	if c.PushName == push {
		return
	}
	c.JID = jid
	c.PushName = push
	_ = db.UpsertContact(ctx, c)
}

// noteContactAction records a phone-book contact change
// (events.Contact), preserving the learned push name.
func noteContactAction(ctx context.Context, db *appstore.DB, jid bridge.JID, fullName string) {
	if db == nil || fullName == "" {
		return
	}
	c, err := db.GetContact(ctx, jid.String())
	if err != nil && err != sql.ErrNoRows {
		return
	}
	if c.ContactName == fullName {
		return
	}
	c.JID = jid
	c.ContactName = fullName
	_ = db.UpsertContact(ctx, c)
}

// noteBusinessName flags a business contact and records its verified name.
func noteBusinessName(ctx context.Context, db *appstore.DB, jid bridge.JID, name string) {
	if db == nil || name == "" {
		return
	}
	c, err := db.GetContact(ctx, jid.String())
	if err != nil && err != sql.ErrNoRows {
		return
	}
	if c.ContactName == name && c.IsBusiness {
		return
	}
	c.JID = jid
	c.ContactName = name
	c.IsBusiness = true
	_ = db.UpsertContact(ctx, c)
}

// contactFromDevice maps a whatsmeow device-store contact to our row.
func contactFromDevice(jid types.JID, info types.ContactInfo) bridge.Contact {
	name := info.FullName
	if name == "" {
		name = info.BusinessName
	}
	return bridge.Contact{
		JID:         fromWAJID(jid),
		Phone:       phoneOf(jid),
		PushName:    info.PushName,
		ContactName: name,
		IsBusiness:  info.BusinessName != "",
	}
}

// phoneOf returns the dialable number for a PN JID, else "".
func phoneOf(jid types.JID) string {
	if jid.Server == types.DefaultUserServer || jid.Server == types.LegacyUserServer {
		return "+" + jid.User
	}
	return ""
}

// syncContactsFromDevice bulk-copies the device store's synced
// contacts (push names, business names, phone-book names from the
// phone's app-state) into app.db. Best-effort: runs once per
// connect; live events keep rows fresh afterwards.
func (e *Engine) syncContactsFromDevice(ctx context.Context) {
	cli, err := e.client()
	if err != nil || e.db == nil {
		return
	}
	all, err := cli.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		e.log.Error().Err(err).Msg("sync device contacts")
		return
	}
	for jid, info := range all {
		if !info.Found {
			continue
		}
		c := contactFromDevice(jid, info)
		existing, err := e.db.GetContact(ctx, c.JID.String())
		if err == nil {
			// Merge: keep the richer side of each field.
			if existing.ContactName != "" && c.ContactName == "" {
				c.ContactName = existing.ContactName
			}
			if existing.PushName != "" && c.PushName == "" {
				c.PushName = existing.PushName
			}
			if existing.Phone != "" {
				c.Phone = existing.Phone
			}
			if existing.About != "" {
				c.About = existing.About
			}
			if existing.AvatarPath != "" {
				c.AvatarPath = existing.AvatarPath
			}
			c.Blocked = c.Blocked || existing.Blocked
			c.IsBusiness = c.IsBusiness || existing.IsBusiness
		} else if err != sql.ErrNoRows {
			continue
		}
		if c.ContactName == "" && c.PushName == "" {
			continue
		}
		_ = e.db.UpsertContact(ctx, c)
	}
}

// dmTitle resolves a DM chat's title from contacts, falling back to
// the stored title (or phone number) when nothing is known yet.
func (e *Engine) dmTitle(ctx context.Context, jid bridge.JID, stored string) string {
	if e.db == nil {
		return stored
	}
	if c, err := e.db.GetContact(ctx, jid.String()); err == nil {
		if n := DisplayName(c, stored); n != "" {
			return n
		}
	}
	if stored != "" {
		return stored
	}
	if p := phoneOf(toWAJID(jid)); p != "" {
		return p
	}
	return jid.String()
}
