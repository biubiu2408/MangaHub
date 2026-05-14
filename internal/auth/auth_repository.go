package auth

import (
	"database/sql"
	"errors"

	"github.com/biubiu2408/MangaHub/package/models"
)

type AuthRepository struct {
	DB *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{DB: db}
}

func (r *AuthRepository) CreateUser(username, passwordHash string, role string) error {
	_, err := r.DB.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		username, passwordHash, role,
	)
	return err
}

func (r *AuthRepository) FindByUsername(username string) (*models.User, error) {
	user := models.User{}
	err := r.DB.QueryRow(
		"SELECT id, username, password_hash, role FROM users WHERE username = ?",
		username,
	).Scan(&user.UserID, &user.Username, &user.PasswordHash, &user.Role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
