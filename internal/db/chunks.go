package db

import "fmt"

type Chunk struct {
	ID             int64
	FileID         int64
	ChunkIndex     int
	MessageID      int
	TelegramFileID string
	ChannelID      int64
	Size           int64
}

func (d *DB) CreateChunk(c Chunk) (Chunk, error) {
	res, err := d.conn.Exec(
		`INSERT INTO chunks (file_id, chunk_index, message_id, telegram_file_id, channel_id, size) VALUES (?, ?, ?, ?, ?, ?)`,
		c.FileID, c.ChunkIndex, c.MessageID, c.TelegramFileID, c.ChannelID, c.Size,
	)
	if err != nil {
		return Chunk{}, fmt.Errorf("insert chunk: %w", err)
	}
	id, _ := res.LastInsertId()
	c.ID = id
	return c, nil
}

func (d *DB) GetChunksByFileID(fileID int64) ([]Chunk, error) {
	rows, err := d.conn.Query(
		`SELECT id, file_id, chunk_index, message_id, telegram_file_id, channel_id, size
		 FROM chunks WHERE file_id = ? ORDER BY chunk_index`,
		fileID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.FileID, &c.ChunkIndex, &c.MessageID, &c.TelegramFileID, &c.ChannelID, &c.Size); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (d *DB) DeleteChunksByFileID(fileID int64) error {
	_, err := d.conn.Exec(`DELETE FROM chunks WHERE file_id = ?`, fileID)
	return err
}
