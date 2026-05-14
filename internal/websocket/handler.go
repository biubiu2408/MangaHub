package websocket

import (
	"fmt"
	"net/http"

	"github.com/biubiu2408/MangaHub/internal/manga"
	"github.com/biubiu2408/MangaHub/package/database"
	"github.com/biubiu2408/MangaHub/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func ServeWS(hub *ChatHub) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := utils.GetUserIdFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		username, err := utils.GetUserNameFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"username error": err.Error()})
			return
		}
		fmt.Printf("Username from context: %s", username)
		room := c.Query("room")
		if room == "" {
			room = "general"
		}
		mangaRepo := manga.NewRepository(database.DB)
		exists, err := mangaRepo.ExistsByTitle(room)
		if err != nil {
			c.JSON(500, gin.H{"error": "db error"})
			return
		}
		if !exists {
			c.JSON(404, gin.H{"error": "manga not found"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := &Client{
			ID:       userID,
			Username: username,
			Conn:     conn,
			Send:     make(chan []byte, 256),
			Rooms:    make(map[string]bool),
		}

		hub.Register <- client

		hub.JoinRoom <- RoomAction{Client: client, Room: room}

		go WritePump(client)
		ReadPump(hub, client)
	}
}
