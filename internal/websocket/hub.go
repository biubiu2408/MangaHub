package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID       int64
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Rooms    map[string]bool
}

type Room struct {
	Name    string
	Clients map[*Client]bool
	mu      sync.RWMutex
}
type ChatHub struct {
	Clients map[*Client]bool
	Rooms   map[string]*Room
	mu      sync.RWMutex

	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan ChatMessage
	JoinRoom   chan RoomAction
	LeaveRoom  chan RoomAction
	ChatRepo   *Repository
}
type ChatMessage struct {
	Type      string `json:"type"` // "chat" | "system" | "presence"
	Room      string `json:"room"`
	UserID    int64  `json:"user_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
	Online    int    `json:"online,omitempty"` // only for presence
}

type RoomAction struct {
	Client *Client
	Room   string
}

func NewChatHub() *ChatHub {
	return &ChatHub{
		Clients: make(map[*Client]bool),
		Rooms:   make(map[string]*Room),

		Broadcast:  make(chan ChatMessage, 256),
		Register:   make(chan *Client, 16),
		Unregister: make(chan *Client, 16),
		JoinRoom:   make(chan RoomAction, 16),
		LeaveRoom:  make(chan RoomAction, 16),
		mu:         sync.RWMutex{},
	}
}
func (hub *ChatHub) Run() {
	for {
		select {
		case client := <-hub.Register:
			hub.Clients[client] = true
		case client := <-hub.Unregister:
			if _, ok := hub.Clients[client]; ok {
				delete(hub.Clients, client)

				for room := range client.Rooms {
					hub.removeFromRoom(client, room)
				}

				close(client.Send) // ✅ correct place
			}

		case action := <-hub.JoinRoom:
			hub.addToRoom(action.Client, action.Room)

		case action := <-hub.LeaveRoom:
			hub.removeFromRoom(action.Client, action.Room)
		case msg := <-hub.Broadcast:
			hub.broadcastToRoom(msg)
		}
	}
}
func (hub *ChatHub) sendSystemMessage(room string, text string) {
	hub.Broadcast <- ChatMessage{
		Type:      "system",
		Room:      room,
		Message:   text,
		Timestamp: time.Now().Unix(),
	}
}
func (hub *ChatHub) broadcastPresence(room string) {
	hub.mu.RLock()
	r, ok := hub.Rooms[room]
	hub.mu.RUnlock()
	if !ok {
		return
	}

	r.mu.RLock()
	userSet := make(map[string]struct{})

	for client := range r.Clients {
		userSet[client.Username] = struct{}{}
	}

	count := len(userSet)

	r.mu.RUnlock()

	hub.Broadcast <- ChatMessage{
		Type:      "presence",
		Room:      room,
		Username:  "System",
		Online:    count,
		Message:   "online users updated",
		Timestamp: time.Now().Unix(),
	}
}

func (hub *ChatHub) addToRoom(c *Client, roomName string) {
	room, ok := hub.Rooms[roomName]
	if !ok {
		room = &Room{
			Name:    roomName,
			Clients: make(map[*Client]bool),
		}
		hub.Rooms[roomName] = room
	}

	room.mu.Lock()
	room.Clients[c] = true
	room.mu.Unlock()

	c.Rooms[roomName] = true

	// ✅ Send chat history to joining user
	history, err := hub.ChatRepo.GetLastMessages(roomName, 50)
	if err == nil {
		for i := len(history) - 1; i >= 0; i-- {
			data, _ := json.Marshal(ChatMessage{
				Room:      history[i].Room,
				UserID:    history[i].UserID,
				Username:  history[i].Username,
				Message:   history[i].Message,
				Timestamp: history[i].Timestamp,
			})
			c.Send <- data
		}
	}

	// 🔔 Notify others
	hub.sendSystemMessage(roomName,
		fmt.Sprintf("%s joined the room", c.Username),
	)

	// 👥 Update online count
	hub.broadcastPresence(roomName)

	log.Printf("[WS] %s joined %s", c.Username, roomName)
}

func (hub *ChatHub) removeFromRoom(client *Client, roomName string) {
	room, ok := hub.Rooms[roomName]
	if !ok {
		return
	}

	room.mu.Lock()
	delete(room.Clients, client)
	room.mu.Unlock()

	delete(client.Rooms, roomName)

	hub.sendSystemMessage(roomName,
		fmt.Sprintf("%s left the room", client.Username),
	)
	hub.broadcastPresence(roomName)
}

func (hub *ChatHub) broadcastToRoom(msg ChatMessage) {
	// 1. Save to DB
	_ = hub.ChatRepo.SaveMessage(Message{
		Room:      msg.Room,
		UserID:    msg.UserID,
		Username:  msg.Username,
		Message:   msg.Message,
		Type:      msg.Type,
		Online:    msg.Online,
		Timestamp: msg.Timestamp,
	})

	// 2. Trim to last 50
	_ = hub.ChatRepo.TrimRoom(msg.Room, 50)

	// 3. Broadcast as usual
	hub.mu.RLock()
	room, exists := hub.Rooms[msg.Room]
	hub.mu.RUnlock()
	if !exists {
		return
	}

	data, _ := json.Marshal(msg)
	room.mu.RLock()
	for client := range room.Clients {
		select {
		case client.Send <- data:
		default:
			delete(room.Clients, client)
		}
	}
	room.mu.RUnlock()
}
func firstRoom(c *Client) string {
	for r := range c.Rooms {
		return r
	}
	return "general"
}
