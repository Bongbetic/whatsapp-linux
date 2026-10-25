package core

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// client returns the connected whatsmeow client or a not-opened error.
func (e *Engine) client() (*whatsmeow.Client, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cli == nil {
		return nil, fmt.Errorf("engine not opened")
	}
	return e.cli, nil
}

// CreateGroup creates a group with the given members (WL-4.1).
func (e *Engine) CreateGroup(ctx context.Context, name string, members []bridge.JID) (bridge.JID, error) {
	cli, err := e.client()
	if err != nil {
		return bridge.JID{}, err
	}
	waMembers := make([]types.JID, 0, len(members))
	for _, m := range members {
		waMembers = append(waMembers, toWAJID(m))
	}
	info, err := cli.CreateGroup(ctx, whatsmeow.ReqCreateGroup{Name: name, Participants: waMembers})
	if err != nil {
		return bridge.JID{}, err
	}
	return fromWAJID(info.JID), nil
}

// GroupMeta fetches group metadata into a DTO (WL-4.1).
func (e *Engine) GroupMeta(ctx context.Context, group bridge.JID) (bridge.GroupMeta, error) {
	cli, err := e.client()
	if err != nil {
		return bridge.GroupMeta{}, err
	}
	info, err := cli.GetGroupInfo(ctx, toWAJID(group))
	if err != nil {
		return bridge.GroupMeta{}, err
	}
	return groupInfoToMeta(info), nil
}

// groupInfoToMeta is pure (unit-tested).
func groupInfoToMeta(info *types.GroupInfo) bridge.GroupMeta {
	m := bridge.GroupMeta{JID: fromWAJID(info.JID), Name: info.GroupName.Name, Topic: info.GroupTopic.Topic}
	for _, p := range info.Participants {
		role := bridge.RoleMember
		if p.IsSuperAdmin {
			role = bridge.RoleSuperAdmin
		} else if p.IsAdmin {
			role = bridge.RoleAdmin
		}
		m.Members = append(m.Members, bridge.GroupMember{User: fromWAJID(p.JID), Role: role})
	}
	return m
}

// GroupMembers adds/removes/promotes/demotes members (WL-4.1).
// action is one of add|remove|promote|demote.
func (e *Engine) GroupMembers(ctx context.Context, group bridge.JID, members []bridge.JID, action string) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	var act whatsmeow.ParticipantChange
	switch action {
	case "add":
		act = whatsmeow.ParticipantChangeAdd
	case "remove":
		act = whatsmeow.ParticipantChangeRemove
	case "promote":
		act = whatsmeow.ParticipantChangePromote
	case "demote":
		act = whatsmeow.ParticipantChangeDemote
	default:
		return fmt.Errorf("unknown member action %q", action)
	}
	waMembers := make([]types.JID, 0, len(members))
	for _, m := range members {
		waMembers = append(waMembers, toWAJID(m))
	}
	_, err = cli.UpdateGroupParticipants(ctx, toWAJID(group), waMembers, act)
	return err
}

// GroupSetName renames a group (WL-4.1).
func (e *Engine) GroupSetName(ctx context.Context, group bridge.JID, name string) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.SetGroupName(ctx, toWAJID(group), name)
}

// GroupSetTopic sets the group description (WL-4.1).
func (e *Engine) GroupSetTopic(ctx context.Context, group bridge.JID, topic string) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	info, err := cli.GetGroupInfo(ctx, toWAJID(group))
	if err != nil {
		return err
	}
	return cli.SetGroupTopic(ctx, toWAJID(group), info.GroupTopic.TopicID, "", topic)
}

// GroupInviteLink returns (optionally reset) the invite link (WL-4.1).
func (e *Engine) GroupInviteLink(ctx context.Context, group bridge.JID, reset bool) (string, error) {
	cli, err := e.client()
	if err != nil {
		return "", err
	}
	return cli.GetGroupInviteLink(ctx, toWAJID(group), reset)
}

// GroupJoin joins via invite code (WL-4.1).
func (e *Engine) GroupJoin(ctx context.Context, code string) (bridge.JID, error) {
	cli, err := e.client()
	if err != nil {
		return bridge.JID{}, err
	}
	jid, err := cli.JoinGroupWithLink(ctx, code)
	if err != nil {
		return bridge.JID{}, err
	}
	return fromWAJID(jid), nil
}

// GroupSubGroups lists a community's subgroups (WL-4.2).
func (e *Engine) GroupSubGroups(ctx context.Context, community bridge.JID) ([]bridge.JID, error) {
	cli, err := e.client()
	if err != nil {
		return nil, err
	}
	subs, err := cli.GetSubGroups(ctx, toWAJID(community))
	if err != nil {
		return nil, err
	}
	out := make([]bridge.JID, 0, len(subs))
	for _, s := range subs {
		out = append(out, fromWAJID(s.JID))
	}
	return out, nil
}

// GroupLink/Unlink manage community membership (WL-4.2).
func (e *Engine) GroupLink(ctx context.Context, parent, child bridge.JID) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.LinkGroup(ctx, toWAJID(parent), toWAJID(child))
}

// GroupUnlink removes a subgroup from a community (WL-4.2).
func (e *Engine) GroupUnlink(ctx context.Context, parent, child bridge.JID) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.UnlinkGroup(ctx, toWAJID(parent), toWAJID(child))
}

// ChannelFollow follows a newsletter channel (WL-4.4).
func (e *Engine) ChannelFollow(ctx context.Context, ch bridge.JID) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.FollowNewsletter(ctx, toWAJID(ch))
}

