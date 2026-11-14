package domain

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"unicode/utf8"
)

const (
	MaxNickLen     = 32
	MaxAvatarBytes = 1 << 20
	MaxAvatarEdge  = 4096
)

// CleanNick trims a per-server display name. An empty result clears the nickname.
func CleanNick(nick string) (string, error) {
	nick = strings.TrimSpace(nick)
	if nick == "" {
		return "", nil
	}
	if utf8.RuneCountInString(nick) > MaxNickLen {
		return "", fmt.Errorf("names are 32 characters or fewer")
	}
	for _, r := range nick {
		if r < 0x20 || r == 0x7f || strings.ContainsRune("@#:`", r) {
			return "", fmt.Errorf("names cannot include @, #, :, or backticks")
		}
	}
	return nick, nil
}

// NormalizeAvatar checks a picture and returns a freshly encoded PNG, JPEG, or GIF.
func NormalizeAvatar(data []byte) ([]byte, string, error) {
	if len(data) == 0 {
		return nil, "", fmt.Errorf("choose a picture")
	}
	if len(data) > MaxAvatarBytes {
		return nil, "", fmt.Errorf("pictures must be 1 MB or smaller")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > MaxAvatarEdge || cfg.Height > MaxAvatarEdge {
		return nil, "", fmt.Errorf("pictures must be 4096 pixels or smaller on each side")
	}
	var buf bytes.Buffer
	switch format {
	case "png":
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
		}
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
		}
		format = "image/png"
	case "jpeg":
		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
			return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
		}
		format = "image/jpeg"
	case "gif":
		img, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
		}
		if err := gif.EncodeAll(&buf, img); err != nil {
			return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
		}
		format = "image/gif"
	default:
		return nil, "", fmt.Errorf("use a PNG, JPEG, or GIF")
	}
	if buf.Len() == 0 || buf.Len() > MaxAvatarBytes {
		return nil, "", fmt.Errorf("pictures must be 1 MB or smaller")
	}
	return buf.Bytes(), format, nil
}
