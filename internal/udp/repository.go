package udp

import (
	"database/sql"
)

type UDPRepository struct {
	DB *sql.DB
}

func NewUDPRepository(db *sql.DB) *UDPRepository {
	return &UDPRepository{DB: db}
}

func (r *UDPRepository) CreateNotificationEntry(userID int64, addr string) error {
	_, err := r.DB.Exec(`
        INSERT INTO notifications (user_id, client_udp_addr)
        VALUES (?, ?)
        ON CONFLICT(user_id)
        DO UPDATE SET client_udp_addr = excluded.client_udp_addr
    `, userID, addr)
	return err
}
func (r *UDPRepository) CreateSubscriptionEntry(user_id int64, manga_id string) error {
	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM mangas WHERE  id = ?",
		manga_id,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	_, err = r.DB.Exec(
		"INSERT INTO subscriptions (user_id, manga_id) VALUES (?, ?)",
		user_id, manga_id,
	)
	return err
}
func (r *UDPRepository) SubscriptionExists(user_id int64, manga_id string) (bool, error) {
	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM subscriptions WHERE user_id = ? AND manga_id = ?",
		user_id, manga_id,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func (r *UDPRepository) GetUserUDPAddress(user_id int64) (string, error) {
	var client_udp_addr string
	err := r.DB.QueryRow(
		"SELECT client_udp_addr FROM notifications WHERE user_id = ?",
		user_id,
	).Scan(&client_udp_addr)
	if err != nil {
		return "", err
	}
	return client_udp_addr, nil
}

func (r *UDPRepository) UpdateNotificationEntry(user_id int64, client_udp_addr string) error {
	_, err := r.DB.Exec(
		"UPDATE notifications SET client_udp_addr = ? WHERE user_id = ?",
		client_udp_addr, user_id,
	)
	return err
}

func (r *UDPRepository) GetMangaSubscribers(manga_id string) ([]int64, error) {
	// get all users subscribed to this manga
	rows, err := r.DB.Query(
		"SELECT user_id FROM subscriptions WHERE manga_id = ?",
		manga_id,
	)
	if err != nil {
		return nil, err
	}
	var user_ids []int64
	defer rows.Close()
	for rows.Next() {
		var user_id int64
		if err := rows.Scan(&user_id); err != nil {
			return nil, err
		}
		user_ids = append(user_ids, user_id)
	}
	return user_ids, nil
}
func (r *UDPRepository) GetUsersUDPAddresses(user_id int64) string {
	var client_udp_addr string
	err := r.DB.QueryRow(
		"SELECT client_udp_addr FROM notifications WHERE user_id = ?",
		user_id,
	).Scan(&client_udp_addr)
	if err != nil {
		return ""
	}
	return client_udp_addr
}
func (r *UDPRepository) GetUserIdFromUsername(username string) (int64, error) {
	var user_id int64
	err := r.DB.QueryRow(
		"SELECT id FROM users WHERE username = ?",
		username,
	).Scan(&user_id)
	if err != nil {
		return 0, err
	}
	return user_id, nil
}
