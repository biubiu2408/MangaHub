package websocket_client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type ChatMessage struct {
	Type      string `json:"type"` // chat | system | presence
	Room      string `json:"room"`
	UserID    int64  `json:"user_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Online    int    `json:"online,omitempty"`
}

func ReadMessages(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("🔌 Connection closed.")
				return
			}

			var m ChatMessage
			if err := json.Unmarshal(msg, &m); err != nil {
				// fallback: raw message
				fmt.Println(string(msg))
				continue
			}

			// Format time (if exists)
			ts := ""
			if m.Timestamp > 0 {
				ts = time.Unix(m.Timestamp, 0).Format("15:04")
			}

			switch m.Type {

			case "presence":
				fmt.Printf("👥 Online users: %d\n", m.Online)

			case "system":
				if ts != "" {
					fmt.Printf("[%s] 🔔 %s\n", ts, m.Message)
				} else {
					fmt.Printf("🔔 %s\n", m.Message)
				}

			case "chat", "":
				// default to chat for backward compatibility
				if ts != "" {
					fmt.Printf("[%s] %s: %s\n", ts, m.Username, m.Message)
				} else {
					fmt.Printf("%s: %s\n", m.Username, m.Message)
				}

			default:
				// Unknown type → print safely
				fmt.Println(string(msg))
			}
		}
	}
}

func WriteMessages(ctx context.Context, conn *websocket.Conn) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !scanner.Scan() {
				return
			}

			text := scanner.Text()
			if text == "" {
				continue
			}

			if err := conn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
				fmt.Println("❌ Failed to send message")
				return
			}
		}
	}
}
