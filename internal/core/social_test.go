package core

import (
	"testing"

	"go.mau.fi/whatsmeow/types"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

func TestGroupInfoToMeta(t *testing.T) {
	info := &types.GroupInfo{
		JID:       types.JID{User: "g", Server: "g.us"},
		GroupName: types.GroupName{Name: "Team"},
	}
	info.GroupTopic.Topic = "desc"
	info.Participants = []types.GroupParticipant{
		{JID: types.JID{User: "1", Server: "s.whatsapp.net"}},
		{JID: types.JID{User: "2", Server: "s.whatsapp.net"}, IsAdmin: true},
		{JID: types.JID{User: "3", Server: "s.whatsapp.net"}, IsSuperAdmin: true},
	}
	m := groupInfoToMeta(info)
	if m.Name != "Team" || m.Topic != "desc" || len(m.Members) != 3 {
		t.Fatalf("%+v", m)
	}
	if m.Members[0].Role != bridge.RoleMember || m.Members[1].Role != bridge.RoleAdmin || m.Members[2].Role != bridge.RoleSuperAdmin {
		t.Fatalf("%+v", m.Members)
	}
}

func TestPrivacyToDTO(t *testing.T) {
	p := privacyToDTO(types.PrivacySettings{LastSeen: types.PrivacySettingContacts, ReadReceipts: types.PrivacySettingNone})
	if p["last"] != "contacts" || p["readreceipts"] != "none" {
		t.Fatalf("%v", p)
	}
}

func TestNewsletterToDTO(t *testing.T) {
	meta := &types.NewsletterMetadata{ID: types.JID{User: "n", Server: "newsletter"}}
	meta.ThreadMeta.Name.Text = "News"
	meta.ThreadMeta.SubscriberCount = 42
	c := newsletterToDTO(meta)
	if c.Name != "News" || c.Subscribers != 42 || c.Muted {
		t.Fatalf("%+v", c)
	}
}
