//go:build windows

package main

import (
	"sync/atomic"
	"testing"
	"time"
)

func ensureWatcherTestApp() {
	if app == nil {
		app = &App{language: langEnglish, quit: make(chan struct{})}
	}
}

func TestApplyWatchChanges(t *testing.T) {
	ensureWatcherTestApp()
	oldEntries := app.entries
	oldIndex := quickEntryIndex
	oldCount := atomic.LoadInt64(&indexedCount)
	defer func() {
		app.mu.Lock()
		app.entries = oldEntries
		app.mu.Unlock()
		quickEntryIndex = oldIndex
		atomic.StoreInt64(&indexedCount, oldCount)
	}()

	gen := atomic.LoadInt64(&watchGen)
	e1 := testEntry("keep.txt", `C:\Users\demo\Desktop\keep.txt`, ".txt", 1, time.Now(), false)
	e2 := testEntry("old.txt", `C:\Users\demo\Desktop\old.txt`, ".txt", 1, time.Now(), false)
	e3 := testEntry("other.log", `C:\Other\other.log`, ".log", 1, time.Now(), false)
	app.mu.Lock()
	app.entries = []fileEntry{e3, e1, e2}
	app.mu.Unlock()
	quickEntryIndex = map[string]int{
		e1.LowerPath: 1,
		e2.LowerPath: 2,
	}
	atomic.StoreInt64(&indexedCount, 2)

	add := testEntry("new.txt", `C:\Users\demo\Desktop\new.txt`, ".txt", 2, time.Now(), false)
	applyWatchChanges([]watchChange{{add: &add}}, gen)
	if got := atomic.LoadInt64(&indexedCount); got != 3 {
		t.Fatalf("count after add = %d, want 3", got)
	}
	if idx, ok := quickEntryIndex[add.LowerPath]; !ok || app.entries[idx].LowerPath != add.LowerPath {
		t.Fatal("added entry not indexed")
	}

	replacement := testEntry("keep.txt", `C:\Users\demo\Desktop\keep.txt`, ".txt", 9, time.Now(), false)
	applyWatchChanges([]watchChange{{add: &replacement}}, gen)
	if idx, ok := quickEntryIndex[e1.LowerPath]; !ok || app.entries[idx].Size != 9 {
		t.Fatal("update was not applied")
	}

	applyWatchChanges([]watchChange{{removeKey: e2.LowerPath}}, gen)
	if _, ok := quickEntryIndex[e2.LowerPath]; ok {
		t.Fatal("removed entry still indexed")
	}
	if got := atomic.LoadInt64(&indexedCount); got != 2 {
		t.Fatalf("count after remove = %d, want 2", got)
	}

	child := testEntry("a.txt", `C:\Users\demo\Desktop\sub\a.txt`, ".txt", 1, time.Now(), false)
	applyWatchChanges([]watchChange{{add: &child}}, gen)
	applyWatchChanges([]watchChange{{removeKey: searchFold(`C:\Users\demo\Desktop\sub`)}}, gen)
	if _, ok := quickEntryIndex[child.LowerPath]; ok {
		t.Fatal("child survived directory removal")
	}

	for key, idx := range quickEntryIndex {
		if idx < 0 || idx >= len(app.entries) || app.entries[idx].LowerPath != key {
			t.Fatalf("index invariant broken for %q", key)
		}
	}
}
