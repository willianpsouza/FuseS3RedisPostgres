package cache

import "testing"

func TestLRUEvict(t *testing.T) {
	l := NewLRU[string, int](2)
	l.Set("a", 1)
	l.Set("b", 2)
	l.Set("c", 3)
	if _, ok := l.Get("a"); ok {
		t.Fatal("expected a evicted")
	}
	if v, ok := l.Get("c"); !ok || v != 3 {
		t.Fatal("expected c present")
	}
}
