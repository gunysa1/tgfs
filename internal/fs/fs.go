package fs

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"bazil.org/fuse"
	fuseFS "bazil.org/fuse/fs"
	"github.com/gunysa1/tgfs/internal/bot"
	"github.com/gunysa1/tgfs/internal/cache"
	"github.com/gunysa1/tgfs/internal/db"
)

type TgFS struct {
	db    *db.DB
	bot   *bot.Client
	cache *cache.Cache
}

func New(d *db.DB, b *bot.Client, c *cache.Cache) *TgFS {
	return &TgFS{db: d, bot: b, cache: c}
}

func (t *TgFS) Root() (fuseFS.Node, error) {
	return &Dir{tgfs: t, path: "/"}, nil
}

// Mount mounts the filesystem at mountPath and blocks until unmounted or context cancelled.
func Mount(ctx context.Context, mountPath string, tgfs *TgFS) error {
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return err
	}
	c, err := fuse.Mount(mountPath,
		fuse.FSName("tgfs"),
		fuse.Subtype("tgfs"),
		fuse.ReadOnly(),
	)
	if err != nil {
		return err
	}
	defer c.Close()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	go func() {
		select {
		case <-ctx.Done():
		case <-sig:
		}
		fuse.Unmount(mountPath)
	}()

	return fuseFS.Serve(c, tgfs)
}
