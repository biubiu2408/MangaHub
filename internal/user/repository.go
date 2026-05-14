package user

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/biubiu2408/MangaHub/package/models"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) MangaExists(mangaID string) (bool, error) {
	var count int
	err := r.DB.QueryRow(`
        SELECT COUNT(*)
        FROM mangas
        WHERE id = ?
    `, mangaID).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
func (r *Repository) AddReadingEntry(userID int64, entry models.ReadingEntry) error {
	_, err := r.DB.Exec(`
		INSERT INTO reading_list (user_id, manga_id, current_chapter, status, last_updated)
		VALUES (?, ?, ?, ?, ?)
	`, userID, entry.MangaID, entry.CurrentChapter, entry.Status, entry.LastUpdated.Format(time.RFC3339))
	return err
}
func (r *Repository) ReadingEntryExists(userID int64, mangaID string) (bool, error) {
	var count int
	err := r.DB.QueryRow(`
        SELECT COUNT(*) FROM reading_list
        WHERE user_id = ? AND manga_id = ?
    `, userID, mangaID).Scan(&count)

	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *Repository) UpdateReadingStatus(userID int64, entry models.ReadingEntry) error {
	if entry.CurrentChapter != 0 {
		_, err := r.DB.Exec(`
		UPDATE reading_list
		SET status = ?, current_chapter = ?,  last_updated = ?
		WHERE user_id = ? AND manga_id = ?
	`, entry.Status, entry.CurrentChapter, entry.LastUpdated.Format(time.RFC3339), userID, entry.MangaID)
		return err
	}
	_, err := r.DB.Exec(`
		UPDATE reading_list
		SET status = ?, last_updated = ?
		WHERE user_id = ? AND manga_id = ?
	`, entry.Status, entry.LastUpdated.Format(time.RFC3339), userID, entry.MangaID)
	return err

}
func (r *Repository) UpdateReadingProgress(userID int64, entry models.ReadingEntry) error {
	if entry.Status != "" {
		_, err := r.DB.Exec(`
		UPDATE reading_list
		SET status = ?, current_chapter = ?,  last_updated = ? 
		WHERE user_id = ? AND manga_id = ? 
	`, entry.Status, entry.CurrentChapter, entry.LastUpdated.Format(time.RFC3339), userID, entry.MangaID)
		return err
	}
	_, err := r.DB.Exec(`
		UPDATE reading_list
		SET current_chapter = ?, last_updated = ?, volume = COALESCE(?, volume), notes = COALESCE(?, notes), status = "reading"
		WHERE user_id = ? AND manga_id = ?
	`, entry.CurrentChapter, entry.LastUpdated.Format(time.RFC3339), entry.Volume, entry.Notes, userID, entry.MangaID)
	return err

}
func (r *Repository) DeleteReadingEntry(userID int64, entry models.ReadingEntry) error {
	_, err := r.DB.Exec(`
		DELETE FROM reading_list
		WHERE user_id = ? AND manga_id = ?
	`, userID, entry.MangaID)
	return err
}
func (r *Repository) IsMangaInUserLibrary(userID int64, mangaID string) (bool, error) {
	var count int
	err := r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM reading_list
		WHERE user_id = ? AND manga_id = ?
	`, userID, mangaID).Scan(&count)
	if err != nil {
		return false, errors.New("failed to check manga in user library: " + err.Error())
	}
	return count > 0, nil
}
func (r *Repository) GetUserReadingLists(userID int64) (*models.ReadingLists, error) {
	user := models.User{}
	err := r.DB.QueryRow(`SELECT id, username FROM users WHERE id = ?`, userID).
		Scan(&user.UserID, &user.Username)
	if err != nil {
		return nil, err
	}

	rows, err := r.DB.Query(`
		SELECT manga_id, current_chapter, status, last_updated
		FROM reading_list WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reading, completed, plan []models.ReadingEntry
	for rows.Next() {
		var entry models.ReadingEntry
		var lastUpdated string
		if err := rows.Scan(&entry.MangaID, &entry.CurrentChapter, &entry.Status, &lastUpdated); err != nil {
			return nil, err
		}
		entry.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)

		switch entry.Status {
		case "reading":
			reading = append(reading, entry)
		case "completed":
			completed = append(completed, entry)
		case "plan_to_read":
			plan = append(plan, entry)
		}
	}

	user.ReadingLists = &models.ReadingLists{
		Reading:    reading,
		Completed:  completed,
		PlanToRead: plan,
	}

	return user.ReadingLists, nil
}
func (r *Repository) GetUserReadingListsViaStatus(userID int64, status string) ([]models.ReadingEntry, error) {
	if status != "reading" && status != "completed" && status != "plan_to_read" {
		return nil, errors.New("invalid status value")
	}
	rows, err := r.DB.Query(`
		SELECT manga_id, current_chapter, status, last_updated
		FROM reading_list WHERE user_id = ? AND status = ?
	`, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.ReadingEntry
	for rows.Next() {
		var entry models.ReadingEntry
		var lastUpdated string
		if err := rows.Scan(&entry.MangaID, &entry.CurrentChapter, &entry.Status, &lastUpdated); err != nil {
			return nil, err
		}
		entry.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)

		switch status {
		case "reading":
			entries = append(entries, entry)
		case "completed":
			entries = append(entries, entry)
		case "plan_to_read":
			entries = append(entries, entry)
		}
	}

	// readingList := &models.ReadingLists{
	// 	Reading:    reading,
	// 	Completed:  completed,
	// 	PlanToRead: plan,
	// }

	return entries, nil
}

