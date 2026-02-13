package metadata

import "testing"

func TestNormalizeVirtualPath(t *testing.T) {
	got := normalizeVirtualPath("20200101/2014/a.pdf")
	if got != "/20200101/2014/a.pdf" {
		t.Fatalf("unexpected path: %s", got)
	}
}

func TestJoinVirtualPath(t *testing.T) {
	got := JoinVirtualPath("/files", "x.txt")
	if got != "/files/x.txt" {
		t.Fatalf("unexpected joined path: %s", got)
	}
}
