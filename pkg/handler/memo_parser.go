package handler

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const memoExcerptLimit = 140

func validateMemoContent(content json.RawMessage) error {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ErrInvalidMemoContent
	}
	return nil
}

func normalizeMemoText(text string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
}

func normalizeMemoExcerpt(excerpt string) string {
	excerpt = normalizeMemoText(excerpt)
	if excerpt == "" {
		return ""
	}
	if utf8.RuneCountInString(excerpt) <= memoExcerptLimit {
		return excerpt
	}
	runes := []rune(excerpt)
	return string(runes[:memoExcerptLimit])
}

func normalizeMemoTags(tags []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := canonicalizeTag(raw)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	sort.Strings(normalized)
	return normalized
}

func normalizeMemoReferences(refs []uuid.UUID) []uuid.UUID {
	if len(refs) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(refs))
	normalized := make([]uuid.UUID, 0, len(refs))
	for _, ref := range refs {
		if ref == uuid.Nil {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		normalized = append(normalized, ref)
	}
	return normalized
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
