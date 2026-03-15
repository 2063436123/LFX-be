package server

import "testing"

func TestDecodePullCursorRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if _, _, err := decodePullCursor("invalid"); err == nil {
		t.Fatal("expected invalid cursor error")
	}
}

func TestBuildPullFilterIncludesSnapshotAndCursorBounds(t *testing.T) {
	t.Parallel()

	filter, args, err := buildPullFilter("2026-03-01T00:00:00Z", "2026-03-02T00:00:00Z|abc123", "2026-03-03T00:00:00Z")
	if err != nil {
		t.Fatalf("buildPullFilter returned error: %v", err)
	}

	expectedFilter := "changedAt >= {:since} && changedAt <= {:until} && (changedAt > {:cursorChangedAt} || (changedAt = {:cursorChangedAt} && id > {:cursorId}))"
	if filter != expectedFilter {
		t.Fatalf("unexpected filter:\nwant: %s\n got: %s", expectedFilter, filter)
	}

	if got := args["since"]; got != "2026-03-01T00:00:00Z" {
		t.Fatalf("unexpected since arg: %v", got)
	}
	if got := args["until"]; got != "2026-03-03T00:00:00Z" {
		t.Fatalf("unexpected until arg: %v", got)
	}
	if got := args["cursorChangedAt"]; got != "2026-03-02T00:00:00Z" {
		t.Fatalf("unexpected cursorChangedAt arg: %v", got)
	}
	if got := args["cursorId"]; got != "abc123" {
		t.Fatalf("unexpected cursorId arg: %v", got)
	}
}
