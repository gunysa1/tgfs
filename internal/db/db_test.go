package db_test

import (
	"os"
	"testing"

	"github.com/gunysa1/tgfs/internal/db"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	f, err := os.CreateTemp("", "tgfs-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	d, err := db.Open(f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestChannelCRUD(t *testing.T) {
	d := testDB(t)

	ch, err := d.CreateChannel(db.Channel{TelegramID: -1001234567890, Name: "movies"})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if ch.ID == 0 {
		t.Error("expected non-zero ID")
	}

	channels, err := d.ListChannels()
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}
	if len(channels) != 1 || channels[0].Name != "movies" {
		t.Errorf("unexpected channels: %v", channels)
	}

	got, err := d.GetChannelByName("movies")
	if err != nil {
		t.Fatalf("get channel by name: %v", err)
	}
	if got.TelegramID != -1001234567890 {
		t.Errorf("unexpected telegram_id: %d", got.TelegramID)
	}

	if err := d.DeleteChannel(ch.ID); err != nil {
		t.Fatalf("delete channel: %v", err)
	}
	channels, _ = d.ListChannels()
	if len(channels) != 0 {
		t.Error("expected empty channels after delete")
	}
}

func TestFileCRUD(t *testing.T) {
	d := testDB(t)

	dir, err := d.CreateFile(db.File{
		Path:  "/Movies",
		Name:  "Movies",
		Size:  0,
		IsDir: true,
	})
	if err != nil {
		t.Fatalf("create dir: %v", err)
	}
	if dir.ID == 0 {
		t.Error("expected non-zero ID")
	}

	f, err := d.CreateFile(db.File{
		Path:     "/Movies/Inception.mkv",
		Name:     "Inception.mkv",
		Size:     4000000000,
		MimeType: "video/x-matroska",
		IsDir:    false,
	})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}

	got, err := d.GetFileByPath("/Movies/Inception.mkv")
	if err != nil {
		t.Fatalf("get file by path: %v", err)
	}
	if got.ID != f.ID {
		t.Errorf("expected ID %d, got %d", f.ID, got.ID)
	}
	if got.Size != 4000000000 {
		t.Errorf("expected size 4000000000, got %d", got.Size)
	}

	children, err := d.ListChildren("/Movies")
	if err != nil {
		t.Fatalf("list children: %v", err)
	}
	if len(children) != 1 || children[0].Name != "Inception.mkv" {
		t.Errorf("unexpected children: %v", children)
	}

	if err := d.UpdateFilePath(f.ID, "/Movies/Inception2.mkv", "Inception2.mkv"); err != nil {
		t.Fatalf("update path: %v", err)
	}
	_, err = d.GetFileByPath("/Movies/Inception.mkv")
	if err != db.ErrNotFound {
		t.Error("expected old path to be gone")
	}

	if err := d.DeleteFile(f.ID); err != nil {
		t.Fatalf("delete file: %v", err)
	}
}

func TestChunkCRUD(t *testing.T) {
	d := testDB(t)

	ch, _ := d.CreateChannel(db.Channel{TelegramID: -100111, Name: "test"})
	f, _ := d.CreateFile(db.File{Path: "/test.mkv", Name: "test.mkv", Size: 3800000000})

	chunk, err := d.CreateChunk(db.Chunk{
		FileID:         f.ID,
		ChunkIndex:     0,
		MessageID:      42,
		TelegramFileID: "BQACAgIAAxkBAAIB",
		ChannelID:      ch.ID,
		Size:           1900000000,
	})
	if err != nil {
		t.Fatalf("create chunk: %v", err)
	}
	if chunk.ID == 0 {
		t.Error("expected non-zero ID")
	}

	chunks, err := d.GetChunksByFileID(f.ID)
	if err != nil {
		t.Fatalf("get chunks: %v", err)
	}
	if len(chunks) != 1 || chunks[0].ChunkIndex != 0 {
		t.Errorf("unexpected chunks: %v", chunks)
	}
	if chunks[0].TelegramFileID != "BQACAgIAAxkBAAIB" {
		t.Errorf("unexpected telegram_file_id: %q", chunks[0].TelegramFileID)
	}

	if err := d.DeleteChunksByFileID(f.ID); err != nil {
		t.Fatalf("delete chunks: %v", err)
	}
	chunks, _ = d.GetChunksByFileID(f.ID)
	if len(chunks) != 0 {
		t.Error("expected empty chunks after delete")
	}
}

func TestGetFileByPath_NotFound(t *testing.T) {
	d := testDB(t)
	_, err := d.GetFileByPath("/nonexistent")
	if err != db.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListChildren_Empty(t *testing.T) {
	d := testDB(t)
	children, err := d.ListChildren("/Movies")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("expected empty, got %d children", len(children))
	}
}