// ChannelUnfollow leaves a channel (WL-4.4).
func (e *Engine) ChannelUnfollow(ctx context.Context, ch bridge.JID) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.UnfollowNewsletter(ctx, toWAJID(ch))
}

// ChannelInfo fetches channel metadata (WL-4.4).
func (e *Engine) ChannelInfo(ctx context.Context, ch bridge.JID) (bridge.Channel, error) {
	cli, err := e.client()
	if err != nil {
		return bridge.Channel{}, err
	}
	meta, err := cli.GetNewsletterInfo(ctx, toWAJID(ch))
	if err != nil {
		return bridge.Channel{}, err
	}
	return newsletterToDTO(meta), nil
}

// newsletterToDTO is pure (unit-tested).
func newsletterToDTO(meta *types.NewsletterMetadata) bridge.Channel {
	c := bridge.Channel{
		JID:         fromWAJID(meta.ID),
		Name:        meta.ThreadMeta.Name.Text,
		Description: meta.ThreadMeta.Description.Text,
		Subscribers: meta.ThreadMeta.SubscriberCount,
	}
	if meta.ViewerMeta != nil {
		c.Muted = meta.ViewerMeta.Mute == types.NewsletterMuteOn
	}
	return c
}

// ChannelSetMute mutes/unmutes a channel (WL-4.4).
func (e *Engine) ChannelSetMute(ctx context.Context, ch bridge.JID, mute bool) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.NewsletterToggleMute(ctx, toWAJID(ch), mute)
}

// StatusPost posts a text status/story (WL-4.3).
func (e *Engine) StatusPost(ctx context.Context, text string) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	_, err = cli.SendMessage(ctx, types.StatusBroadcastJID, textMessage(text, bridge.SendOpts{}))
	return err
}

// PrivacyGet reads server-side privacy settings (WL-4.6).
func (e *Engine) PrivacyGet(ctx context.Context) (bridge.PrivacySettings, error) {
	cli, err := e.client()
	if err != nil {
		return nil, err
	}
	return privacyToDTO(cli.GetPrivacySettings(ctx)), nil
}

// privacyToDTO is pure (unit-tested).
func privacyToDTO(p types.PrivacySettings) bridge.PrivacySettings {
	return bridge.PrivacySettings{
		"groupadd":     string(p.GroupAdd),
		"last":         string(p.LastSeen),
		"status":       string(p.Status),
		"profile":      string(p.Profile),
		"readreceipts": string(p.ReadReceipts),
		"calladd":      string(p.CallAdd),
		"online":       string(p.Online),
		"messages":     string(p.Messages),
		"defense":      string(p.Defense),
		"stickers":     string(p.Stickers),
	}
}

// PrivacySet writes one privacy setting (WL-4.6).
func (e *Engine) PrivacySet(ctx context.Context, name, value string) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	_, err = cli.SetPrivacySetting(ctx, types.PrivacySettingType(name), types.PrivacySetting(value))
	return err
}

// BlocklistGet returns blocked JIDs (WL-4.6).
func (e *Engine) BlocklistGet(ctx context.Context) ([]bridge.JID, error) {
	cli, err := e.client()
	if err != nil {
		return nil, err
	}
	list, err := cli.GetBlocklist(ctx, "")
	if err != nil {
		return nil, err
	}
	jids := make([]bridge.JID, 0, len(list.Items))
	for _, item := range list.Items {
		jid := item.PN
		if jid.IsEmpty() {
			jid = item.LID
		}
		jids = append(jids, fromWAJID(jid))
	}
	return jids, nil
}

// BlockSet blocks/unblocks a user and mirrors the contacts row (WL-4.6).
func (e *Engine) BlockSet(ctx context.Context, user bridge.JID, block bool) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	action := events.BlocklistChangeActionBlock
	if !block {
		action = events.BlocklistChangeActionUnblock
	}
	if _, err := cli.UpdateBlocklist(ctx, toWAJID(user), action, ""); err != nil {
		return err
	}
	if e.db != nil {
		_ = e.db.SetBlocked(ctx, user.String(), block)
	}
	return nil
}

// DisappearingSet sets a chat's disappearing-message timer (WL-4.6).
func (e *Engine) DisappearingSet(ctx context.Context, chat bridge.JID, secs int64) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.SetDisappearingTimer(ctx, toWAJID(chat), time.Duration(secs)*time.Second, time.Now())
}

// DisappearingDefaultSet sets the default disappearing timer (WL-4.6).
func (e *Engine) DisappearingDefaultSet(ctx context.Context, secs int64) error {
	cli, err := e.client()
	if err != nil {
		return err
	}
	return cli.SetDefaultDisappearingTimer(ctx, time.Duration(secs)*time.Second)
}

// BusinessProfile fetches a business profile (WL-4.5).
func (e *Engine) BusinessProfile(ctx context.Context, biz bridge.JID) (bridge.BusinessProfile, error) {
	cli, err := e.client()
	if err != nil {
		return bridge.BusinessProfile{}, err
	}
	p, err := cli.GetBusinessProfile(ctx, toWAJID(biz))
	if err != nil {
		return bridge.BusinessProfile{}, err
	}
	out := bridge.BusinessProfile{JID: fromWAJID(p.JID), Address: p.Address, Email: p.Email}
	for _, c := range p.Categories {
		out.Categories = append(out.Categories, c.Name)
	}
	for k, v := range p.ProfileOptions {
		if k == "description" {
			out.Description = v
		}
	}
	return out, nil
}
