package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type File struct {
	ID        int64
	Path      string
	Name      string
	Size      int64
	MimeType  string
	IsDir     bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (d *DB) CreateFile(f File) (File, error) {
	isDir := 0
	if f.IsDir {
		isDir = 1
	}
	res, err := d.conn.Exec(
		`INSERT INTO files (path, name, size, mime_type, is_dir) VALUES (?, ?, ?, ?, ?)`,
		f.Path, f.Name, f.Size, f.MimeType, isDir,
	)
	if err != nil {
		return File{}, fmt.Errorf("insert file: %w", err)
	}
	id, _ := res.LastInsertId()
	f.ID = id
	return f, nil
}

func (d *DB) GetFileByPath(path string) (File, error) {
	var f File
	var isDir int
	err := d.conn.QueryRow(
		`SELECT id, path, name, size, mime_type, is_dir, created_at, updated_at FROM files WHERE path = ?`, path,
	).Scan(&f.ID, &f.Path, &f.Name, &f.Size, &f.MimeType, &isDir, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, ErrNotFound
	}
	if err != nil {
		return File{}, err
	}
	f.IsDir = isDir == 1
	return f, nil
}

// ListChildren returns direct children of the given directory path.
func (d *DB) ListChildren(dirPath string) ([]File, error) {
	prefix := strings.TrimRight(dirPath, "/") + "/"
	rows, err := d.conn.Query(
		`SELECT id, path, name, size, mime_type, is_dir, created_at, updated_at
		 FROM files
		 WHERE path LIKE ? AND path NOT LIKE ?
		 ORDER BY name`,
		prefix+"%",
		prefix+"%/%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []File
	for rows.Next() {
		var f File
		var isDir int
		if err := rows.Scan(&f.ID, &f.Path, &f.Name, &f.Size, &f.MimeType, &isDir, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		f.IsDir = isDir == 1
		out = append(out, f)
	}
	return out, rows.Err()
}

func (d *DB) UpdateFilePath(id int64, newPath, newName string) error {
	_, err := d.conn.Exec(
		`UPDATE files SET path = ?, name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		newPath, newName, id,
	)
	return err
}

func (d *DB) DeleteFile(id int64) error {
	_, err := d.conn.Exec(`DELETE FROM files WHERE id = ?`, id)
	return err
}
