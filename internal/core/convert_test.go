package core

import (
	"testing"
	"time"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

func testInfo() types.MessageInfo {
	return types.MessageInfo{
		Chat:      types.JID{User: "123", Server: "s.whatsapp.net"},
		Sender:    types.JID{User: "123", Server: "s.whatsapp.net"},
		IsFromMe:  false,
		ID:        types.MessageID("ABC123"),
		Timestamp: time.Unix(1700000000, 0),
	}
}

// WL-1.1 accept: unit test for DTO conversion (recorded shapes).
func TestMessageToDTOText(t *testing.T) {
	evt := &events.Message{Info: testInfo(), Message: &waE2E.Message{Conversation: proto.String("hello")}}
	m := messageToDTO(evt)
	if m.Kind != bridge.MsgText || m.Body != "hello" || m.ID != "ABC123" {
		t.Fatalf("got %+v", m)
	}
	if m.Chat.User != "123" || m.State != bridge.StateDelivered {
		t.Fatalf("chat/state: %+v", m)
	}
}

func TestMessageToDTOKinds(t *testing.T) {
	cases := []struct {
		name string
		msg  *waE2E.Message
		kind string
		body string
	}{
		{"image", &waE2E.Message{ImageMessage: &waE2E.ImageMessage{Caption: proto.String("cap")}}, bridge.MsgImage, "cap"},
		{"voice", &waE2E.Message{AudioMessage: &waE2E.AudioMessage{PTT: proto.Bool(true)}}, bridge.MsgVoice, ""},
		{"audio", &waE2E.Message{AudioMessage: &waE2E.AudioMessage{}}, bridge.MsgAudio, ""},
		{"doc", &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{}}, bridge.MsgDoc, ""},
		{"sticker", &waE2E.Message{StickerMessage: &waE2E.StickerMessage{}}, bridge.MsgSticker, ""},
		{"poll", &waE2E.Message{PollCreationMessage: &waE2E.PollCreationMessage{Name: proto.String("Q?")}}, bridge.MsgPoll, "Q?"},
		{"reaction", &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{Text: proto.String("👍")}}, bridge.MsgReaction, "👍"},
		{"revoke", &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{Type: waE2E.ProtocolMessage_REVOKE.Enum()}}, bridge.MsgRevoked, ""},
		{"nil", nil, bridge.MsgSystem, ""},
	}
	for _, c := range cases {
		kind, body := classify(c.msg)
		if kind != c.kind || body != c.body {
			t.Errorf("%s: got (%q,%q) want (%q,%q)", c.name, kind, body, c.kind, c.body)
		}
	}
}

func TestReceiptToDTO(t *testing.T) {
	evt := &events.Receipt{
		MessageSource: types.MessageSource{Chat: types.JID{User: "1", Server: "s.whatsapp.net"}},
		MessageIDs:    []types.MessageID{"A", "B"},
		Type:          types.ReceiptTypeRead,
		Timestamp:     time.Unix(9, 0),
	}
	r := receiptToDTO(evt)
	if r.State != bridge.StateRead || len(r.IDs) != 2 {
		t.Fatalf("got %+v", r)
	}
}
