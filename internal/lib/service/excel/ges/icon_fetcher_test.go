package ges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIconFetcher_CachesBytes(t *testing.T) {
	var hits atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("fake-png-bytes"))
	}))
	defer ts.Close()

	f := newIconFetcher(2*time.Second, func(code string) string {
		return ts.URL + "/" + code + ".png"
	})

	ctx := context.Background()

	first, err := f.Get(ctx, "10d")
	if err != nil {
		t.Fatalf("first Get: unexpected error: %v", err)
	}
	if string(first) != "fake-png-bytes" {
		t.Fatalf("first Get: bytes = %q, want %q", string(first), "fake-png-bytes")
	}

	second, err := f.Get(ctx, "10d")
	if err != nil {
		t.Fatalf("second Get: unexpected error: %v", err)
	}
	if string(second) != "fake-png-bytes" {
		t.Fatalf("second Get: bytes = %q, want %q", string(second), "fake-png-bytes")
	}

	if got := hits.Load(); got != 1 {
		t.Errorf("server hits = %d, want 1 (second call should hit cache)", got)
	}
}

func TestIconFetcher_PropagatesError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	f := newIconFetcher(2*time.Second, func(code string) string {
		return ts.URL + "/" + code + ".png"
	})

	_, err := f.Get(context.Background(), "99x")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestIconFetcher_RespectsTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("too-late"))
	}))
	defer ts.Close()

	f := newIconFetcher(50*time.Millisecond, func(code string) string {
		return ts.URL + "/" + code + ".png"
	})

	start := time.Now()
	_, err := f.Get(context.Background(), "10d")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "deadline") && !strings.Contains(msg, "timeout") {
		t.Errorf("error = %q, want to contain 'deadline' or 'timeout'", err.Error())
	}
	if elapsed > 400*time.Millisecond {
		t.Errorf("Get took %v, expected to fail near 50ms timeout", elapsed)
	}
}
