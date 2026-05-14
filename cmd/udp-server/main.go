package udpserver

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/biubiu2408/MangaHub/internal/udp"
	"github.com/biubiu2408/MangaHub/utils"
)

type NotificationSubscribeResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
type UDPRequest struct {
	Type string `json:"type"`
}

type DiscoverResponse struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

func StartUDPListener(port int, h *udp.UDPHandler) error {

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return err
	}
	defer conn.Close()

	fmt.Printf("UDP Listener running on port %d...\n", port)

	buffer := make([]byte, 2048)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Read error:", err)
			continue
		}

		raw := buffer[:n]
		fmt.Printf("📦 Received UDP packet from %s: %s\n", clientAddr, string(raw))

		var req udp.UDPClientRequest
		if err := json.Unmarshal(raw, &req); err == nil {
			fmt.Printf("✅ Parsed request - Type: %s, Action: %s\n", req.Type, req.Action)
			switch {

			case req.Type == "DISCOVER_MANGAHUB":
				replyIP := utils.GetReplyIP(clientAddr)

				resp := DiscoverResponse{
					Type: "MANGAHUB_OFFER",
					Name: "mangahub-udp",
					Host: replyIP.String(),
					Port: port,
				}
				respBytes, _ := json.Marshal(resp)
				conn.WriteToUDP(respBytes, clientAddr)

				fmt.Printf("🔍 Discovery response sent to %s\n", clientAddr)
			case req.Type == "MANGAHUB_REQUEST":
				fmt.Printf("📡 Processing MANGAHUB_REQUEST - Action: %s from %s\n", req.Action, clientAddr)
				resp := h.ProcessUDPRequest(
					req.Action,
					req.Token,
					clientAddr.String(),
					req.Payload,
				)

				respBytes, err := json.Marshal(resp)
				if err != nil {
					fmt.Println("❌ Error marshaling response:", err)
				}
				n, err := conn.WriteToUDP(respBytes, clientAddr)
				if err != nil {
					fmt.Printf("❌ Error sending response to %s: %v\n", clientAddr, err)
				} else {
					fmt.Printf("✅ Sent %d bytes response to %s: %s\n", n, clientAddr, string(respBytes))
				}
			default:
				fmt.Printf("⚠️  Unknown UDP request type from %s: %s\n", clientAddr.String(), req.Type)
			}
		} else {
			fmt.Printf("❌ Error unmarshaling UDP request from %s: %v\n", clientAddr.String(), err)
			fmt.Printf("   Raw data: %s\n", string(raw))
		}
	}

}

func StartUDPServer(h *udp.UDPHandler) error {
	go func() {
		if err := StartUDPListener(9091, h); err != nil {
			fmt.Println("UDP listener error:", err)
		}
	}()
	return nil
}