func (r *Repository) GetMangaTotalChapters(mangaID string) (int, error) {
	var totalChapters int
	err := r.DB.QueryRow(
		"SELECT chapter_count FROM mangas WHERE id = ?",
		mangaID,
	).Scan(&totalChapters)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("manga '%s' not found", mangaID)
	} else if err != nil {
		return 0, err
	}

	return totalChapters, nil
}

func (r *Repository) GetReadingEntry(
	userID int64,
	mangaID string,
) (*models.ReadingEntry, error) {

	var entry models.ReadingEntry
	var lastUpdated string

	err := r.DB.QueryRow(`
		SELECT manga_id, current_chapter, status, last_updated
		FROM reading_list
		WHERE user_id = ? AND manga_id = ?
	`, userID, mangaID).Scan(
		&entry.MangaID,
		&entry.CurrentChapter,
		&entry.Status,
		&lastUpdated,
	)

	if err != nil {
		return nil, err
	}

	entry.LastUpdated, _ = time.Parse(time.RFC3339, lastUpdated)
	return &entry, nil
}

func (r *Repository) LogReadingProgress(
	userID int64,
	mangaID string,
	chapter int,
	dateRead time.Time,
) error {
	currentDate := time.Now().Format("2006-01-02")

	_, err := r.DB.Exec(`
        INSERT INTO reading_logs (user_id, manga_id, chapter, date_read)
        VALUES (?, ?, ?, ?)
    `, userID, mangaID, chapter, currentDate)

	return err
}

func (r *Repository) GetReadingStreak(userID int64) (int, error) {
	rows, err := r.DB.Query(`
		SELECT DISTINCT date_read
		FROM reading_logs
		WHERE user_id = ?
		ORDER BY date_read DESC
	`, userID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return 0, err
		}
		dates = append(dates, d)
	}

	return calculateStreak(dates), nil
}

// Calculate streak from a list of dates
func calculateStreak(dates []time.Time) int {
	if len(dates) == 0 {
		return 0
	}

	reading_streak := 1

	for i := 1; i < len(dates); i++ {
		if dates[i-1].Add(24 * time.Hour).Equal(dates[i]) {
			reading_streak++
		} else {
			break
		}
	}

	return reading_streak
}

func (r *Repository) GetReadingHistory(userID int64, mangaID *string) ([]models.ReadingLog, error) {
	query := `
        SELECT user_id, manga_id, chapter, date_read
        FROM reading_logs
        WHERE user_id = ?
    `
	args := []any{userID}

	if mangaID != nil {
		query += " AND manga_id = ?"
		args = append(args, *mangaID)
	}

	query += " ORDER BY date_read DESC"

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.ReadingLog
	for rows.Next() {
		var log models.ReadingLog
		if err := rows.Scan(
			&log.UserID,
			&log.MangaID,
			&log.Chapter,
			&log.DateRead,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// SyncReadingProgress save the last sync time
func (r *Repository) SyncReadingProgress(userID int64) error {
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err := r.DB.Exec(`
        INSERT INTO sync_state (user_id, last_synced_at)
        VALUES (?, ?)
        ON CONFLICT(user_id)
        DO UPDATE SET last_synced_at = excluded.last_synced_at
    `, userID, now)

	return err
}

// GetSyncStatus retrieve the last sync time for a user
func (r *Repository) GetSyncStatus(userID int64) (map[string]string, error) {
	row := r.DB.QueryRow(`
        SELECT last_synced_at
        FROM sync_state
        WHERE user_id = ?
    `, userID)

	var lastSync string
	err := row.Scan(&lastSync)
	if err == sql.ErrNoRows {
		return map[string]string{
			"status": "never synced",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"status":         "up-to-date",
		"last_synced_at": lastSync,
	}, nil
}
