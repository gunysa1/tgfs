package fs

import (
	"context"
	"os"
	"strings"

	"bazil.org/fuse"
	fuseFS "bazil.org/fuse/fs"
	"github.com/gunysa1/tgfs/internal/db"
)

type Dir struct {
	tgfs *TgFS
	path string
}

var _ fuseFS.Node = (*Dir)(nil)
var _ fuseFS.HandleReadDirAller = (*Dir)(nil)
var _ fuseFS.NodeStringLookuper = (*Dir)(nil)

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0555
	return nil
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	children, err := d.tgfs.db.ListChildren(d.path)
	if err != nil {
		return nil, err
	}
	var entries []fuse.Dirent
	for _, f := range children {
		t := fuse.DT_File
		if f.IsDir {
			t = fuse.DT_Dir
		}
		entries = append(entries, fuse.Dirent{Name: f.Name, Type: t})
	}
	return entries, nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fuseFS.Node, error) {
	childPath := strings.TrimRight(d.path, "/") + "/" + name
	f, err := d.tgfs.db.GetFileByPath(childPath)
	if err == db.ErrNotFound {
		return nil, fuse.ENOENT
	}
	if err != nil {
		return nil, err
	}
	if f.IsDir {
		return &Dir{tgfs: d.tgfs, path: childPath}, nil
	}
	return &File{tgfs: d.tgfs, file: f}, nil
}
