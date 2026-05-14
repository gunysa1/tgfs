package migrate

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

type Entry struct {
	LocalPath   string
	VirtualPath string
	Name        string
	Size        int64
	MimeType    string
	IsDir       bool
}

// WalkLibrary walks localRoot and returns all entries mapped to virtual paths.
// The virtual path is the relative path from localRoot prefixed with "/".
func WalkLibrary(localRoot string) ([]Entry, error) {
	var entries []Entry
	err := filepath.WalkDir(localRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(localRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		virtualPath := "/" + filepath.ToSlash(rel)
		name := d.Name()

		if d.IsDir() {
			entries = append(entries, Entry{
				LocalPath:   path,
				VirtualPath: virtualPath,
				Name:        name,
				IsDir:       true,
			})
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		mimeType := mime.TypeByExtension(filepath.Ext(name))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		entries = append(entries, Entry{
			LocalPath:   path,
			VirtualPath: virtualPath,
			Name:        name,
			Size:        info.Size(),
			MimeType:    mimeType,
			IsDir:       false,
		})
		return nil
	})
	return entries, err
}

// Migrator holds configuration for a migration run.
type Migrator struct {
	ChunkSizeBytes int64
	DryRun         bool
	FileWorkers    int
	OnProgress     func(virtualPath string, done, total int)
	OnSkip         func(virtualPath string)
	OnError        func(virtualPath string, err error)
}

// Run executes the migration for all entries in localRoot.
// Directories are created sequentially in walk order; file uploads
// are fanned out to FileWorkers goroutines. Per-file upload errors
// are reported via OnError and do not abort the run.
func (m *Migrator) Run(
	localRoot string,
	fileExists func(virtualPath string) bool,
	createDir func(virtualPath, name string) error,
	uploadAndRecord func(entry Entry) error,
) error {
	entries, err := WalkLibrary(localRoot)
	if err != nil {
		return fmt.Errorf("walk library: %w", err)
	}

	workers := m.FileWorkers
	if workers < 1 {
		workers = 1
	}

	total := len(entries)
	var done int64

	// Pass 1: directories synchronously (ensures parents exist before children).
	for _, e := range entries {
		if !e.IsDir {
			continue
		}
		i := atomic.AddInt64(&done, 1) - 1
		if m.OnProgress != nil {
			m.OnProgress(e.VirtualPath, int(i), total)
		}
		if fileExists(e.VirtualPath) {
			if m.OnSkip != nil {
				m.OnSkip(e.VirtualPath)
			}
			continue
		}
		if m.DryRun {
			fmt.Printf("[dry-run] would mkdir: %s\n", e.VirtualPath)
			continue
		}
		if err := createDir(e.VirtualPath, e.Name); err != nil {
			return fmt.Errorf("create dir %s: %w", e.VirtualPath, err)
		}
	}

	// Pass 2: files in parallel.
	jobs := make(chan Entry)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range jobs {
				i := atomic.AddInt64(&done, 1) - 1
				if m.OnProgress != nil {
					m.OnProgress(e.VirtualPath, int(i), total)
				}
				if fileExists(e.VirtualPath) {
					if m.OnSkip != nil {
						m.OnSkip(e.VirtualPath)
					}
					continue
				}
				if m.DryRun {
					fmt.Printf("[dry-run] would upload: %s\n", e.VirtualPath)
					continue
				}
				if err := uploadAndRecord(e); err != nil {
					if m.OnError != nil {
						m.OnError(e.VirtualPath, err)
					}
				}
			}
		}()
	}
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		jobs <- e
	}
	close(jobs)
	wg.Wait()
	return nil
}

// ChunkRange describes a byte range within a file for one upload chunk.
type ChunkRange struct {
	Offset int64
	Size   int64
	Index  int
}

// PlanChunks splits a file of fileSize into chunks of at most chunkSize bytes.
func PlanChunks(fileSize, chunkSize int64) []ChunkRange {
	var chunks []ChunkRange
	var offset int64
	idx := 0
	for offset < fileSize {
		size := chunkSize
		if offset+size > fileSize {
			size = fileSize - offset
		}
		chunks = append(chunks, ChunkRange{Offset: offset, Size: size, Index: idx})
		offset += size
		idx++
	}
	return chunks
}

// ChunkFilename generates a deterministic chunk filename for upload.
func ChunkFilename(virtualPath string, chunkIndex, totalChunks int) string {
	base := strings.TrimPrefix(virtualPath, "/")
	if totalChunks == 1 {
		return base
	}
	return fmt.Sprintf("%s.part%03d", base, chunkIndex)
}

// OpenChunk opens a file and returns an io.SectionReader for the given byte range.
// Caller must close the returned *os.File.
func OpenChunk(localPath string, offset, size int64) (*io.SectionReader, *os.File, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, nil, err
	}
	return io.NewSectionReader(f, offset, size), f, nil
}
