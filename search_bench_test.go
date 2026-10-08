//go:build windows

package main

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func buildBenchEntries(n int) []fileEntry {
	exts := []string{".pdf", ".txt", ".go", ".png", ".exe", ".log"}
	entries := make([]fileEntry, 0, n)
	for i := 0; i < n; i++ {
		ext := exts[i%len(exts)]
		name := fmt.Sprintf("report_%d%s", i, ext)
		path := fmt.Sprintf(`C:\Users\demo\Documents\folder%d\%s`, i%500, name)
		entries = append(entries, fileEntry{
			Path:      path,
			Name:      name,
			LowerPath: searchFold(path),
			LowerName: searchFold(name),
			LowerExt:  searchFold(ext),
			Size:      int64(i * 1024),
			ModTime:   time.Now().Add(-time.Duration(i) * time.Minute),
			Priority:  i % 6,
		})
	}
	return entries
}

func withBenchEntries(tb testing.TB, entries []fileEntry) func() {
	tb.Helper()
	app.mu.Lock()
	old := app.entries
	app.entries = entries
	app.mu.Unlock()
	return func() {
		app.mu.Lock()
		app.entries = old
		app.mu.Unlock()
	}
}

func TestSearchIndexTopN(t *testing.T) {
	cleanup := withBenchEntries(t, buildBenchEntries(1000))
	defer cleanup()

	results := searchIndex("report ext:pdf", 50, atomic.LoadInt64(&searchGen))
	if len(results) > 50 {
		t.Fatalf("got %d results, want <= 50", len(results))
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	parsed := parseSearchQuery("report ext:pdf")
	for i, e := range results {
		if !matchesQuery(e, parsed) {
			t.Fatalf("result %d does not match the query", i)
		}
		if i > 0 && scoreEntry(results[i-1], parsed) > scoreEntry(e, parsed) {
			t.Fatalf("results are not sorted at index %d", i)
		}
	}
}

func BenchmarkSearchIndex100k(b *testing.B) {
	cleanup := withBenchEntries(b, buildBenchEntries(100000))
	defer cleanup()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = searchIndex("report ext:pdf", 500, atomic.LoadInt64(&searchGen))
	}
}

func BenchmarkSearchFold(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = searchFold("İstanbul Raporları Çözüm Ölçümü Şükran")
	}
}
