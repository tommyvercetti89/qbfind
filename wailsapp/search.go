//go:build windows

package main

import (
	"container/heap"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type searchQuery struct {
	tokens      []string
	alternates  [][]string
	names       []string
	paths       []string
	folders     []string
	exts        []string
	globs       []string
	fuzzy       []string
	nameRegexes []*regexp.Regexp
	pathRegexes []*regexp.Regexp
	phrase      string
	sizeMin     int64
	sizeMax     int64
	dateMin     int64
	dateMax     int64
}

func (q searchQuery) empty() bool {
	return len(q.tokens) == 0 && len(q.alternates) == 0 && len(q.names) == 0 && len(q.paths) == 0 &&
		len(q.folders) == 0 && len(q.exts) == 0 && len(q.globs) == 0 && len(q.fuzzy) == 0 &&
		len(q.nameRegexes) == 0 && len(q.pathRegexes) == 0 &&
		q.sizeMin < 0 && q.sizeMax < 0 && q.dateMin < 0 && q.dateMax < 0
}

func matchAnyPart(s, value string) bool {
	for _, part := range strings.Split(value, "|") {
		if part != "" && strings.Contains(s, part) {
			return true
		}
	}
	return false
}

func isSubsequence(needle, hay string) bool {
	if needle == "" {
		return true
	}
	i := 0
	for j := 0; j < len(hay) && i < len(needle); j++ {
		if hay[j] == needle[i] {
			i++
		}
	}
	return i == len(needle)
}

type rankedEntry struct {
	entry fileEntry
	score int
}

type scoreHeap []rankedEntry

func worseEntry(a, b rankedEntry) bool {
	if a.score != b.score {
		return a.score > b.score
	}
	return a.entry.LowerPath > b.entry.LowerPath
}

func (h scoreHeap) Len() int            { return len(h) }
func (h scoreHeap) Less(i, j int) bool  { return worseEntry(h[i], h[j]) }
func (h scoreHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *scoreHeap) Push(x interface{}) { *h = append(*h, x.(rankedEntry)) }
func (h *scoreHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func rankedResults(h scoreHeap) []fileEntry {
	sort.Slice(h, func(i, j int) bool {
		if h[i].score != h[j].score {
			return h[i].score < h[j].score
		}
		return h[i].entry.LowerPath < h[j].entry.LowerPath
	})
	results := make([]fileEntry, len(h))
	for i := range h {
		results[i] = h[i].entry
	}
	return results
}

func searchIndex(query string, limit int) []fileEntry {
	parsed := parseSearchQuery(query)
	if parsed.empty() {
		return nil
	}

	app.mu.RLock()
	entries := app.entries
	app.mu.RUnlock()

	if limit <= 0 {
		return nil
	}
	h := make(scoreHeap, 0, limit)

	for _, e := range entries {
		if !matchesQuery(e, parsed) {
			continue
		}
		item := rankedEntry{entry: e, score: scoreEntry(e, parsed)}
		if len(h) < limit {
			heap.Push(&h, item)
			continue
		}
		if worseEntry(item, h[0]) {
			continue
		}
		h[0] = item
		heap.Fix(&h, 0)
	}
	return rankedResults(h)
}

func parseSearchQuery(query string) searchQuery {
	rawTokens := strings.Fields(query)
	parsed := searchQuery{sizeMin: -1, sizeMax: -1, dateMin: -1, dateMax: -1}
	for _, raw := range rawTokens {
		lower := strings.ToLower(raw)
		if strings.HasPrefix(lower, "regex:") && len(raw) > 6 {
			if re, err := regexp.Compile("(?i)" + raw[6:]); err == nil {
				parsed.nameRegexes = append(parsed.nameRegexes, re)
			}
			continue
		}
		if strings.HasPrefix(lower, "pathregex:") && len(raw) > 10 {
			if re, err := regexp.Compile("(?i)" + raw[10:]); err == nil {
				parsed.pathRegexes = append(parsed.pathRegexes, re)
			}
			continue
		}
		token := searchFold(raw)
		if exts, ok := parseExtensionToken(token); ok {
			parsed.exts = append(parsed.exts, exts...)
			continue
		}
		if value, ok := strings.CutPrefix(token, "name:"); ok && value != "" {
			parsed.names = append(parsed.names, value)
			continue
		}
		if value, ok := strings.CutPrefix(token, "path:"); ok && value != "" {
			parsed.paths = append(parsed.paths, value)
			continue
		}
		if value, ok := strings.CutPrefix(token, "folder:"); ok && value != "" {
			parsed.folders = append(parsed.folders, value)
			continue
		}
		if value, ok := strings.CutPrefix(token, "size:"); ok {
			if min, max, ok := parseSizeRange(value); ok {
				parsed.sizeMin, parsed.sizeMax = min, max
				continue
			}
		}
		if value, ok := strings.CutPrefix(token, "date:"); ok {
			if min, max, ok := parseDateRange(value, time.Now()); ok {
				parsed.dateMin, parsed.dateMax = min, max
				continue
			}
		}
		if strings.HasPrefix(token, "~") && len(token) > 1 {
			parsed.fuzzy = append(parsed.fuzzy, strings.TrimPrefix(token, "~"))
			continue
		}
		if strings.Contains(token, "|") {
			var group []string
			for _, part := range strings.Split(token, "|") {
				if part != "" {
					group = append(group, part)
				}
			}
			if len(group) > 1 {
				parsed.alternates = append(parsed.alternates, group)
				continue
			}
		}
		if strings.ContainsAny(token, "*?") {
			parsed.globs = append(parsed.globs, token)
			continue
		}
		parsed.tokens = append(parsed.tokens, token)
	}
	parsed.phrase = strings.Join(parsed.tokens, " ")
	return parsed
}

func parseSizeRange(s string) (int64, int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	switch {
	case strings.HasPrefix(s, ">="):
		if v, ok := parseSize(strings.TrimPrefix(s, ">=")); ok {
			return v, -1, true
		}
	case strings.HasPrefix(s, "<="):
		if v, ok := parseSize(strings.TrimPrefix(s, "<=")); ok {
			return 0, v, true
		}
	case strings.HasPrefix(s, ">"):
		if v, ok := parseSize(strings.TrimPrefix(s, ">")); ok {
			return v + 1, -1, true
		}
	case strings.HasPrefix(s, "<"):
		if v, ok := parseSize(strings.TrimPrefix(s, "<")); ok && v > 0 {
			return 0, v - 1, true
		}
	}
	if i := strings.Index(s, "-"); i > 0 {
		min, ok1 := parseSize(s[:i])
		max, ok2 := parseSize(s[i+1:])
		if !ok1 || !ok2 || max < min {
			return 0, 0, false
		}
		return min, max, true
	}
	if v, ok := parseSize(s); ok {
		return v, v, true
	}
	return 0, 0, false
}

func parseSize(s string) (int64, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "tb"):
		mult = 1 << 40
		s = strings.TrimSuffix(s, "tb")
	case strings.HasSuffix(s, "gb"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "gb")
	case strings.HasSuffix(s, "mb"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "mb")
	case strings.HasSuffix(s, "kb"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "b"):
		s = strings.TrimSuffix(s, "b")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return int64(f * float64(mult)), true
}

func parseDateRange(s string, now time.Time) (int64, int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, false
	}
	if strings.HasPrefix(s, ">=") || strings.HasPrefix(s, "<=") || strings.HasPrefix(s, ">") || strings.HasPrefix(s, "<") {
		op := s[:1]
		rest := s[1:]
		if strings.HasPrefix(s, ">=") || strings.HasPrefix(s, "<=") {
			op = s[:2]
			rest = s[2:]
		}
		start, end, ok := parseDateSpan(rest, now)
		if !ok {
			return 0, 0, false
		}
		switch op {
		case ">=":
			return start, -1, true
		case ">":
			return end + 1, -1, true
		case "<=":
			return 0, end, true
		default: // "<"
			if start <= 0 {
				return 0, 0, false
			}
			return 0, start - 1, true
		}
	}
	start, end, ok := parseDateSpan(s, now)
	if !ok {
		return 0, 0, false
	}
	return start, end, true
}

