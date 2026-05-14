package models

type Manga struct {
	ID            string   `json:"id" db:"id"`
	Title         string   `json:"title" db:"title"`
	Author        string   `json:"author" db:"author"`
	Artist        string   `json:"artist" db:"artist"`
	Genres        []string `json:"genres" db:"genres"`
	ChapterCount  int      `json:"chapter_count" db:"chapter_count"`
	VolumeCount   int      `json:"volume_count" db:"volume_count"`
	PublishedYear int      `json:"published_year" db:"published_year"`
	Popularity    int      `json:"popularity" db:"popularity"`
	Ranking       int      `json:"ranking" db:"ranking"`
	Status        string   `json:"status" db:"status"`
	CoverURL      string   `json:"cover_url" db:"cover_url"`
	Description   string   `json:"description" db:"description"`
}
type PaginatedMangas struct {
	Items      []Manga
	TotalItems int
	TotalPages int
}
type PaginatedMangasResponse struct {
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_pages"`
	TotalItems int     `json:"total_items"`
	Items      []Manga `json:"items"`
}
