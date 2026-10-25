// Package bridge is the UI's only view of the backend (plan §5, §7).
//
// It holds plain-data DTOs plus the Engine/Store interfaces. UI code
// must never import core or whatsmeow; core must never leak protobuf
// types through this boundary.
package bridge

import (
	"context"
	"io"
	"time"
)

// App identity (D-7).
const (
	AppID          = "com.bongbetic.WhatsAppLinux"
	DisplayName    = "WhatsApp (unofficial)"
	BinaryName     = "whatsapp-linux"
	DevicePlatform = "linux-desktop"
)

// JID is a WhatsApp address stripped of device details.
type JID struct {
	User   string
	Server string
	Device uint16
}

func (j JID) String() string { return j.User + "@" + j.Server }

// Chat kinds (§6 chats.kind).
const (
	ChatDM              = "dm"
	ChatGroup           = "group"
	ChatCommunity       = "community"
	ChatChannel         = "channel"
	ChatStatusBroadcast = "status-broadcast"
)

// Chat is one row of the chat list.
type Chat struct {
	JID           JID
	Kind          string
	Title         string
	AvatarPath    string
	Unread        int
	Mentions      int
	Archived      bool
	Pinned        bool
	MutedUntil    int64 // unix; -1 = always
	EphemeralSecs int
	LastMsgID     string
	LastTs        int64
	Draft         string
}

// Message kinds (§6 messages.kind).
const (
	MsgText         = "text"
	MsgImage        = "image"
	MsgVideo        = "video"
	MsgAudio        = "audio"
	MsgVoice        = "voice"
	MsgDoc          = "doc"
	MsgSticker      = "sticker"
	MsgLocation     = "location"
	MsgLiveLocation = "live-location"
	MsgContact      = "contact"
	MsgPoll         = "poll"
	MsgEvent        = "event"
	MsgReaction     = "reaction"
	MsgSystem       = "system"
	MsgCallLog      = "call-log"
	MsgRevoked      = "revoked"
	MsgEdited       = "edited"
)

// Message states (§6 messages.state).
const (
	StateSending   = "sending"
	StateSent      = "sent"
	StateDelivered = "delivered"
	StateRead      = "read"
	StatePlayed    = "played"
	StateFailed    = "failed"
)

// Message is one chat message as the UI sees it.
type Message struct {
	Chat      JID
	ID        string
	Sender    JID
	IsFromMe  bool
	Ts        time.Time
	Kind      string
	Body      string
	QuotedID  string
	MediaPath string
	MediaMIME string
	MediaSize int64
	ThumbPath string
	Waveform  []byte
	State     string
	Edited    bool
	Starred   bool
	ViewOnce  bool
	Reactions []Reaction
}

// Reaction is one emoji reaction on a message.
type Reaction struct {
	Sender JID
	Emoji  string
	Ts     time.Time
}

// Contact is one address-book / profile row.
type Contact struct {
	JID         JID
	LID         string
	Phone       string
	PushName    string
	ContactName string
	About       string
	AvatarPath  string
	Blocked     bool
	IsBusiness  bool
}

// GroupMember roles (§6 group_members.role).
const (
	RoleMember     = "member"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "superadmin"
)

// GroupMeta is group metadata for the chat-info drawer.
type GroupMeta struct {
	JID     JID
	Name    string
	Topic   string
	Members []GroupMember
}

// GroupMember is one group membership row.
type GroupMember struct {
	User JID
	Role string
}

// ConnState mirrors §8 reconnect banner states.
type ConnState string

const (
	ConnOffline    ConnState = "offline"
	ConnConnecting ConnState = "connecting"
	ConnConnected  ConnState = "connected"
	ConnLoggedOut  ConnState = "loggedout"
)

// Event is the sealed sum of backend→UI events (§7, §8).
// Every event carries Kind(); no protobuf leaks.
type Event interface{ Kind() string }

// ConnEvent reports connection/banner state (§8).
type ConnEvent struct{ State ConnState }

func (ConnEvent) Kind() string { return "conn" }

// QRCode carries one pairing code image payload (§9 login).
type QRCode struct {
	Code    string // raw QR payload to encode as modules
	Timeout time.Duration
}

func (QRCode) Kind() string { return "qr" }

// PairDone reports a finished pairing attempt.
type PairDone struct{ OK bool }

func (PairDone) Kind() string { return "pair-done" }

// MessageEvent carries a new or updated message row.
type MessageEvent struct{ Msg Message }

func (MessageEvent) Kind() string { return "message" }

// ChatEvent carries a new or updated chat row.
type ChatEvent struct{ Chat Chat }

func (ChatEvent) Kind() string { return "chat" }

// ReceiptEvent carries a delivery/read/play state change.
type ReceiptEvent struct {
	Chat  JID
	IDs   []string
	State string
	Ts    time.Time
}

func (ReceiptEvent) Kind() string { return "receipt" }

// TypingEvent carries presence/typing (droppable at 90% buffer, §8).
type TypingEvent struct {
	Chat   JID
	Sender JID
	Typing bool
}

func (TypingEvent) Kind() string { return "typing" }

// SyncProgress carries history-sync 0→100 progress (WL-1.3).
type SyncProgress struct{ Percent int }

