package chunker_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/gunysa1/tgfs/internal/chunker"
)

func TestSplit_SingleChunk(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 100)
	var chunks [][]byte
	err := chunker.Split(bytes.NewReader(data), 200, func(idx int, r io.Reader) error {
		b, _ := io.ReadAll(r)
		chunks = append(chunks, b)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if !bytes.Equal(chunks[0], data) {
		t.Error("chunk content mismatch")
	}
}

func TestSplit_MultipleChunks(t *testing.T) {
	data := bytes.Repeat([]byte("ab"), 100) // 200 bytes
	var chunks [][]byte
	err := chunker.Split(bytes.NewReader(data), 60, func(idx int, r io.Reader) error {
		b, _ := io.ReadAll(r)
		chunks = append(chunks, b)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 200 / 60 = 3 full + 1 partial = 4 chunks
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks, got %d", len(chunks))
	}
	var reassembled []byte
	for _, c := range chunks {
		reassembled = append(reassembled, c...)
	}
	if !bytes.Equal(reassembled, data) {
		t.Error("reassembled content mismatch")
	}
}

func TestResolveRange(t *testing.T) {
	// 3 chunks: [0..99], [100..199], [200..249]
	chunks := []chunker.ChunkMeta{
		{Index: 0, Size: 100},
		{Index: 1, Size: 100},
		{Index: 2, Size: 50},
	}
	// Read bytes 80..149 (spans chunk 0 and chunk 1)
	result := chunker.ResolveRange(chunks, 80, 70)
	if len(result) != 2 {
		t.Fatalf("expected 2 chunk refs, got %d", len(result))
	}
	if result[0].ChunkIndex != 0 || result[0].OffsetInChunk != 80 || result[0].Length != 20 {
		t.Errorf("unexpected first ref: %+v", result[0])
	}
	if result[1].ChunkIndex != 1 || result[1].OffsetInChunk != 0 || result[1].Length != 50 {
		t.Errorf("unexpected second ref: %+v", result[1])
	}
}

func TestResolveRange_SingleChunk(t *testing.T) {
	chunks := []chunker.ChunkMeta{{Index: 0, Size: 1000}}
	result := chunker.ResolveRange(chunks, 100, 200)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk ref, got %d", len(result))
	}
	if result[0].OffsetInChunk != 100 || result[0].Length != 200 {
		t.Errorf("unexpected ref: %+v", result[0])
	}
}

func TestResolveRange_ExactChunkBoundary(t *testing.T) {
	chunks := []chunker.ChunkMeta{
		{Index: 0, Size: 100},
		{Index: 1, Size: 100},
	}
	// Read exactly chunk 1
	result := chunker.ResolveRange(chunks, 100, 100)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk ref, got %d", len(result))
	}
	if result[0].ChunkIndex != 1 || result[0].OffsetInChunk != 0 || result[0].Length != 100 {
		t.Errorf("unexpected ref: %+v", result[0])
	}
}
