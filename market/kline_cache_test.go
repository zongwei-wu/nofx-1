package market

import (
	"testing"
	"time"
)

func TestIsKlineCacheStale(t *testing.T) {
	now := time.Now().UnixMilli()
	fresh := []Kline{{CloseTime: now}}
	if isKlineCacheStale(fresh, "3m") {
		t.Fatal("fresh klines should not be stale")
	}

	stale := []Kline{{CloseTime: now - int64(10*time.Minute/time.Millisecond)}}
	if !isKlineCacheStale(stale, "3m") {
		t.Fatal("old klines should be stale")
	}
	if !isKlineCacheStale(nil, "3m") {
		t.Fatal("empty klines should be stale")
	}
}
