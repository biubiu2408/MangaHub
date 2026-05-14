package models

import "time"

type User struct {
	UserID       int64         `json:"user_id" db:"id"`
	Username     string        `json:"username" db:"username"`
	PasswordHash string        `json:"password_hash" db:"password_hash"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	ReadingLists *ReadingLists `json:"reading_lists,omitempty"`
	Role         string        `json:"role" db:"role"`
}
type ReadingEntry struct {
	MangaID        string    `json:"manga_id" db:"manga_id"`
	CurrentChapter int       `json:"current_chapter" db:"current_chapter"`
	Volume         *int      `json:"volume,omitempty" db:"volume"`
	Notes          *string   `json:"notes,omitempty" db:"notes"`
	Status         string    `json:"status" db:"status"`
	LastUpdated    time.Time `json:"last_updated" db:"last_updated"`
}

type ReadingLists struct {
	Reading    []ReadingEntry `json:"reading"`
	Completed  []ReadingEntry `json:"completed"`
	PlanToRead []ReadingEntry `json:"plan_to_read"`
}

type ReadingLog struct {
	UserID   int64  `json:"user_id" db:"user_id"`
	MangaID  string `json:"manga_id" db:"manga_id"`
	Chapter  int    `json:"chapter" db:"chapter"`
	DateRead string `json:"date_read" db:"date_read"`
}
