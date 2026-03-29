package handler

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseMemoContent(t *testing.T) {
	t.Parallel()

	refID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	content := "#Work shipping note with [[memo:" + refID.String() + "]] and #golang plus [docs](https://example.com)"

	parsed := parseMemoContent(content)

	if len(parsed.Tags) != 2 || parsed.Tags[0] != "golang" || parsed.Tags[1] != "work" {
		t.Fatalf("unexpected tags: %#v", parsed.Tags)
	}
	if len(parsed.ReferenceIDs) != 1 || parsed.ReferenceIDs[0] != refID {
		t.Fatalf("unexpected references: %#v", parsed.ReferenceIDs)
	}
	if parsed.Excerpt == "" {
		t.Fatal("expected excerpt")
	}
}

func TestExtractTags_DedupAndHierarchy(t *testing.T) {
	t.Parallel()

	tags := extractTags("#Work/workflows #Work/workflows text C# #Ideas/backend #ideas/backend")
	if len(tags) != 2 {
		t.Fatalf("unexpected tag count: %#v", tags)
	}
	if tags[0] != "ideas/backend" || tags[1] != "work/workflows" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}
