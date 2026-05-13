package chunker

import (
	"bytes"
	"io"
)

// ChunkMeta describes one stored chunk's position in the full file.
type ChunkMeta struct {
	Index int
	Size  int64
}

// ChunkRef describes a byte range within a specific chunk needed to satisfy a read request.
type ChunkRef struct {
	ChunkIndex    int
	OffsetInChunk int64
	Length        int64
}

// Split reads from r and calls fn for each chunk of at most chunkSize bytes.
// fn receives the 0-based chunk index and a reader for that chunk's data.
func Split(r io.Reader, chunkSize int64, fn func(idx int, r io.Reader) error) error {
	buf := make([]byte, chunkSize)
	idx := 0
	for {
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			if ferr := fn(idx, bytes.NewReader(buf[:n])); ferr != nil {
				return ferr
			}
			idx++
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ResolveRange maps a file-level byte range [offset, offset+length) to a list of ChunkRefs.
func ResolveRange(chunks []ChunkMeta, offset, length int64) []ChunkRef {
	var refs []ChunkRef
	end := offset + length
	var chunkStart int64
	for _, c := range chunks {
		chunkEnd := chunkStart + c.Size
		if chunkEnd <= offset {
			chunkStart = chunkEnd
			continue
		}
		if chunkStart >= end {
			break
		}
		readStart := max64(offset, chunkStart)
		readEnd := min64(end, chunkEnd)
		refs = append(refs, ChunkRef{
			ChunkIndex:    c.Index,
			OffsetInChunk: readStart - chunkStart,
			Length:        readEnd - readStart,
		})
		chunkStart = chunkEnd
	}
	return refs
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
