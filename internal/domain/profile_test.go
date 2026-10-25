package domain

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func TestCleanNick(t *testing.T) {
	nick, err := CleanNick("  Canopy  ")
	if err != nil || nick != "Canopy" {
		t.Fatalf("trim: %q %v", nick, err)
	}
	nick, err = CleanNick("   ")
	if err != nil || nick != "" {
		t.Fatalf("blank: %q %v", nick, err)
	}
	if _, err := CleanNick(strings.Repeat("a", 33)); err == nil {
		t.Fatal("expected a length error")
	}
	if _, err := CleanNick("mod@here"); err == nil {
		t.Fatal("expected a character error")
	}
	if _, err := CleanNick("line\nbreak"); err == nil {
		t.Fatal("expected a control character error")
	}
}

func TestAvatarImage(t *testing.T) {
	kind := mustNormalize(t, encodePNG(t, 8, 8), "image/png")
	if _, decoded, err := image.Decode(bytes.NewReader(kind)); err != nil || decoded != "png" {
		t.Fatalf("png decode: %s %v", decoded, err)
	}
	mustNormalize(t, encodeJPEG(t), "image/jpeg")
	mustNormalize(t, encodeGIF(t), "image/gif")
	if _, _, err := NormalizeAvatar([]byte("<svg></svg>")); err == nil {
		t.Fatal("svg should be rejected")
	}
	if _, _, err := NormalizeAvatar(bytes.Repeat([]byte{0x89}, MaxAvatarBytes+1)); err == nil {
		t.Fatal("oversize should be rejected")
	}
	if _, _, err := NormalizeAvatar(encodePNG(t, MaxAvatarEdge+1, 1)); err == nil {
		t.Fatal("wide image should be rejected")
	}
}

func mustNormalize(t *testing.T, data []byte, want string) []byte {
	t.Helper()
	clean, kind, err := NormalizeAvatar(data)
	if err != nil || kind != want || len(clean) == 0 {
		t.Fatalf("%s: %q %d %v", want, kind, len(clean), err)
	}
	return clean
}

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeGIF(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	palette := color.Palette{color.Black, color.White}
	if err := gif.Encode(&buf, image.NewPaletted(image.Rect(0, 0, 8, 8), palette), nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