func (SyncProgress) Kind() string { return "sync-progress" }

// RefreshEvent asks the UI to re-list chats/messages (WL-1.5
// app-state sync: pin/mute/archive/star/read changed remotely).
type RefreshEvent struct{}

func (RefreshEvent) Kind() string { return "refresh" }

// CallEvent carries a call-signaling update (Phase 5).
type CallEvent struct {
	ID        string
	Chat      JID
	Direction string // in|out
	Media     string // audio|video
	Stage     string // offering|ringing|in-call|ended|...
}

func (CallEvent) Kind() string { return "call" }

// SendOpts modifies one send.
type SendOpts struct {
	QuotedID string
	Caption  string // media caption (WL-2.3 composer attachments)
}

// MediaFile is one outbound attachment.
type MediaFile struct {
	Path string
	MIME string
	Kind string // image|video|audio|voice|doc|sticker
}

// MediaRef identifies one inbound attachment to download.
type MediaRef struct {
	Chat     JID
	MsgID    string
	Direct   string
	MIME     string
	Size     int64
	FileSHA  []byte
	FileEnc  []byte
	MediaKey []byte
}

// CallLog is one row of the call log (WL-5.3).
type CallLog struct {
	ID        string
	Chat      JID
	Direction string // in|out
	Kind      string // audio|video
	StartedTs int64
	DurationS int64
	Outcome   string // answered|missed|rejected|failed
}

// PrivacySettings maps privacy key → value (WL-4.6), e.g.
// lastseen → "all"|"contacts"|"none". Server-side: read/write
// through Engine, cached read-only in store kv for offline display.
type PrivacySettings map[string]string

// BusinessProfile is a business contact's public profile (WL-4.5).
type BusinessProfile struct {
	JID         JID
	Address     string
	Email       string
	Description string
	Categories  []string
}

// Channel is a followed newsletter's metadata (WL-4.4).
type Channel struct {
	JID         JID
	Name        string
	Description string
	Subscribers int
	Muted       bool
}

// MsgID is an opaque sent-message id.
type MsgID string

// Engine is the only backend API the UI sees (plan §7).
// Phase 5 adds CallPlace/CallAccept/CallReject/CallHangup.
type Engine interface {
	Start(ctx context.Context) error
	Stop() error
	Events() <-chan Event
	PairQR(ctx context.Context) (<-chan QRCode, error)
	PairPhone(ctx context.Context, number string) (string, error)
	Logout(ctx context.Context) error
	SendText(ctx context.Context, chat JID, body string, opts SendOpts) (MsgID, error)
	SendMedia(ctx context.Context, chat JID, f MediaFile, opts SendOpts) (MsgID, error)
	SendReaction(ctx context.Context, chat JID, msgID, emoji string) error
	SendEdit(ctx context.Context, chat JID, msgID, body string) error
	SendRevoke(ctx context.Context, chat JID, msgID string) error
	MarkRead(ctx context.Context, chat JID, ids ...string) error
	SetTyping(ctx context.Context, chat JID, typing bool) error
	DownloadMedia(ctx context.Context, ref MediaRef, w io.Writer) error
	// Social (Phase 4). Groups/communities.
	CreateGroup(ctx context.Context, name string, members []JID) (JID, error)
	GroupMeta(ctx context.Context, group JID) (GroupMeta, error)
	GroupMembers(ctx context.Context, group JID, members []JID, action string) error
	GroupSetName(ctx context.Context, group JID, name string) error
	GroupSetTopic(ctx context.Context, group JID, topic string) error
	GroupInviteLink(ctx context.Context, group JID, reset bool) (string, error)
	GroupJoin(ctx context.Context, code string) (JID, error)
	GroupSubGroups(ctx context.Context, community JID) ([]JID, error)
	GroupLink(ctx context.Context, parent, child JID) error
	GroupUnlink(ctx context.Context, parent, child JID) error
	// Channels (newsletters).
	ChannelFollow(ctx context.Context, ch JID) error
	ChannelUnfollow(ctx context.Context, ch JID) error
	ChannelInfo(ctx context.Context, ch JID) (Channel, error)
	ChannelSetMute(ctx context.Context, ch JID, mute bool) error
	// Status.
	StatusPost(ctx context.Context, text string) error
	// Privacy & safety.
	PrivacyGet(ctx context.Context) (PrivacySettings, error)
	PrivacySet(ctx context.Context, name, value string) error
	BlocklistGet(ctx context.Context) ([]JID, error)
	BlockSet(ctx context.Context, user JID, block bool) error
	DisappearingSet(ctx context.Context, chat JID, secs int64) error
	// Business.
	BusinessProfile(ctx context.Context, biz JID) (BusinessProfile, error)
	// Disappearing default for new chats (WL-4.6).
	DisappearingDefaultSet(ctx context.Context, secs int64) error
	// Calls (Phase 5). Place/Accept need the whatsmeow fork (WL-5.1)
	// and report ErrCallsForkRequired until it lands.
	CallPlace(ctx context.Context, chat JID, video bool) (CallID string, err error)
	CallAccept(ctx context.Context, callID string, video bool) error
	CallReject(ctx context.Context, chat JID, callID string) error
	CallHangup(ctx context.Context, callID string) error
}
