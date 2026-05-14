package websocket

import (
	"database/sql"
)

type Message struct {
	Room      string
	UserID    int64
	Username  string
	Message   string
	Timestamp int64
	Type      string
	Online    int
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveMessage(m Message) error {
	_, err := r.db.Exec(`
		INSERT INTO chat_messages
		(room, user_id, username, message, type, online, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.Room,
		m.UserID,
		m.Username,
		m.Message,
		m.Type,
		m.Online,
		m.Timestamp,
	)
	return err
}

func (r *Repository) GetLastMessages(room string, limit int) ([]Message, error) {
	rows, err := r.db.Query(`
		SELECT room, user_id, username, message,  timestamp
		FROM chat_messages
		WHERE room = ? 
		ORDER BY timestamp DESC
		LIMIT ?`, room, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Room, &m.UserID, &m.Username, &m.Message, &m.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// keep only latest 50 per room
func (r *Repository) TrimRoom(room string, keep int) error {
	_, err := r.db.Exec(`
		DELETE FROM chat_messages
		WHERE room = ?
		  AND id NOT IN (
			  SELECT id FROM chat_messages
			  WHERE room = ?
			  ORDER BY timestamp DESC
			  LIMIT ?
		  )`, room, room, keep)
	return err
}
