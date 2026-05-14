package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func ReadPump(hub *ChatHub, c *Client) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		// 🚨 THIS WAS MISSING
		hub.Broadcast <- ChatMessage{
			Type:      "chat",
			Room:      firstRoom(c),
			UserID:    c.ID,
			Username:  c.Username,
			Message:   string(msg),
			Timestamp: time.Now().Unix(),
		}
	}
}

func WritePump(c *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if !ok {
				log.Println("Client send channel closed")
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Write error:", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}

}
