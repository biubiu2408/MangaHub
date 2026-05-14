package manga

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/biubiu2408/MangaHub/package/models"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) GetAll(page int, pageSize int, sortBy string) (*models.PaginatedMangas, error) {
	var totalItems int
	err := r.DB.QueryRow(`SELECT COUNT(*) FROM mangas`).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))

	// Determine ORDER BY clause based on sortBy parameter
	orderBy := "id" // default
	switch strings.ToLower(sortBy) {
	case "ranking":
		orderBy = "ranking ASC"
	case "popularity":
		orderBy = "popularity ASC"
	case "title":
		orderBy = "title ASC"
	default:
		orderBy = "id"
	}

	// 2. Fetch paginated rows with sorting
	query := fmt.Sprintf(`
		SELECT id, title, author, artist, genres, chapter_count,
		       published_year, status, cover_url, description, ranking, popularity
		FROM mangas
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, orderBy)

	rows, err := r.DB.Query(query, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []models.Manga

	for rows.Next() {
		var m models.Manga
		var genresJSON sql.NullString

		if err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.Author,
			&m.Artist,
			&genresJSON,
			&m.ChapterCount,
			&m.PublishedYear,
			&m.Status,
			&m.CoverURL,
			&m.Description,
			&m.Ranking,
			&m.Popularity,
		); err != nil {
			return nil, err
		}

		if genresJSON.Valid {
			_ = json.Unmarshal([]byte(genresJSON.String), &m.Genres)
		}

		mangas = append(mangas, m)
	}

	return &models.PaginatedMangas{
		Items:      mangas,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}
func (r *Repository) GetByID(id string) (*models.Manga, error) {
	row := r.DB.QueryRow(`
		SELECT id, title, author, artist, genres, chapter_count, volume_count, published_year, status, cover_url, description, ranking, popularity
		FROM mangas WHERE id = ?
	`, id)

	var m models.Manga
	var genresJSON string
	if err := row.Scan(
		&m.ID, &m.Title, &m.Author, &m.Artist, &genresJSON,
		&m.ChapterCount, &m.VolumeCount, &m.PublishedYear, &m.Status,
		&m.CoverURL, &m.Description, &m.Ranking, &m.Popularity,
	); err != nil {
		return nil, err
	}

	if genresJSON != "" {
		_ = json.Unmarshal([]byte(genresJSON), &m.Genres)
	}
	return &m, nil
}

func (r *Repository) Search(query string) ([]models.Manga, error) {
	searchTerm := strings.ToLower(query)
	rows, err := r.DB.Query(`
		SELECT id, title, author, artist, genres, chapter_count,
       	published_year, status, cover_url, description
		FROM mangas
		WHERE
		LOWER(title) LIKE '%' || LOWER(?) || '%'
		OR LOWER(id) LIKE '%' || LOWER(?) || '%'

	`, searchTerm, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []models.Manga
	for rows.Next() {
		var m models.Manga
		var genresJSON string

		if err := rows.Scan(
			&m.ID, &m.Title, &m.Author, &m.Artist, &genresJSON,
			&m.ChapterCount, &m.PublishedYear, &m.Status,
			&m.CoverURL, &m.Description,
		); err != nil {
			return nil, err
		}

		if genresJSON != "" {
			_ = json.Unmarshal([]byte(genresJSON), &m.Genres)
		}
		mangas = append(mangas, m)
	}
	return mangas, nil
}

func (r *Repository) FilterByGenre(
	genres []string,
	page int,
	pageSize int,
	sortBy string,
) (*models.PaginatedMangas, error) {

	if len(genres) == 0 {
		return &models.PaginatedMangas{
			Items:      []models.Manga{},
			TotalItems: 0,
			TotalPages: 0,
		}, nil
	}

	// Build placeholders (?, ?, ?)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(genres)), ",")

	// Prepare args
	args := make([]interface{}, 0, len(genres)+1)
	for _, g := range genres {
		args = append(args, strings.ToLower(g))
	}

	// Determine ORDER BY clause based on sortBy parameter
	orderBy := "id" // default
	switch strings.ToLower(sortBy) {
	case "ranking":
		orderBy = "ranking ASC"
	case "popularity":
		orderBy = "popularity ASC"
	case "title":
		orderBy = "title ASC"
	default:
		orderBy = "id"
	}

	// -----------------------------
	// 1. COUNT query
	// -----------------------------
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM mangas
		WHERE (
			SELECT COUNT(DISTINCT json_each.value)
			FROM json_each(mangas.genres)
			WHERE LOWER(json_each.value) IN (%s)
		) = ?
	`, placeholders)

	var totalItems int
	err := r.DB.QueryRow(countQuery, append(args, len(genres))...).Scan(&totalItems)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	offset := (page - 1) * pageSize

	// -----------------------------
	// 2. DATA query with sorting
	// -----------------------------
	dataQuery := fmt.Sprintf(`
		SELECT id, title, author, artist, genres, chapter_count,
		       published_year, status, cover_url, description, ranking, popularity
		FROM mangas
		WHERE (
			SELECT COUNT(DISTINCT json_each.value)
			FROM json_each(mangas.genres)
			WHERE LOWER(json_each.value) IN (%s)
		) = ?
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, placeholders, orderBy)

	dataArgs := append(args, len(genres), pageSize, offset)

	rows, err := r.DB.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangas []models.Manga

	for rows.Next() {
		var m models.Manga
		var genresJSON sql.NullString

		if err := rows.Scan(
			&m.ID,
			&m.Title,
			&m.Author,
			&m.Artist,
			&genresJSON,
			&m.ChapterCount,
			&m.PublishedYear,
			&m.Status,
			&m.CoverURL,
			&m.Description,
			&m.Ranking,
			&m.Popularity,
		); err != nil {
			return nil, err
		}

		if genresJSON.Valid {
			_ = json.Unmarshal([]byte(genresJSON.String), &m.Genres)
		}

		mangas = append(mangas, m)
	}

	return &models.PaginatedMangas{
		Items:      mangas,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}, nil
}

func (r *Repository) UpdateManga(m models.Manga) (bool, error) {
	var genresJSON string
	if len(m.Genres) > 0 {
		data, err := json.Marshal(m.Genres)
		if err != nil {
			return false, err
		}
		genresJSON = string(data)
	}
	_, err := r.DB.Exec(`
		UPDATE mangas
		SET title = ?, author = ?, artist = ?, genres = ?, chapter_count = ?,
		published_year = ?, status = ?, cover_url = ?, description = ?
		WHERE id = ?
	`, m.Title, m.Author, m.Artist, genresJSON, m.ChapterCount,
		m.PublishedYear, m.Status, m.CoverURL, m.Description, m.ID,
	)
	if err != nil {
		return false, err
	}
	return true, nil

}
func (r *Repository) UpdateMangaChapterRelease(id string, chapterCount int) (bool, error) {
	_, err := r.DB.Exec(`
		UPDATE mangas
		SET chapter_count = ?
		WHERE id = ?
	`, chapterCount, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) ExistsByTitle(title string) (bool, error) {
	row := r.DB.QueryRow(`
		SELECT COUNT(1)
		FROM mangas
		WHERE id = ?
	`, title)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
