package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// Presence helpers (§9 typing indicator, 3 s debounce by caller).
func typeComposing(paused bool) types.ChatPresence {
	if paused {
		return types.ChatPresencePaused
	}
	return types.ChatPresenceComposing
}

func typeMediaText() types.ChatPresenceMedia { return types.ChatPresenceMediaText }

func chatPresenceComposing() types.ChatPresence { return types.ChatPresenceComposing }

// toWAJID converts a bridge JID to whatsmeow's type.
func toWAJID(j bridge.JID) types.JID {
	return types.JID{User: j.User, Server: j.Server, Device: j.Device}
}

// fromWAJID converts back (device/agent detail folded away).
func fromWAJID(j types.JID) bridge.JID {
	return bridge.JID{User: j.User, Server: j.Server}
}

// textMessage builds a plain (or quoted) text message.
func textMessage(body string, opts bridge.SendOpts) *waE2E.Message {
	if opts.QuotedID == "" {
		return &waE2E.Message{Conversation: proto.String(body)}
	}
	return &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text: proto.String(body),
		ContextInfo: &waE2E.ContextInfo{
			StanzaID: proto.String(opts.QuotedID),
		},
	}}
}

// messageToDTO converts one inbound event (WL-1.4). Pure: unit-tested.
func messageToDTO(evt *events.Message) bridge.Message {
	m := bridge.Message{
		Chat:     fromWAJID(evt.Info.Chat),
		ID:       string(evt.Info.ID),
		Sender:   fromWAJID(evt.Info.Sender),
		IsFromMe: evt.Info.IsFromMe,
		Ts:       evt.Info.Timestamp,
		State:    bridge.StateDelivered,
		ViewOnce: evt.IsViewOnce || evt.IsViewOnceV2,
		Edited:   evt.IsEdit,
	}
	if m.IsFromMe {
		m.State = bridge.StateSent
	}
	m.Kind, m.Body = classify(evt.Message)
	if m.Kind == bridge.MsgEdited {
		m.Edited = true
	}
	return m
}

// classify maps a proto message to (kind, body/caption).
func classify(msg *waE2E.Message) (string, string) {
	if msg == nil {
		return bridge.MsgSystem, ""
	}
	switch {
	case msg.Conversation != nil:
		return bridge.MsgText, msg.GetConversation()
	case msg.ExtendedTextMessage != nil:
		return bridge.MsgText, msg.GetExtendedTextMessage().GetText()
	case msg.ImageMessage != nil:
		return bridge.MsgImage, msg.GetImageMessage().GetCaption()
	case msg.VideoMessage != nil:
		return bridge.MsgVideo, msg.GetVideoMessage().GetCaption()
	case msg.AudioMessage != nil:
		if msg.GetAudioMessage().GetPTT() {
			return bridge.MsgVoice, ""
		}
		return bridge.MsgAudio, ""
	case msg.DocumentMessage != nil:
		return bridge.MsgDoc, msg.GetDocumentMessage().GetCaption()
	case msg.StickerMessage != nil:
		return bridge.MsgSticker, ""
	case msg.LocationMessage != nil:
		loc := msg.GetLocationMessage()
		return bridge.MsgLocation, fmt.Sprintf("%f,%f %s", loc.GetDegreesLatitude(), loc.GetDegreesLongitude(), loc.GetName())
	case msg.LiveLocationMessage != nil:
		return bridge.MsgLiveLocation, msg.GetLiveLocationMessage().GetCaption()
	case msg.ContactMessage != nil:
		return bridge.MsgContact, msg.GetContactMessage().GetDisplayName()
	case msg.ContactsArrayMessage != nil:
		return bridge.MsgContact, msg.GetContactsArrayMessage().GetDisplayName()
	case msg.PollCreationMessage != nil:
		return bridge.MsgPoll, msg.GetPollCreationMessage().GetName()
	case msg.PollUpdateMessage != nil:
		return bridge.MsgPoll, ""
	case msg.ReactionMessage != nil:
		return bridge.MsgReaction, msg.GetReactionMessage().GetText()
	case msg.ProtocolMessage != nil:
		pm := msg.GetProtocolMessage()
		if pm.GetType() == waE2E.ProtocolMessage_REVOKE {
			return bridge.MsgRevoked, ""
		}
		if pm.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT && pm.GetEditedMessage() != nil {
			k, b := classify(pm.GetEditedMessage())
			if k == bridge.MsgText {
				return bridge.MsgEdited, b
			}
			return bridge.MsgEdited, b
		}
		return bridge.MsgSystem, ""
	case msg.CallLogMesssage != nil: // sic: upstream typo
		return bridge.MsgCallLog, ""
	case msg.EventMessage != nil:
		return bridge.MsgEvent, msg.GetEventMessage().GetName()
	default:
		return bridge.MsgSystem, ""
	}
}

