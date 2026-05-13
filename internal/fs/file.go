package fs

import (
	"context"
	"io"
	"os"

	"bazil.org/fuse"
	fuseFS "bazil.org/fuse/fs"
	"github.com/gunysa1/tgfs/internal/chunker"
	"github.com/gunysa1/tgfs/internal/db"
)

type File struct {
	tgfs *TgFS
	file db.File
}

var _ fuseFS.Node = (*File)(nil)
var _ fuseFS.HandleReader = (*File)(nil)

func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.FileMode(0444)
	a.Size = uint64(f.file.Size)
	a.Mtime = f.file.UpdatedAt
	a.Ctime = f.file.CreatedAt
	return nil
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	chunks, err := f.tgfs.db.GetChunksByFileID(f.file.ID)
	if err != nil {
		return err
	}

	metas := make([]chunker.ChunkMeta, len(chunks))
	for i, c := range chunks {
		metas[i] = chunker.ChunkMeta{Index: c.ChunkIndex, Size: c.Size}
	}

	refs := chunker.ResolveRange(metas, req.Offset, int64(req.Size))
	var result []byte

	for _, ref := range refs {
		chunk := chunks[ref.ChunkIndex]

		if cached, ok := f.tgfs.cache.Get(f.file.ID, ref.ChunkIndex); ok {
			result = append(result, cached[ref.OffsetInChunk:ref.OffsetInChunk+ref.Length]...)
			continue
		}

		rc, err := f.tgfs.bot.DownloadByFileID(chunk.TelegramFileID)
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		f.tgfs.cache.Set(f.file.ID, ref.ChunkIndex, data)
		result = append(result, data[ref.OffsetInChunk:ref.OffsetInChunk+ref.Length]...)
	}

	resp.Data = result
	return nil
}
