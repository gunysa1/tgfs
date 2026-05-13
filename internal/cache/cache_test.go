package cache_test

import (
	"bytes"
	"testing"

	"github.com/gunysa1/tgfs/internal/cache"
)

func TestCache_GetSet(t *testing.T) {
	c := cache.New(10) // 10 bytes max
	data := []byte("hello")
	c.Set(1, 0, data)

	got, ok := c.Get(1, 0)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !bytes.Equal(got, data) {
		t.Errorf("expected %q, got %q", data, got)
	}
}

func TestCache_Miss(t *testing.T) {
	c := cache.New(100)
	_, ok := c.Get(99, 0)
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestCache_Eviction(t *testing.T) {
	c := cache.New(10) // 10 bytes max
	c.Set(1, 0, []byte("12345678")) // 8 bytes
	c.Set(1, 1, []byte("abcde"))    // 5 bytes — should evict first entry
	_, ok := c.Get(1, 0)
	if ok {
		t.Fatal("expected eviction of first entry")
	}
	_, ok = c.Get(1, 1)
	if !ok {
		t.Fatal("expected second entry present")
	}
}

func TestCache_Clear(t *testing.T) {
	c := cache.New(100)
	c.Set(1, 0, []byte("data"))
	c.Clear()
	_, ok := c.Get(1, 0)
	if ok {
		t.Fatal("expected cache miss after clear")
	}
}

func TestCache_Overwrite(t *testing.T) {
	c := cache.New(100)
	c.Set(1, 0, []byte("hello"))   // 5 bytes
	c.Set(1, 0, []byte("hi"))      // 2 bytes — overwrites, should not double-count
	cur, _ := c.Stats()
	if cur != 2 {
		t.Errorf("expected 2 bytes after overwrite, got %d", cur)
	}
	got, ok := c.Get(1, 0)
	if !ok || string(got) != "hi" {
		t.Errorf("expected 'hi', got %q ok=%v", got, ok)
	}
}

func TestCache_Stats(t *testing.T) {
	c := cache.New(100)
	c.Set(1, 0, []byte("hello")) // 5 bytes
	cur, max := c.Stats()
	if cur != 5 {
		t.Errorf("expected 5 bytes used, got %d", cur)
	}
	if max != 100 {
		t.Errorf("expected max 100, got %d", max)
	}
}
