package handler

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

const memoExcerptLimit = 140

var (
	memoRefPattern      = regexp.MustCompile(`\[\[memo:([0-9a-fA-F-]{36})\]\]`)
	markdownLinkPattern = regexp.MustCompile(`!?\[([^\]]+)\]\([^\)]+\)`)
	whitespacePattern   = regexp.MustCompile(`\s+`)
)

type parsedMemoContent struct {
	Tags         []string
	ReferenceIDs []uuid.UUID
	Excerpt      string
}

func parseMemoContent(content string) parsedMemoContent {
	return parsedMemoContent{
		Tags:         extractTags(content),
		ReferenceIDs: extractMemoReferenceIDs(content),
		Excerpt:      buildMemoExcerpt(content),
	}
}

func extractMemoReferenceIDs(content string) []uuid.UUID {
	matches := memoRefPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]struct{}, len(matches))
	refs := make([]uuid.UUID, 0, len(matches))
	for _, match := range matches {
		id, err := uuid.Parse(match[1])
		if err != nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		refs = append(refs, id)
	}
	return refs
}

func extractTags(content string) []string {
	runes := []rune(content)
	seen := map[string]struct{}{}
	tags := make([]string, 0)

	for i := 0; i < len(runes); i++ {
		if runes[i] != '#' {
			continue
		}
		if i > 0 && isTagBodyRune(runes[i-1]) {
			continue
		}

		j := i + 1
		segmentStart := j
		for j < len(runes) && isTagAtomRune(runes[j]) {
			j++
		}
		if j == segmentStart {
			continue
		}

		for j < len(runes) && runes[j] == '/' {
			next := j + 1
			segmentStart = next
			for next < len(runes) && isTagAtomRune(runes[next]) {
				next++
			}
			if next == segmentStart {
				break
			}
			j = next
		}

		tag := canonicalizeTag(string(runes[i+1 : j]))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
		i = j - 1
	}

	sort.Strings(tags)
	return tags
}

func canonicalizeTag(raw string) string {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "#"))
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return ""
		}
		clean = append(clean, strings.ToLower(part))
	}
	return strings.Join(clean, "/")
}

func isTagAtomRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func isTagBodyRune(r rune) bool {
	return isTagAtomRune(r) || r == '/'
}

func buildMemoExcerpt(content string) string {
	text := memoRefPattern.ReplaceAllString(content, " ")
	text = markdownLinkPattern.ReplaceAllString(text, "$1")
	text = strings.NewReplacer(
		"`", " ",
		"*", " ",
		"_", " ",
		">", " ",
		"~", " ",
	).Replace(text)
	text = whitespacePattern.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	if utf8.RuneCountInString(text) <= memoExcerptLimit {
		return text
	}

	runes := []rune(text)
	return string(runes[:memoExcerptLimit])
}
