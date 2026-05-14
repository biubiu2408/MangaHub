package udp_client

import (
	"encoding/json"
	"fmt"
	"net"
	"syscall"
	"time"
)

type Notification struct {
	MangaID   string
	Chapter   int64
	Timestamp time.Time
}
type DiscoverResponse struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

func DiscoverUDPServer(timeout time.Duration) (string, error) {
	discoveryMsg := map[string]string{
		"type":    "DISCOVER_MANGAHUB",
		"action":  "",
		"token":   "",
		"payload": "",
	}
	body, _ := json.Marshal(discoveryMsg)

	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:9091")
	if err != nil {
		return "", err
	}

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return "", err
	}

	rawConn.Control(func(fd uintptr) {
		err := syscall.SetsockoptInt(
			syscall.Handle(fd),
			syscall.SOL_SOCKET,
			syscall.SO_BROADCAST,
			1,
		)
		if err != nil {
			fmt.Println("Failed to enable broadcast:", err)
		}
	})
	// Enable broadcast
	conn.SetWriteBuffer(1024)

	_, err = conn.WriteToUDP(body, broadcastAddr)
	if err != nil {
		return "", err
	}

	fmt.Println("🔍 Sent UDP discovery broadcast")

	buffer := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(timeout))

	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return "", fmt.Errorf("no UDP server discovered")
	}

	var resp DiscoverResponse
	if err := json.Unmarshal(buffer[:n], &resp); err != nil {
		return "", err
	}

	if resp.Type != "MANGAHUB_OFFER" {
		return "", fmt.Errorf("invalid discovery response from %s", addr)
	}

	serverAddr := fmt.Sprintf("%s:%d", resp.Host, resp.Port)
	fmt.Printf("✅ Discovered UDP server: %s (%s)\n", resp.Name, serverAddr)

	return serverAddr, nil
}

type UDPResponse struct {
	Status  string `json:"status"`
	Payload string `json:"payload"`
}

func RegisterUDPNotification(serverAddr string, jwt string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return err
	}

	data := map[string]string{
		"type":    "MANGAHUB_REQUEST",
		"action":  "register",
		"token":   jwt,
		"payload": "",
	}

	body, _ := json.Marshal(data)

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Println("📡 Sending UDP registration request...")

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	_, err = conn.Write(body)
	if err != nil {
		return err
	}
	fmt.Println("Sent UDP request! Waiting...")

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("no response from UDP server")
	}

	var resp UDPResponse
	if err := json.Unmarshal(buffer[:n], &resp); err != nil {
		return err
	}

	if resp.Status != "success" {
		return fmt.Errorf("registration failed: %s", resp.Payload)
	}

	fmt.Println("✅ UDP server registered for notifications")
	return nil
}
func SubscribeMangaUDP(serverAddr string, jwt string, mangaID string) error {
	serverAddress, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return fmt.Errorf("error resolving address: %v", err)
	}
	if mangaID == "" {
		return fmt.Errorf("--manga required")
	}
	// payload := map[string]string{
	// 	"manga_id": mangaID,
	// }
	data := map[string]string{
		"type":    "MANGAHUB_REQUEST",
		"action":  "subscribe",
		"token":   jwt,
		"payload": mangaID,
	}
	body, _ := json.Marshal(data)
	conn, err := net.DialUDP("udp", nil, serverAddress)
	if err != nil {
		return fmt.Errorf("error connecting: %v", err)
	}
	defer conn.Close()

	conn.Write([]byte(body))
	if err != nil {
		return fmt.Errorf("error sending subscribe Message: %v", err)
	}

	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	if err != nil {
		return err
	}
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error receiving:", err)
		return fmt.Errorf("error receiving subscribe response: %v", err)
	}
	raw := buffer[:n]

	var resp UDPResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("invalid JSON response: %s", string(raw))
	}
	if resp.Status != "success" {
		return fmt.Errorf("subscription failed: %s", resp.Payload)
	}
	fmt.Println("✅ Subscribed to manga notifications successfully.")
	return nil
}
func StartUDPListener(port int, serverID string) error {

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

	fmt.Printf("UDP Listener (%s) running on port %d...\n", serverID, port)

	buffer := make([]byte, 2048)

	for {
		n, serverAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Read error:", err)
			continue // do NOT exit — keep listening
		}

		raw := buffer[:n]

		// Try parsing JSON
		var resp Notification
		if err := json.Unmarshal(raw, &resp); err == nil {
			fmt.Printf("[JSON RECEIVED from %s]\nManga: %s | Chapter: %d | Timestamp: %s\n",
				serverAddr, resp.MangaID, resp.Chapter, resp.Timestamp,
			)
		} else {
			fmt.Printf("[RAW RECEIVED from %s] %s\n", serverAddr, string(raw))
		}

		// Optional: reply PONG if you ever need RTT
		// conn.WriteToUDP([]byte("PONG"), serverAddr)
	}

}

func StartUDPServer(username string) error {
	go func() {
		if err := StartUDPListener(3002, username); err != nil {
			fmt.Println("UDP listener error:", err)
		}
	}()
	return nil
}
