package sv_grpc

import (
	"context"
	"fmt"
	"time"

	pb "github.com/biubiu2408/MangaHub/internal/grpc/manga"
	manga "github.com/biubiu2408/MangaHub/internal/manga"
	"github.com/biubiu2408/MangaHub/internal/tcp"
	"github.com/biubiu2408/MangaHub/internal/user"
	"github.com/biubiu2408/MangaHub/package/models"
)

type Server struct {
	pb.UnimplementedMangaServiceServer
	repo     manga.Repository
	userRepo *user.Repository
	tcpHub   *tcp.Hub
}

func NewServer(repo manga.Repository, userRepo *user.Repository, tcpHub *tcp.Hub) *Server {
	return &Server{
		repo:     repo,
		userRepo: userRepo,
		tcpHub:   tcpHub,
	}
}

/* ========== UC-014 ========== */

func (s *Server) GetManga(
	ctx context.Context,
	req *pb.GetMangaRequest,
) (*pb.GetMangaResponse, error) {

	manga, err := s.repo.GetByID(req.MangaId)
	if err != nil {
		return nil, fmt.Errorf("manga not found: %w", err)
	}

	return &pb.GetMangaResponse{
		Manga: &pb.Manga{
			Id:          manga.ID,
			Title:       manga.Title,
			Author:      manga.Author,
			Description: manga.Description,
		},
	}, nil
}

/* ========== UC-015 ========== */

func (s *Server) Search(
	ctx context.Context,
	req *pb.SearchMangaRequest,
) (*pb.SearchMangaResponse, error) {

	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize

	rows, err := s.repo.DB.Query(`
		SELECT id, title, author, description
		FROM mangas
		WHERE LOWER(title) LIKE '%' || LOWER(?) || '%'
		LIMIT ? OFFSET ?
	`, req.Keyword, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*pb.Manga, 0)

	for rows.Next() {
		m := &pb.Manga{}
		if err := rows.Scan(
			&m.Id,
			&m.Title,
			&m.Author,
			&m.Description,
		); err != nil {
			return nil, err
		}
		results = append(results, m)
	}

	var total int64
	_ = s.repo.DB.QueryRow(`
		SELECT COUNT(*)
		FROM mangas
		WHERE LOWER(title) LIKE '%' || LOWER(?) || '%'
	`, req.Keyword).Scan(&total)

	return &pb.SearchMangaResponse{
		Results: results,
		Total:   total,
	}, nil

}

/* ========== UC-016 ========== */
func (s *Server) UpdateProgress(
	ctx context.Context,
	req *pb.UpdateProgressRequest,
) (*pb.UpdateProgressResponse, error) {

	if req.UserId == 0 || req.MangaId == "" {
		return nil, fmt.Errorf("invalid request: user_id and manga_id are required")
	}

	if req.Chapter <= 0 {
		return nil, fmt.Errorf("current_chapter must be > 0")
	}

	// Check manga in library
	exists, err := s.userRepo.IsMangaInUserLibrary(req.UserId, req.MangaId)
	if err != nil {
		return nil, fmt.Errorf("failed to check library: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("manga not found in user library, add to library first")
	}

	// Validate chapter number
	totalChapters, err := s.userRepo.GetMangaTotalChapters(req.MangaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get total chapters: %w", err)
	}

	if int(req.Chapter) > totalChapters {
		return nil, fmt.Errorf("chapter %d exceeds manga's total chapters (%d). Valid range: 1-%d",
			req.Chapter, totalChapters, totalChapters)
	}

	// Get previous progress
	prevEntry, err := s.userRepo.GetReadingEntry(req.UserId, req.MangaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous progress: %w", err)
	}

	// Validate backward progress
	if int(req.Chapter) < prevEntry.CurrentChapter {
		return nil, fmt.Errorf("chapter is behind current progress, run 'history' to view past progress")
	}

	// Log reading progress if moving forward
	if int(req.Chapter) > prevEntry.CurrentChapter {
		currentDate := time.Now()
		_ = s.userRepo.LogReadingProgress(
			req.UserId,
			req.MangaId,
			int(req.Chapter),
			currentDate,
		)
	}

	// Update progress
	entry := models.ReadingEntry{
		MangaID:        req.MangaId,
		CurrentChapter: int(req.Chapter),
		LastUpdated:    time.Now(),
	}

	if err := s.userRepo.UpdateReadingProgress(req.UserId, entry); err != nil {
		return nil, fmt.Errorf("failed to update progress: %w", err)
	}

	// Get updated progress
	newEntry, err := s.userRepo.GetReadingEntry(req.UserId, req.MangaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated progress: %w", err)
	}

	// Broadcast to TCP devices
	s.tcpHub.Broadcast(req.UserId, user.ProgressUpdateMessage{
		Type:          "reading_progress_updated",
		MangaID:       req.MangaId,
		Previous:      prevEntry.CurrentChapter,
		Current:       newEntry.CurrentChapter,
		UpdatedAt:     newEntry.LastUpdated,
		DevicesSynced: s.tcpHub.CountDevices(req.UserId),
	})

	return &pb.UpdateProgressResponse{
		Success: true,
	}, nil
}
