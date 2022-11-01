package main

import (
	"context"
	"encoding/json"
	//"time"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

const CREATE_TABLES = `

CREATE TABLE IF NOT EXISTS messages (
	pk     SERIAL PRIMARY KEY,
	sender VARCHAR(64),
	data   JSON NOT NULL
);

CREATE TABLE IF NOT EXISTS receivers (
	message  INTEGER REFERENCES messages,
	receiver VARCHAR(64),
	read     BOOLEAN DEFAULT false,
	PRIMARY KEY (receiver, message)
);
`

const SAVE_MESSAGES = `
INSERT INTO messages (sender, data) VALUES ($1, $2) RETURNING pk;
`

const SAVE_RECV = `
INSERT INTO receivers (receiver, message, read) VALUES ($1, $2, false);
`

const READ_ALL_RECV = `
SELECT m.pk, m.sender, m.data, r.read FROM messages AS m JOIN receivers AS r ON m.pk=r.message WHERE r.receiver = $1;
`

const READ_UNREAD_RECV = `
SELECT m.pk, m.sender, m.data, r.read FROM messages AS m JOIN receivers AS r ON m.pk=r.message WHERE r.receiver = $1 AND NOT r.read;
`

const UPDATE_READ_RECV = `
UPDATE receivers SET read = $1 WHERE receiver = $2 AND message = $3;
`

const COUNT_UNREAD_RECV = `
SELECT COUNT(message) FROM receivers WHERE receiver = $1 AND NOT read;
`

type SqlStorage struct {
	db  *sql.DB
	ctx context.Context
}

type ReadAll struct {
	Id      int                    `json:"id"`
	Sender  string                 `json:"sender"`
	Content map[string]interface{} `json:"content"`
	Read    bool                   `json:"read"`
}

func (storage *SqlStorage) init(connectionStr string) error {
	var err error
	storage.db, err = sql.Open("postgres", connectionStr)
	if err != nil {
		return err
	}
	err = storage.db.Ping()
	if err != nil {
		return err
	}
	_, err = storage.db.ExecContext(storage.ctx, CREATE_TABLES)
	if err != nil {
		return err
	}
	return nil
}

func (storage *SqlStorage) send(message Message) (int, error) {
	stmt, err := storage.db.Prepare(SAVE_MESSAGES)
	if err != nil {
		return 0, err
	}
	id := 0
	jsonData, err := json.Marshal(message.Content)
	err = stmt.QueryRow(message.Sender, jsonData).Scan(&id)
	if err != nil {
		return 0, err
	}
	fmt.Println(id)
	stmt.Close()
	stmt, err = storage.db.Prepare(SAVE_RECV)
	if err != nil {
		return 0, err
	}
	count := 0
	for i := 0; i < len(message.Receivers); i++ {
		_, err = stmt.Exec(message.Receivers[i], id)
		if err == nil {
			count = count + 1
		}
	}
	stmt.Close()
	return count, nil
}

func (storage *SqlStorage) read(who string, onlyUnread bool) ([]ReadAll, error) {
	query := READ_ALL_RECV
	if onlyUnread {
		query = READ_UNREAD_RECV
	}
	rows, err := storage.db.Query(query, who)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []ReadAll

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var current ReadAll
		var data []byte
		if err := rows.Scan(&current.Id, &current.Sender, &data, &current.Read); err != nil {
			return messages, err
		}
		err := json.Unmarshal(data, &current.Content)
		if err != nil {
			return messages, err
		}
		messages = append(messages, current)
	}
	if err = rows.Err(); err != nil {
		return messages, err
	}

	return messages, nil
}

func (storage *SqlStorage) set(who string, message_id int, read bool) error {
	stmt, err := storage.db.Prepare(UPDATE_READ_RECV)
	if err != nil {
		return err
	}
	if _, err := stmt.Exec(read, who, message_id); err != nil {
		return err
	}
	return nil
}

func (storage *SqlStorage) countUnread(who string) (int, error) {
	stmt, err := storage.db.Prepare(COUNT_UNREAD_RECV)
	if err != nil {
		return 0, err
	}
	var count int
	err = stmt.QueryRow(who).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
