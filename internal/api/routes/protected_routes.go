package routes

import (
	"github.com/biubiu2408/MangaHub/internal/api/middleware"
	"github.com/biubiu2408/MangaHub/internal/user"
	"github.com/biubiu2408/MangaHub/package/database"
	"github.com/gin-gonic/gin"
)

func RegisterProtectedRoutes(router *gin.Engine) {

	router.Use(middleware.AuthMiddleware())
	userRepo := user.NewRepository(database.DB)
	userHandler := user.NewHandler(userRepo, nil) // TCP hub will be set in main.go

	// router.PATCH("/users/progress", userHandler.UpdateReadingProgress)        // mangahub progress update --manga-id <id> --current-chapter <chapter>
	router.GET("/users/progress/history", userHandler.GetProgressHistory) // mangahub progress history --manga-id <id>
	router.POST("/users/progress/sync", userHandler.SyncProgress)         // mangahub progress sync
	router.GET("/users/progress/sync-status", userHandler.GetSyncStatus)  // mangahub progress sync-status
	// udpRepo := udp.NewUDPRepository(database.DB)
	// udpHandler := udp.NewUDPHandler(udpRepo)
	router.POST("/users/library", userHandler.AddReadingEntry)                // mangahub library add
	router.PATCH("/users/library", userHandler.UpdateReadingStatus)           // mangahub library update --manga-id <id> --status <new-status>
	router.DELETE("/users/library", userHandler.DeleteReadingEntry)           // mangahub library remove --manga-id <id>
	router.GET("/users/library", userHandler.GetUserLibrary)                  // mangahub library list
	router.GET("/users/library/:status", userHandler.GetUserLibraryViaStatus) // mangahub library list --status=<status>

}
