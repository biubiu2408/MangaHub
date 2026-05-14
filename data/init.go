package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/biubiu2408/MangaHub/internal/auth"
	"github.com/biubiu2408/MangaHub/package/database"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Read DB path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/mangahub.db"
	}

	database.InitSQLite(dbPath)

	populateManga()
	populateUsers()
	fmt.Println("Database created and populated successfully.")
	// err := ExportMangaToJSON(database.DB, "mangas_export.json")
	// if err != nil {
	// 	log.Fatal(err)
	// }

}

type MangaMeta struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Artist        string   `json:"artist"`
	Genres        []string `json:"genres"`
	ChapterCount  *int     `json:"chapter_count"`
	VolumeCount   *int     `json:"volume_count"`
	PublishedYear *int     `json:"published_year"`
	Status        string   `json:"status"`
	CoverURL      string   `json:"cover_url"`
	Description   string   `json:"description"`
	Popularity    *int     `json:"popularity"`
	Ranking       *int     `json:"ranking"`
}

func populateManga() {
	file, err := os.ReadFile("data/mangas_export.json")
	if err != nil {
		log.Fatalf("failed to read json: %v", err)
	}

	var mangas []MangaMeta
	if err := json.Unmarshal(file, &mangas); err != nil {
		log.Fatalf("failed to unmarshal json: %v", err)
	}

	stmt := `
INSERT INTO mangas (
  id,
  title,
  author,
  artist,
  genres,
  chapter_count,
  volume_count,
  published_year,
  status,
  cover_url,
  description,
  popularity,
  ranking
) VALUES (
  @id,
  @title,
  @author,
  @artist,
  @genres,
  @chapter_count,
  @volume_count,
  @published_year,
  @status,
  @cover_url,
  @description,
  @popularity,
  @ranking
)
ON CONFLICT(id) DO UPDATE SET
  title          = excluded.title,
  author         = excluded.author,
  artist         = excluded.artist,
  genres         = excluded.genres,
  chapter_count  = excluded.chapter_count,
  volume_count   = excluded.volume_count,
  published_year = excluded.published_year,
  status         = excluded.status,
  cover_url      = excluded.cover_url,
  description    = excluded.description,
  popularity     = excluded.popularity,
  ranking        = excluded.ranking;
`

	for _, m := range mangas {
		genresJSON, _ := json.Marshal(m.Genres)

		_, err := database.DB.Exec(
			stmt,
			sql.Named("id", m.ID),
			sql.Named("title", m.Title),
			sql.Named("author", m.Author),
			sql.Named("artist", m.Artist),
			sql.Named("genres", string(genresJSON)),
			sql.Named("chapter_count", m.ChapterCount),
			sql.Named("volume_count", m.VolumeCount),
			sql.Named("published_year", m.PublishedYear),
			sql.Named("status", m.Status),
			sql.Named("cover_url", m.CoverURL),
			sql.Named("description", m.Description),
			sql.Named("popularity", m.Popularity),
			sql.Named("ranking", m.Ranking),
		)

		if err != nil {
			log.Printf("❌ Failed to import %s: %v", m.ID, err)
		}
	}

	log.Println("✅ Manga seed import completed")
}

func ExportMangaToJSON(db *sql.DB, outFile string) error {
	rows, err := db.Query(`
		SELECT
			id,
			title,
			author,
			artist,
			genres,
			chapter_count,
			volume_count,
			published_year,
			status,
			cover_url,
			description,
			popularity,
			ranking
		FROM mangas
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var result []MangaMeta

	for rows.Next() {
		var (
			m      MangaMeta
			rawGen string

			artist sql.NullString
		)

		err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.Author,
			&artist,
			&rawGen, // 👈 JSON TEXT
			&m.ChapterCount,
			&m.VolumeCount,
			&m.PublishedYear,
			&m.Status,
			&m.CoverURL,
			&m.Description,
			&m.Popularity,
			&m.Ranking,
		)
		if err != nil {
			return err
		}
		m.Artist = artist.String

		// 🔥 Decode JSON string into []string
		if err := json.Unmarshal([]byte(rawGen), &m.Genres); err != nil {
			return fmt.Errorf("invalid genres JSON for %s: %v", m.ID, err)
		}

		result = append(result, m)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outFile, data, 0644)
}

func populateUsers() {
	// Create auth repository
	authRepo := auth.NewAuthRepository(database.DB)

	// Define default users
	users := []struct {
		username string
		password string
		role     string
	}{
		{username: "admin", password: "admin123", role: "admin"},
		{username: "user", password: "user123", role: "user"},
	}

	log.Println("🔐 Populating users...")

	for _, u := range users {
		// Check if user already exists
		existingUser, err := authRepo.FindByUsername(u.username)
		if err != nil {
			log.Printf("❌ Error checking user %s: %v", u.username, err)
			continue
		}

		if existingUser != nil {
			log.Printf("⚠️  User %s already exists, skipping...", u.username)
			continue
		}

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("❌ Failed to hash password for %s: %v", u.username, err)
			continue
		}

		// Create the user
		err = authRepo.CreateUser(u.username, string(hashedPassword), u.role)
		if err != nil {
			log.Printf("❌ Failed to create user %s: %v", u.username, err)
			continue
		}

		log.Printf("✅ Created %s user: %s (password: %s)", u.role, u.username, u.password)
	}

	log.Println("✅ User seed import completed")
}
