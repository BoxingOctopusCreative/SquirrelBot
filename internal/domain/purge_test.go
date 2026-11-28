package domain

import (
	"testing"
	"time"
)

func TestParsePurgeInterval(t *testing.T) {
	hour, err := ParsePurgeInterval("1h", nil)
	if err != nil || hour.Preset != "1h" || hour.Seconds != 3600 {
		t.Fatalf("1h: %+v %v", hour, err)
	}
	for _, key := range []string{"4h", "12h", "24h", "7d"} {
		got, err := ParsePurgeInterval(key, nil)
		if err != nil || got.Preset != key || got.Seconds <= 0 {
			t.Fatalf("%s: %+v %v", key, got, err)
		}
	}
	custom, err := ParsePurgeInterval("", intPtr(90))
	if err != nil || custom.Preset != "" || custom.Seconds != 90*60 {
		t.Fatalf("90 minutes: %+v %v", custom, err)
	}
	same, err := ParsePurgeInterval("", intPtr(60))
	if err != nil || same.Preset != "" || same.Seconds != 3600 {
		t.Fatalf("60 minutes stays custom: %+v %v", same, err)
	}
	if _, err := ParsePurgeInterval("1h", intPtr(60)); err == nil {
		t.Fatal("preset and minutes together should fail")
	}
	if _, err := ParsePurgeInterval("", nil); err == nil {
		t.Fatal("empty interval should fail")
	}
	if _, err := ParsePurgeInterval("2h", nil); err == nil {
		t.Fatal("unknown preset should fail")
	}
	if _, err := ParsePurgeInterval("", intPtr(10)); err == nil {
		t.Fatal("10 minutes should fail")
	}
	if _, err := ParsePurgeInterval("", intPtr(43201)); err == nil {
		t.Fatal("over 30 days should fail")
	}
}

func TestTooOldToBulkDelete(t *testing.T) {
	now := time.Date(2026, 11, 27, 18, 0, 0, 0, time.UTC)
	if TooOldToBulkDelete(now.Add(-12*24*time.Hour), now) {
		t.Fatal("12 days is still deletable")
	}
	if TooOldToBulkDelete(now.Add(-PurgeMaxAge), now) {
		t.Fatal("the cutoff instant matches manual purge and is still deletable")
	}
	if !TooOldToBulkDelete(now.Add(-PurgeMaxAge-time.Second), now) {
		t.Fatal("past the shared cutoff should be skipped")
	}
	if !TooOldToBulkDelete(now.Add(-15*24*time.Hour), now) {
		t.Fatal("15 days should be skipped")
	}
}

func TestPurgeChannelAllowed(t *testing.T) {
	if !PurgeChannelAllowed(0) || !PurgeChannelAllowed(5) {
		t.Fatal("text and announcement channels can be scheduled")
	}
	for _, kind := range []int{2, 4, 15, 16} {
		if PurgeChannelAllowed(kind) {
			t.Fatalf("channel type %d should be rejected", kind)
		}
	}
}

func intPtr(n int) *int { return &n }
