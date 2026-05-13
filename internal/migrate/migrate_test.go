package migrate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gunysa1/tgfs/internal/migrate"
)

func TestWalkLibrary(t *testing.T) {
	root := t.TempDir()
	moviesDir := filepath.Join(root, "Movies", "Inception (2010) {imdb-tt1375666}")
	os.MkdirAll(moviesDir, 0755)
	os.WriteFile(filepath.Join(moviesDir, "Inception.mkv"), []byte("fake video"), 0644)
	os.WriteFile(filepath.Join(moviesDir, "Inception.srt"), []byte("subtitle"), 0644)
	os.WriteFile(filepath.Join(root, "Movies", "Flatfile.mkv"), []byte("flat"), 0644)

	entries, err := migrate.WalkLibrary(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	paths := make(map[string]migrate.Entry)
	for _, e := range entries {
		paths[e.VirtualPath] = e
	}

	if _, ok := paths["/Movies"]; !ok {
		t.Error("expected /Movies dir entry")
	}
	if e, ok := paths["/Movies/Inception (2010) {imdb-tt1375666}/Inception.mkv"]; !ok {
		t.Error("expected Inception.mkv entry")
	} else if e.IsDir {
		t.Error("Inception.mkv should not be a dir")
	}
	if _, ok := paths["/Movies/Flatfile.mkv"]; !ok {
		t.Error("expected flat Flatfile.mkv entry")
	}
}

func TestPlanChunks_SingleChunk(t *testing.T) {
	chunks := migrate.PlanChunks(100, 200)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Offset != 0 || chunks[0].Size != 100 || chunks[0].Index != 0 {
		t.Errorf("unexpected chunk: %+v", chunks[0])
	}
}

func TestPlanChunks_MultipleChunks(t *testing.T) {
	chunks := migrate.PlanChunks(500, 200)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if chunks[2].Offset != 400 || chunks[2].Size != 100 {
		t.Errorf("unexpected last chunk: %+v", chunks[2])
	}
}

func TestChunkFilename_Single(t *testing.T) {
	name := migrate.ChunkFilename("/Movies/foo.mkv", 0, 1)
	if name != "Movies/foo.mkv" {
		t.Errorf("expected 'Movies/foo.mkv', got %q", name)
	}
}

func TestChunkFilename_Multi(t *testing.T) {
	name := migrate.ChunkFilename("/Movies/foo.mkv", 1, 3)
	if name != "Movies/foo.mkv.part001" {
		t.Errorf("expected 'Movies/foo.mkv.part001', got %q", name)
	}
}

func TestOpenChunk(t *testing.T) {
	f, err := os.CreateTemp("", "chunk-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("hello world")
	f.Close()
	defer os.Remove(f.Name())

	sr, file, err := migrate.OpenChunk(f.Name(), 6, 5)
	if err != nil {
		t.Fatalf("open chunk: %v", err)
	}
	defer file.Close()

	buf := make([]byte, 5)
	n, _ := sr.Read(buf)
	if string(buf[:n]) != "world" {
		t.Errorf("expected 'world', got %q", string(buf[:n]))
	}
}

func TestMigrator_DryRun(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "test.mkv"), []byte("video"), 0644)

	m := &migrate.Migrator{
		ChunkSizeBytes: 1000,
		DryRun:         true,
	}

	var uploaded []string
	err := m.Run(
		root,
		func(vpath string) bool { return false },
		func(vpath, name string) error { return nil },
		func(entry migrate.Entry) error {
			uploaded = append(uploaded, entry.VirtualPath)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uploaded) != 0 {
		t.Errorf("dry-run should not upload, got: %v", uploaded)
	}
}

func TestMigrator_SkipsExisting(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "test.mkv"), []byte("video"), 0644)

	m := &migrate.Migrator{ChunkSizeBytes: 1000}

	var skipped []string
	var uploaded []string
	err := m.Run(
		root,
		func(vpath string) bool { return vpath == "/test.mkv" },
		func(vpath, name string) error { return nil },
		func(entry migrate.Entry) error {
			uploaded = append(uploaded, entry.VirtualPath)
			return nil
		},
	)
	_ = skipped
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(uploaded) != 0 {
		t.Errorf("should skip existing, but uploaded: %v", uploaded)
	}
}