// firstDownloadable extracts the downloadable attachment proto, if any.
func firstDownloadable(msg *waE2E.Message) (whatsmeow.DownloadableMessage, bool) {
	if msg == nil {
		return nil, false
	}
	switch {
	case msg.ImageMessage != nil:
		return msg.GetImageMessage(), true
	case msg.VideoMessage != nil:
		return msg.GetVideoMessage(), true
	case msg.AudioMessage != nil:
		return msg.GetAudioMessage(), true
	case msg.DocumentMessage != nil:
		return msg.GetDocumentMessage(), true
	case msg.StickerMessage != nil:
		return msg.GetStickerMessage(), true
	}
	return nil, false
}

// receiptToDTO maps receipt types to message states.
func receiptToDTO(evt *events.Receipt) bridge.ReceiptEvent {
	state := bridge.StateSent
	switch evt.Type {
	case types.ReceiptTypeDelivered, types.ReceiptTypeSender, types.ReceiptTypeServerError:
		state = bridge.StateDelivered
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf:
		state = bridge.StateRead
	case types.ReceiptTypePlayed, types.ReceiptTypePlayedSelf:
		state = bridge.StatePlayed
	}
	ids := make([]string, 0, len(evt.MessageIDs))
	for _, id := range evt.MessageIDs {
		ids = append(ids, string(id))
	}
	return bridge.ReceiptEvent{Chat: fromWAJID(evt.Chat), IDs: ids, State: state, Ts: evt.Timestamp}
}

// groupInfoToDTO turns membership/name changes into a system message event.
func groupInfoToDTO(evt *events.GroupInfo) bridge.MessageEvent {
	var parts []string
	for _, j := range evt.Join {
		parts = append(parts, "+"+j.User+" joined")
	}
	for _, j := range evt.Leave {
		parts = append(parts, "+"+j.User+" left")
	}
	for _, j := range evt.Promote {
		parts = append(parts, "+"+j.User+" promoted")
	}
	for _, j := range evt.Demote {
		parts = append(parts, "+"+j.User+" demoted")
	}
	if evt.Name != nil {
		parts = append(parts, "renamed to "+evt.Name.Name)
	}
	m := bridge.Message{
		Chat: fromWAJID(evt.JID),
		Kind: bridge.MsgSystem,
		Body: strings.Join(parts, "; "),
		Ts:   evt.Timestamp,
	}
	if evt.Sender != nil {
		m.Sender = fromWAJID(*evt.Sender)
	}
	return bridge.MessageEvent{Msg: m}
}