func parseDateSpan(s string, now time.Time) (int64, int64, bool) {
	loc := now.Location()
	dayStart := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	}
	switch s {
	case "today":
		start := dayStart(now)
		return start.Unix(), start.AddDate(0, 0, 1).Unix() - 1, true
	case "yesterday":
		start := dayStart(now).AddDate(0, 0, -1)
		return start.Unix(), start.AddDate(0, 0, 1).Unix() - 1, true
	case "week":
		return dayStart(now).AddDate(0, 0, -7).Unix(), -1, true
	case "month":
		return dayStart(now).AddDate(0, -1, 0).Unix(), -1, true
	}
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return t.Unix(), t.AddDate(0, 0, 1).Unix() - 1, true
	}
	if t, err := time.ParseInLocation("2006-01", s, loc); err == nil {
		return t.Unix(), t.AddDate(0, 1, 0).Unix() - 1, true
	}
	if t, err := time.ParseInLocation("2006", s, loc); err == nil {
		return t.Unix(), t.AddDate(1, 0, 0).Unix() - 1, true
	}
	return 0, 0, false
}

func parseExtensionToken(token string) ([]string, bool) {
	switch {
	case strings.HasPrefix(token, "extension:"):
		token = strings.TrimPrefix(token, "extension:")
	case strings.HasPrefix(token, "ext:"):
		token = strings.TrimPrefix(token, "ext:")
	case strings.HasPrefix(token, "*."):
		token = strings.TrimPrefix(token, "*.")
	case strings.HasPrefix(token, ".") && len(token) > 1:
		token = strings.TrimPrefix(token, ".")
	default:
		return nil, false
	}
	var out []string
	for _, part := range strings.Split(token, "|") {
		part = strings.Trim(part, ". ")
		if part == "" || strings.ContainsAny(part, `\/:*?"<>|`) {
			continue
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func matchesQuery(e fileEntry, query searchQuery) bool {
	if len(query.exts) > 0 {
		ext := strings.TrimPrefix(e.LowerExt, ".")
		if ext == "" {
			return false
		}
		matchedExt := false
		for _, wanted := range query.exts {
			if ext == wanted {
				matchedExt = true
				break
			}
		}
		if !matchedExt {
			return false
		}
	}
	for _, want := range query.names {
		if !matchAnyPart(e.LowerName, want) {
			return false
		}
	}
	for _, want := range query.paths {
		if !matchAnyPart(e.LowerPath, want) {
			return false
		}
	}
	for _, want := range query.folders {
		if !matchAnyPart(e.LowerPath, want) {
			return false
		}
	}
	for _, glob := range query.globs {
		if ok, err := filepath.Match(glob, e.LowerName); err != nil || !ok {
			return false
		}
	}
	for _, group := range query.alternates {
		matched := false
		for _, alt := range group {
			if strings.Contains(e.LowerPath, alt) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, fuzzy := range query.fuzzy {
		if !isSubsequence(fuzzy, e.LowerName) && !isSubsequence(fuzzy, e.LowerPath) {
			return false
		}
	}
	for _, re := range query.nameRegexes {
		if !re.MatchString(e.LowerName) {
			return false
		}
	}
	for _, re := range query.pathRegexes {
		if !re.MatchString(e.LowerPath) {
			return false
		}
	}
	if query.sizeMin >= 0 && e.Size < query.sizeMin {
		return false
	}
	if query.sizeMax >= 0 && e.Size > query.sizeMax {
		return false
	}
	if query.dateMin >= 0 || query.dateMax >= 0 {
		ts := e.ModTime.Unix()
		if query.dateMin >= 0 && ts < query.dateMin {
			return false
		}
		if query.dateMax >= 0 && ts > query.dateMax {
			return false
		}
	}
	return matchesAll(e.LowerPath, query.tokens)
}

func scoreEntry(e fileEntry, query searchQuery) int {
	score := e.Priority*1000 + pathDepth(e.Path)*2
	if e.IsDir {
		score -= 80
	}
	ext := strings.TrimPrefix(e.LowerExt, ".")
	switch ext {
	case "lnk", "url", "exe", "appref-ms":
		score -= 60
	case "dll", "sys", "tmp", "log", "cache", "dat", "pak", "manifest":
		score += 180
	}
	if query.phrase != "" {
		stem := strings.TrimSuffix(e.LowerName, e.LowerExt)
		switch {
		case e.LowerName == query.phrase || stem == query.phrase:
			score -= 320
		case strings.HasPrefix(stem, query.phrase) || strings.HasPrefix(e.LowerName, query.phrase):
			score -= 220
		case strings.Contains(stem, query.phrase) || strings.Contains(e.LowerName, query.phrase):
			score -= 130
		case matchesAll(e.LowerName, query.tokens):
			score -= 90
		}
	}
	for _, wanted := range query.exts {
		if ext == wanted {
			score -= 120
			break
		}
	}
	for _, want := range query.names {
		switch {
		case e.LowerName == want:
			score -= 320
		case strings.HasPrefix(e.LowerName, want):
			score -= 220
		case strings.Contains(e.LowerName, want):
			score -= 130
		}
	}
	if strings.Contains(e.LowerPath, `\setup\`) || strings.Contains(e.LowerPath, `\installer\`) {
		score += 120
	}
	return score
}

func pathDepth(path string) int {
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == string(osPathSeparator()) {
		return 0
	}
	return strings.Count(cleaned, string(osPathSeparator()))
}

func matchesAll(s string, tokens []string) bool {
	for _, token := range tokens {
		if !strings.Contains(s, token) {
			return false
		}
	}
	return true
}

func searchFold(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'ı', 'İ':
			r = 'i'
		case '\u0307':
			continue
		case 'ç':
			r = 'c'
		case 'ğ':
			r = 'g'
		case 'ö':
			r = 'o'
		case 'ş':
			r = 's'
		case 'ü':
			r = 'u'
		}
		b.WriteRune(r)
	}
	return b.String()
}

func osPathSeparator() byte {
	return '\\'
}
