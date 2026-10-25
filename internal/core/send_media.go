package core

import (
	"context"
	"fmt"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"github.com/Bongbetic/whatsapp-linux/internal/bridge"
)

// buildMediaMessage uploads raw bytes and builds the matching
// waE2E message (WL-1.4). kind is one of image|video|audio|voice|doc.
func buildMediaMessage(ctx context.Context, cli *whatsmeow.Client, f bridge.MediaFile, raw []byte, opts bridge.SendOpts) (*waE2E.Message, error) {
	mt := whatsmeow.MediaImage
	switch f.Kind {
	case "video":
		mt = whatsmeow.MediaVideo
	case "audio", "voice":
		mt = whatsmeow.MediaAudio
	case "doc":
		mt = whatsmeow.MediaDocument
	}
	resp, err := cli.Upload(ctx, raw, mt)
	if err != nil {
		return nil, err
	}
	ci := &waE2E.ContextInfo{}
	if opts.QuotedID != "" {
		ci.StanzaID = proto.String(opts.QuotedID)
	}
	mime := f.MIME
	if mime == "" {
		mime = "application/octet-stream"
	}
	length := uint64(len(raw))
	switch f.Kind {
	case "video":
		return &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
			URL: proto.String(resp.URL), DirectPath: proto.String(resp.DirectPath),
			MediaKey: resp.MediaKey, Mimetype: proto.String(mime),
			FileSHA256: resp.FileSHA256, FileEncSHA256: resp.FileEncSHA256,
			FileLength: proto.Uint64(length), Caption: proto.String(opts.Caption),
			ContextInfo: ci,
		}}, nil
	case "audio", "voice":
		return &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
			URL: proto.String(resp.URL), DirectPath: proto.String(resp.DirectPath),
			MediaKey: resp.MediaKey, Mimetype: proto.String(mime),
			FileSHA256: resp.FileSHA256, FileEncSHA256: resp.FileEncSHA256,
			FileLength: proto.Uint64(length), PTT: proto.Bool(f.Kind == "voice"),
			ContextInfo: ci,
		}}, nil
	case "doc":
		return &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
			URL: proto.String(resp.URL), DirectPath: proto.String(resp.DirectPath),
			MediaKey: resp.MediaKey, Mimetype: proto.String(mime),
			FileSHA256: resp.FileSHA256, FileEncSHA256: resp.FileEncSHA256,
			FileLength: proto.Uint64(length), FileName: proto.String(filepath.Base(f.Path)),
			Caption: proto.String(opts.Caption), ContextInfo: ci,
		}}, nil
	case "sticker":
		return &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
			URL: proto.String(resp.URL), DirectPath: proto.String(resp.DirectPath),
			MediaKey: resp.MediaKey, Mimetype: proto.String(mime),
			FileSHA256: resp.FileSHA256, FileEncSHA256: resp.FileEncSHA256,
			FileLength: proto.Uint64(length), ContextInfo: ci,
		}}, nil
	case "image", "":
		return &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
			URL: proto.String(resp.URL), DirectPath: proto.String(resp.DirectPath),
			MediaKey: resp.MediaKey, Mimetype: proto.String(mime),
			FileSHA256: resp.FileSHA256, FileEncSHA256: resp.FileEncSHA256,
			FileLength: proto.Uint64(length), Caption: proto.String(opts.Caption),
			ContextInfo: ci,
		}}, nil
	default:
		return nil, fmt.Errorf("unknown media kind %q", f.Kind)
	}
}