// ingestHistorySync upserts conversations + messages (WL-1.3),
// emitting monotonic 0→100 progress.
func (e *Engine) ingestHistorySync(evt *events.HistorySync) {
	if e.db == nil || evt.Data == nil {
		e.post(bridge.SyncProgress{Percent: 100})
		return
	}
	convs := evt.Data.GetConversations()
	total := len(convs)
	ctx := context.Background()
	for i, c := range convs {
		chat := bridge.Chat{
			JID:      parseWAJID(c.GetID()),
			Title:    c.GetName(),
			Unread:   int(c.GetUnreadCount()),
			Archived: c.GetArchived(),
			Pinned:   c.GetPinned() > 0,
			LastTs:   int64(c.GetConversationTimestamp()),
		}
		if chat.JID.Server == "g.us" {
			chat.Kind = bridge.ChatGroup
		} else {
			chat.Kind = bridge.ChatDM
		}
		if err := e.db.UpsertChat(ctx, chat); err != nil {
			e.log.Error().Err(err).Msg("history chat upsert")
			continue
		}
		for _, hm := range c.GetMessages() {
			if wm := hm.GetMessage(); wm != nil {
				m := webMsgToDTO(chat.JID, wm)
				if err := e.db.UpsertMessage(ctx, m); err != nil {
					e.log.Error().Err(err).Msg("history msg upsert")
				}
				// Learn sender push names from history rows too.
				if push := wm.GetPushName(); push != "" && !m.IsFromMe {
					notePushName(ctx, e.db, m.Sender, push)
				}
			}
		}
		if total > 0 {
			// Upgrade bare DM titles once contact names are learned.
			if chat.JID.Server != "g.us" && looksBare(chat.Title, chat.JID) {
				if better := e.dmTitle(ctx, chat.JID, ""); better != "" {
					chat.Title = better
					_ = e.db.UpsertChat(ctx, chat)
				}
			}
			e.post(bridge.SyncProgress{Percent: (i + 1) * 100 / total})
		}
	}
	e.post(bridge.SyncProgress{Percent: 100})
}

// webMsgToDTO converts a history-sync WebMessageInfo row.
func webMsgToDTO(chat bridge.JID, wm interface {
	GetKey() *waCommon.MessageKey
	GetMessage() *waE2E.Message
	GetMessageTimestamp() uint64
	GetPushName() string
}) bridge.Message {
	key := wm.GetKey()
	m := bridge.Message{
		Chat:     chat,
		ID:       key.GetID(),
		IsFromMe: key.GetFromMe(),
		Ts:       time.Unix(int64(wm.GetMessageTimestamp()), 0),
		State:    bridge.StateDelivered,
	}
	if key.GetParticipant() != "" {
		m.Sender = parseWAJID(key.GetParticipant())
	} else if m.IsFromMe {
		m.Sender = chat
	} else {
		m.Sender = chat
	}
	m.Kind, m.Body = classify(wm.GetMessage())
	return m
}

// parseWAJID parses "user@server" without device detail.
func parseWAJID(s string) bridge.JID {
	if i := strings.IndexByte(s, '@'); i >= 0 {
		return bridge.JID{User: s[:i], Server: s[i+1:]}
	}
	return bridge.JID{User: s}
}

// SendReaction sends (or removes, emoji=="") a reaction (WL-2.4).
func (e *Engine) SendReaction(ctx context.Context, chat bridge.JID, msgID, emoji string) error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("engine not opened")
	}
	key := &waCommon.MessageKey{RemoteJID: proto.String(chat.String()), FromMe: proto.Bool(false), ID: proto.String(msgID)}
	_, err := cli.SendMessage(ctx, toWAJID(chat), &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{
		Key: key, Text: proto.String(emoji), SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
	}})
	return err
}

// SendEdit edits an own text message (WL-2.4).
func (e *Engine) SendEdit(ctx context.Context, chat bridge.JID, msgID, body string) error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("engine not opened")
	}
	key := &waCommon.MessageKey{RemoteJID: proto.String(chat.String()), FromMe: proto.Bool(true), ID: proto.String(msgID)}
	_, err := cli.SendMessage(ctx, toWAJID(chat), &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{
		Key: key, Type: waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
		EditedMessage: textMessage(body, bridge.SendOpts{}),
		TimestampMS:   proto.Int64(time.Now().UnixMilli()),
	}})
	return err
}

// SendRevoke deletes a message for everyone (WL-2.4).
func (e *Engine) SendRevoke(ctx context.Context, chat bridge.JID, msgID string) error {
	e.mu.Lock()
	cli := e.cli
	e.mu.Unlock()
	if cli == nil {
		return fmt.Errorf("engine not opened")
	}
	key := &waCommon.MessageKey{RemoteJID: proto.String(chat.String()), FromMe: proto.Bool(true), ID: proto.String(msgID)}
	_, err := cli.SendMessage(ctx, toWAJID(chat), &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{
		Key: key, Type: waE2E.ProtocolMessage_REVOKE.Enum(),
	}})
	return err
}
