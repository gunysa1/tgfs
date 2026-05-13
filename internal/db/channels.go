package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Channel struct {
	ID         int64
	TelegramID int64
	Name       string
	CreatedAt  time.Time
}

func (d *DB) CreateChannel(ch Channel) (Channel, error) {
	res, err := d.conn.Exec(
		`INSERT INTO channels (telegram_id, name) VALUES (?, ?)`,
		ch.TelegramID, ch.Name,
	)
	if err != nil {
		return Channel{}, fmt.Errorf("insert channel: %w", err)
	}
	id, _ := res.LastInsertId()
	ch.ID = id
	return ch, nil
}

func (d *DB) ListChannels() ([]Channel, error) {
	rows, err := d.conn.Query(`SELECT id, telegram_id, name, created_at FROM channels ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		var ch Channel
		if err := rows.Scan(&ch.ID, &ch.TelegramID, &ch.Name, &ch.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (d *DB) GetChannelByName(name string) (Channel, error) {
	var ch Channel
	err := d.conn.QueryRow(
		`SELECT id, telegram_id, name, created_at FROM channels WHERE name = ?`, name,
	).Scan(&ch.ID, &ch.TelegramID, &ch.Name, &ch.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Channel{}, ErrNotFound
	}
	if err != nil {
		return Channel{}, err
	}
	return ch, nil
}

func (d *DB) DeleteChannel(id int64) error {
	_, err := d.conn.Exec(`DELETE FROM channels WHERE id = ?`, id)
	return err
}
