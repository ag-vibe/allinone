package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateMemoContent(t *testing.T) {
	t.Parallel()

	if err := validateMemoContent(json.RawMessage("")); err == nil {
		t.Fatal("expected empty content to be invalid")
	}
	if err := validateMemoContent(json.RawMessage("null")); err == nil {
		t.Fatal("expected null content to be invalid")
	}
	if err := validateMemoContent(json.RawMessage("{}")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeMemoExcerpt(t *testing.T) {
	t.Parallel()

	excerpt := normalizeMemoExcerpt("  Hello\r\nworld  ")
	if excerpt != "Hello\nworld" {
		t.Fatalf("unexpected excerpt: %q", excerpt)
	}

	long := strings.Repeat("a", memoExcerptLimit+10)
	shortened := normalizeMemoExcerpt(long)
	if len([]rune(shortened)) != memoExcerptLimit {
		t.Fatalf("expected excerpt length %d, got %d", memoExcerptLimit, len([]rune(shortened)))
	}
}

func TestNormalizeMemoTags(t *testing.T) {
	t.Parallel()

	tags := normalizeMemoTags([]string{"#Work/workflows", "Work/workflows", "C#", "Ideas/backend", "ideas/backend"})
	if len(tags) != 3 {
		t.Fatalf("unexpected tag count: %#v", tags)
	}
	if tags[0] != "c#" || tags[1] != "ideas/backend" || tags[2] != "work/workflows" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}

func TestNormalizeMemoReferences(t *testing.T) {
	t.Parallel()

	refID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	refs := normalizeMemoReferences([]uuid.UUID{uuid.Nil, refID, refID})
	if len(refs) != 1 || refs[0] != refID {
		t.Fatalf("unexpected references: %#v", refs)
	}
}
