package user

import (
	"fmt"
	"net/http"
	"time"

	"github.com/biubiu2408/MangaHub/internal/tcp"
	"github.com/biubiu2408/MangaHub/package/models"
	"github.com/biubiu2408/MangaHub/utils"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo   *Repository
	tcpHub *tcp.Hub
}
type ProgressUpdateMessage struct {
	Type          string    `json:"type"`
	MangaID       string    `json:"manga_id"`
	Previous      int       `json:"previous_chapter"`
	Current       int       `json:"current_chapter"`
	UpdatedAt     time.Time `json:"updated_at"`
	DevicesSynced int       `json:"devices_synced"`
}

func NewHandler(repo *Repository, tcpHub *tcp.Hub) *Handler {
	return &Handler{repo: repo, tcpHub: tcpHub}
}

func (h *Handler) AddReadingEntry(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var entry models.ReadingEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if entry.MangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id is required"})
		return
	}

	// check if manga exists
	exists, err := h.repo.MangaExists(entry.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id does not exist"})
		return
	}

	// check duplicate entry
	existsRL, err := h.repo.ReadingEntryExists(userID, entry.MangaID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database error"})
		return
	}
	if existsRL {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga already added to library"})
		return
	}

	entry.LastUpdated = time.Now()

	if err := h.repo.AddReadingEntry(userID, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "reading entry added"})
}
func (h *Handler) UpdateReadingStatus(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var entry models.ReadingEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if entry.MangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id is required"})
		return
	}
	entry.LastUpdated = time.Now()

	var mangaExists bool
	mangaExists, err = h.repo.IsMangaInUserLibrary(userID, entry.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !mangaExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "manga not found in user library"})
		return
	}
	if entry.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be provided"})
		return
	}

	if err := h.repo.UpdateReadingStatus(userID, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reading entry updated"})
}
func (h *Handler) UpdateReadingProgress(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var entry models.ReadingEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if entry.MangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id is required"})
		return
	}
	if entry.CurrentChapter <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current_chapter must be > 0"})
		return
	}

	// check manga in library
	exists, err := h.repo.IsMangaInUserLibrary(userID, entry.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "manga not found in user library, add to library first"})
		return
	}
	// validate chapter number
	totalChapters, err := h.repo.GetMangaTotalChapters(entry.MangaID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if entry.CurrentChapter > totalChapters {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Chapter %d exceeds manga's total chapters (%d).Valid range: 1-%d", entry.CurrentChapter, totalChapters, totalChapters),
		})
		return
	}

	// get previous progress
	prevEntry, err := h.repo.GetReadingEntry(userID, entry.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get previous progress"})
		return
	}

	// validate backward progress (force xử lý ở CLI hoặc thêm flag sau)
	if entry.CurrentChapter < prevEntry.CurrentChapter {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "chapter is behind current progress, run 'history' to view past progress",
		})
		return
	}

	if entry.CurrentChapter > prevEntry.CurrentChapter {
		currentDate := time.Now()

		_ = h.repo.LogReadingProgress(
			userID,
			entry.MangaID,
			entry.CurrentChapter,
			currentDate,
		)
	}

	// update
	entry.LastUpdated = time.Now()
	if err := h.repo.UpdateReadingProgress(userID, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// get updated progress
	newEntry, err := h.repo.GetReadingEntry(userID, entry.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated progress"})
		return
	}
	readingStreak, _ := h.repo.GetReadingStreak(userID)

	c.JSON(http.StatusOK, gin.H{
		"manga_title":         entry.MangaID, // nếu có bảng manga → join lấy title
		"previous_chapter":    prevEntry.CurrentChapter,
		"current_chapter":     newEntry.CurrentChapter,
		"updated_at":          newEntry.LastUpdated,
		"devices_synced":      h.tcpHub.CountDevices(userID),
		"total_chapters_read": newEntry.CurrentChapter,
		"reading_streak":      readingStreak,
		"next_chapter":        newEntry.CurrentChapter + 1,
	})
	// notify via TCP
	h.tcpHub.Broadcast(userID, ProgressUpdateMessage{
		Type:          "reading_progress_updated",
		MangaID:       entry.MangaID,
		Previous:      prevEntry.CurrentChapter,
		Current:       newEntry.CurrentChapter,
		UpdatedAt:     newEntry.LastUpdated,
		DevicesSynced: h.tcpHub.CountDevices(userID),
	})

}

func (h *Handler) DeleteReadingEntry(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var entry models.ReadingEntry
	if err := c.ShouldBindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if entry.MangaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "manga_id is required"})
		return
	}
	entry.LastUpdated = time.Now()

	var mangaExists bool
	mangaExists, err = h.repo.IsMangaInUserLibrary(userID, entry.MangaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !mangaExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "manga not found in user library"})
		return
	}
	if err := h.repo.DeleteReadingEntry(userID, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "manga entry " + entry.MangaID + " deleted"})
}
func (h *Handler) GetUserLibrary(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	user, err := h.repo.GetUserReadingLists(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}
func (h *Handler) GetUserLibraryViaStatus(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	status := c.Param("status")

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	readingLists, err := h.repo.GetUserReadingListsViaStatus(userID, status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "reading lists not found"})
		return
	}
	c.JSON(http.StatusOK, readingLists)
}

func (h *Handler) GetProgressHistory(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	mangaID := c.Query("manga_id")
	var mid *string
	if mangaID != "" {
		mid = &mangaID
	}

	history, err := h.repo.GetReadingHistory(userID, mid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch reading history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"history": history,
	})
}

func (h *Handler) SyncProgress(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	err = h.repo.SyncReadingProgress(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "sync completed",
	})
}

func (h *Handler) GetSyncStatus(c *gin.Context) {
	userID, err := utils.GetUserIdFromContext(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	status, err := h.repo.GetSyncStatus(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, status)
}
