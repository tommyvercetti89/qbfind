//go:build windows

package main

import (
	"reflect"
	"testing"
	"time"
)

func TestSearchFold(t *testing.T) {
	cases := map[string]string{
		"İSTANBUL": "istanbul",
		"ĞÜŞİÖÇ":   "gusioc",
		"Iğdır":    "igdir",
		"ABC":      "abc",
		"i̇stanbul": "istanbul",
	}
	for in, want := range cases {
		if got := searchFold(in); got != want {
			t.Errorf("searchFold(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseSearchQuery(t *testing.T) {
	parsed := parseSearchQuery("ext:pdf Report *.TXT")
	if len(parsed.exts) != 2 || parsed.exts[0] != "pdf" || parsed.exts[1] != "txt" {
		t.Fatalf("exts = %v, want [pdf txt]", parsed.exts)
	}
	if len(parsed.tokens) != 1 || parsed.tokens[0] != "report" {
		t.Fatalf("tokens = %v, want [report]", parsed.tokens)
	}
	if parsed.phrase != "report" {
		t.Fatalf("phrase = %q, want report", parsed.phrase)
	}
}

func TestParseExtensionToken(t *testing.T) {
	cases := []struct {
		token string
		want  []string
		ok    bool
	}{
		{"ext:pdf", []string{"pdf"}, true},
		{"extension:docx", []string{"docx"}, true},
		{"*.go", []string{"go"}, true},
		{".png", []string{"png"}, true},
		{"ext:pdf|docx", []string{"pdf", "docx"}, true},
		{"readme.md", nil, false},
		{".", nil, false},
		{"ext:", nil, false},
		{`ext:a\b`, nil, false},
	}
	for _, c := range cases {
		got, ok := parseExtensionToken(c.token)
		if ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseExtensionToken(%q) = (%v,%v), want (%v,%v)", c.token, got, ok, c.want, c.ok)
		}
	}
}

func testEntry(name, path, ext string, size int64, mod time.Time, isDir bool) fileEntry {
	return fileEntry{
		Path:      path,
		Name:      name,
		LowerPath: searchFold(path),
		LowerName: searchFold(name),
		LowerExt:  searchFold(ext),
		Size:      size,
		ModTime:   mod,
		IsDir:     isDir,
	}
}

func TestMatchesQuery(t *testing.T) {
	e := testEntry("Report.pdf", `C:\Users\demo\Documents\Report.pdf`, ".pdf", 2048, time.Now(), false)

	if !matchesQuery(e, parseSearchQuery("report documents")) {
		t.Error("expected multi-token match")
	}
	if matchesQuery(e, parseSearchQuery("invoice")) {
		t.Error("unexpected token match")
	}
	if !matchesQuery(e, parseSearchQuery(".pdf")) {
		t.Error("expected extension match")
	}
	if matchesQuery(e, parseSearchQuery(".png")) {
		t.Error("unexpected extension match")
	}
}

func ensureTestApp() {
	if app == nil {
		app = &App{language: langEnglish, quit: make(chan struct{})}
	}
}

func TestUIStringsComplete(t *testing.T) {
	en := reflect.ValueOf(englishUI)
	tr := reflect.ValueOf(turkishUI)
	for i := 0; i < en.NumField(); i++ {
		name := en.Type().Field(i).Name
		if en.Field(i).String() == "" {
			t.Errorf("englishUI.%s is empty", name)
		}
		if tr.Field(i).String() == "" {
			t.Errorf("turkishUI.%s is empty", name)
		}
	}
}

func TestQueryLanguage(t *testing.T) {
	now := time.Now()
	e := testEntry("Report.pdf", `C:\Users\demo\Documents\Report.pdf`, ".pdf", 5*1024*1024, now.AddDate(0, 0, -1), false)

	if !matchesQuery(e, parseSearchQuery("name:report")) {
		t.Error("name: filter should match")
	}
	if matchesQuery(e, parseSearchQuery("name:invoice")) {
		t.Error("name: filter should not match")
	}
	if !matchesQuery(e, parseSearchQuery("path:documents")) {
		t.Error("path: filter should match")
	}
	if !matchesQuery(e, parseSearchQuery("size:>1mb")) {
		t.Error("size:>1mb should match a 5 MB file")
	}
	if matchesQuery(e, parseSearchQuery("size:<1mb")) {
		t.Error("size:<1mb should not match a 5 MB file")
	}
	if !matchesQuery(e, parseSearchQuery("size:1mb-10mb")) {
		t.Error("size range should match")
	}
	if !matchesQuery(e, parseSearchQuery("date:yesterday")) {
		t.Error("date:yesterday should match a file from yesterday")
	}
	if matchesQuery(e, parseSearchQuery("date:today")) {
		t.Error("date:today should not match a file from yesterday")
	}
	if !matchesQuery(e, parseSearchQuery("*.pdf")) {
		t.Error("glob should match")
	}
	if matchesQuery(e, parseSearchQuery("*.png")) {
		t.Error("glob should not match")
	}
	if !matchesQuery(e, parseSearchQuery("ext:pdf report")) {
		t.Error("combined ext + token should match")
	}
	if !matchesQuery(e, parseSearchQuery("folder:documents")) {
		t.Error("folder: filter should match")
	}
	if !matchesQuery(e, parseSearchQuery("regex:^rep.*\\.pdf$")) {
		t.Error("regex: filter should match")
	}
	if matchesQuery(e, parseSearchQuery("regex:^inv")) {
		t.Error("regex: filter should not match")
	}
	if !matchesQuery(e, parseSearchQuery("report|invoice")) {
		t.Error("OR group should match one alternative")
	}
	if matchesQuery(e, parseSearchQuery("invoice|receipt")) {
		t.Error("OR group should not match when no alternative matches")
	}
	if !matchesQuery(e, parseSearchQuery("ext:pdf|docx")) {
		t.Error("ext OR list should match")
	}
	if matchesQuery(e, parseSearchQuery("ext:png|xlsx")) {
		t.Error("ext OR list should not match")
	}
	if !matchesQuery(e, parseSearchQuery("~rprt")) {
		t.Error("fuzzy subsequence should match")
	}
	if matchesQuery(e, parseSearchQuery("~zzz")) {
		t.Error("fuzzy subsequence should not match")
	}
}

func TestFormatInt(t *testing.T) {
	ensureTestApp()
	app.language = langEnglish
	if got := formatInt(1234567); got != "1,234,567" {
		t.Fatalf("formatInt en = %q", got)
	}
	app.language = langTurkish
	if got := formatInt(1234567); got != "1.234.567" {
		t.Fatalf("formatInt tr = %q", got)
	}
	app.language = langEnglish
}
