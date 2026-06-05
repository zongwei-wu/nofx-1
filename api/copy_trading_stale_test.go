package api

import (
	"testing"
	"time"
)

func TestLeadOrderTimeToTime(t *testing.T) {
	ms := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).UnixMilli()
	got := leadOrderTimeToTime(ms)
	if !got.Equal(time.UnixMilli(ms)) {
		t.Fatalf("ms parse failed: %v", got)
	}
	sec := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC).Unix()
	got = leadOrderTimeToTime(sec)
	if !got.Equal(time.Unix(sec, 0)) {
		t.Fatalf("sec parse failed: %v", got)
	}
}

func TestIsLeadOrderStaleForAutoFollow(t *testing.T) {
	now := time.Date(2026, 6, 1, 13, 0, 0, 0, time.UTC)
	recent := now.Add(-10 * time.Minute).UnixMilli()
	stale := now.Add(-31 * time.Minute).UnixMilli()

	if isLeadOrderStaleForAutoFollow(recent, now) {
		t.Fatal("10m old order should not be stale")
	}
	if !isLeadOrderStaleForAutoFollow(stale, now) {
		t.Fatal("31m old order should be stale")
	}
	if isLeadOrderStaleForAutoFollow(0, now) {
		t.Fatal("zero order time should not be stale")
	}
}
