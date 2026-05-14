package tcp

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/biubiu2408/MangaHub/utils"
)

type Client struct {
	UserID   int64
	DeviceID string
	Conn     net.Conn
}
type Handshake struct {
	Token    string `json:"token"`
	DeviceID string `json:"device_id"`
}
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[string]*Client // userID → deviceID → client
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int64]map[string]*Client),
	}
}

func (h *Hub) AddClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[c.UserID]; !ok {
		h.clients[c.UserID] = make(map[string]*Client)
	}
	h.clients[c.UserID][c.DeviceID] = c
}

func (h *Hub) RemoveClient(userID int64, deviceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[userID]; !ok {
		return
	}
	delete(h.clients[userID], deviceID)
	fmt.Printf("Client: user %d  %s left the sync progress\n", userID, deviceID)
}
func (h *Hub) CountDevices(userID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[userID])
}
func (h *Hub) Broadcast(userID int64, payload any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	devices, ok := h.clients[userID]
	if !ok {
		return
	}

	for _, client := range devices {
		_ = json.NewEncoder(client.Conn).Encode(payload)
	}
}

func StartTCPServer(hub *Hub, port int) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, hub)
	}
}

func handleConnection(conn net.Conn, hub *Hub) {
	defer conn.Close()

	var hs Handshake
	if err := json.NewDecoder(conn).Decode(&hs); err != nil {
		return
	}

	claims, err := utils.ValidateToken(hs.Token)
	if err != nil {
		return
	}

	userID := claims.UserId

	client := &Client{
		UserID:   userID,
		DeviceID: hs.DeviceID,
		Conn:     conn,
	}

	hub.AddClient(client)
	fmt.Printf("User %d connected with device %s\n", userID, hs.DeviceID)
	defer hub.RemoveClient(userID, hs.DeviceID)

	dec := json.NewDecoder(conn)
	for {
		var msg any
		if err := dec.Decode(&msg); err != nil {
			return // client disconnected
		}
	}
}
