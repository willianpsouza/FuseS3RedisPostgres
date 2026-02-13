package s3io

import "testing"

func TestAlignRange(t *testing.T) {
	start, end := AlignRange(9<<20, 4096, 8<<20, 32<<20)
	if start != 8<<20 {
		t.Fatalf("want start 8MiB got %d", start)
	}
	if end != (9<<20)+4096-1 {
		t.Fatalf("unexpected end %d", end)
	}
}
