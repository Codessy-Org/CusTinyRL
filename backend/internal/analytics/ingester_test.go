package analytics

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestClassifyDevice(t *testing.T) {
	cases := map[string]string{
		"": "",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120": "desktop",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)":                        "mobile",
		"Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X)":                                 "tablet",
		"Mozilla/5.0 (compatible; Googlebot/2.1)":                                       "bot",
		"curl/8.4.0": "bot", // HTTP clients are classified as non-human
	}
	for ua, want := range cases {
		if got := ClassifyDevice(ua); got != want {
			t.Fatalf("ClassifyDevice(%q) = %q, want %q", ua, got, want)
		}
	}
}

func TestSanitizeReferrer(t *testing.T) {
	cases := map[string]string{
		"":                               "",
		"https://Example.com/path?q=1#f": "example.com",
		"http://twitter.com/x":           "twitter.com",
		"nonsense":                       "nonsense",
	}
	for in, want := range cases {
		if got := SanitizeReferrer(in); got != want {
			t.Fatalf("SanitizeReferrer(%q) = %q, want %q", in, got, want)
		}
	}
}

type memStore struct {
	mu      sync.Mutex
	batches [][]Event
	fail    bool
}

func (m *memStore) InsertBatch(ctx context.Context, events []Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail {
		return context.DeadlineExceeded
	}
	cp := make([]Event, len(events))
	copy(cp, events)
	m.batches = append(m.batches, cp)
	return nil
}

func (m *memStore) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, b := range m.batches {
		n += len(b)
	}
	return n
}

func TestIngesterFlushesAndDrains(t *testing.T) {
	store := &memStore{}
	ing := NewIngester(store, 64, 8, 50*time.Millisecond, nil)
	ctx, cancel := context.WithCancel(context.Background())
	go ing.Run(ctx)

	for i := 0; i < 20; i++ {
		ing.Record(Event{ShortURLID: 1, ClickedAt: time.Now()}, func() { t.Error("dropped event") })
	}
	// Wait for ticker flushes.
	deadline := time.After(2 * time.Second)
	for store.count() < 20 {
		select {
		case <-deadline:
			t.Fatalf("timed out; got %d events", store.count())
		default:
			time.Sleep(20 * time.Millisecond)
		}
	}
	cancel()
	if !ing.Wait(time.Second) {
		t.Fatal("ingester did not stop")
	}
}

func TestIngesterDropsWhenFull(t *testing.T) {
	store := &memStore{}
	// batchSize larger than buffer forces buffer to stay full without a runner.
	ing := NewIngester(store, 2, 2, time.Hour, nil)
	var dropped int
	for i := 0; i < 10; i++ {
		ing.Record(Event{ShortURLID: 1}, func() { dropped++ })
	}
	if dropped != 8 {
		t.Fatalf("dropped = %d, want 8", dropped)
	}
}
