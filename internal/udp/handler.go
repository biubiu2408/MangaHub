package udp

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/biubiu2408/MangaHub/utils"
)

type Notification struct {
	MangaID   string
	Chapter   int64
	Timestamp time.Time
}
type UDPClientRequest struct {
	Type    string `json:"type"`
	Action  string `json:"action"`
	Token   string `json:"token"`
	Payload string `json:"payload"`
}
type UDPSubscribeRequest struct {
	MangaID string `json:"manga_id"`
}

type UDPResponse struct {
	Status  string `json:"status"`
	Payload string `json:"payload"`
}
type UDPHandler struct {
	repo *UDPRepository
}

func NewUDPHandler(repo *UDPRepository) *UDPHandler {
	return &UDPHandler{repo: repo}
}

func (h *UDPHandler) ProcessUDPRequest(action string, token string, clientAddr string, payload string) UDPResponse {
	var UDPRes UDPResponse
	//join user address and port 3002
	clientAddrParts := strings.Split(clientAddr, ":")
	clientAddr = fmt.Sprintf("%s:%d", clientAddrParts[0], 3002)
	if token == "" {
		UDPRes.Status = "error"
		UDPRes.Payload = "no token provided"
		return UDPRes
	}
	claims, err := utils.ValidateToken(token)
	if err != nil {
		UDPRes.Status = "error"
		UDPRes.Payload = fmt.Sprintf("invalid token: %v", err)
		return UDPRes
	}
	if claims.ExpiresAt.Time.Before(time.Now()) {
		UDPRes.Status = "error"
		UDPRes.Payload = "token expired, please login again"
		return UDPRes
	}
	userID := claims.UserId
	switch action {
	case "register":
		err := h.repo.CreateNotificationEntry(userID, clientAddr)
		if err != nil {
			UDPRes.Status = "error"
			UDPRes.Payload = fmt.Sprintf("failed to register UDP address: %v", err)
			return UDPRes
		}
		UDPRes.Status = "success"
		UDPRes.Payload = "UDP address registered successfully"
	case "subscribe":
		var storedAddr string
		storedAddr, _ = h.repo.GetUserUDPAddress(userID)
		if err != nil || storedAddr == "" {
			UDPRes.Status = "error"
			UDPRes.Payload = fmt.Sprintf("failed to get UDP address: please register your UDP address first by running mangahub notify subscribe. Error: %v", err)
			return UDPRes
		}

		if storedAddr != clientAddr {
			err = h.repo.UpdateNotificationEntry(userID, clientAddr)
			if err != nil {
				UDPRes.Status = "error"
				UDPRes.Payload = fmt.Sprintf("error in checking stored and current address: %v", err)
				return UDPRes
			}
		}
		isSubscriptionExist, err := h.repo.SubscriptionExists(userID, payload)
		if err != nil {
			UDPRes.Status = "error"
			UDPRes.Payload = fmt.Sprintf("failed to check subscription: %v", err)
			return UDPRes
		}
		if isSubscriptionExist {
			UDPRes.Status = "error"
			UDPRes.Payload = "subscription already exists for this manga"
			return UDPRes
		}
		err = h.repo.CreateSubscriptionEntry(userID, payload)
		if err != nil {
			UDPRes.Status = "error"
			UDPRes.Payload = fmt.Sprintf("failed to create subscription: %v", err)
			return UDPRes
		}
		UDPRes.Status = "success"
		UDPRes.Payload = "subscribed to manga successfully"
	default:
		UDPRes.Status = "error"
		UDPRes.Payload = "unknown action"
	}
	return UDPRes
}

func (h *UDPHandler) NotifyNewChapter(mangaID string, chapter int64) (int64, error) {
	subscribers, err := h.repo.GetMangaSubscribers(mangaID)
	if err != nil {
		return 0, err
	}
	successCount := int64(0)

	// for _, userID := range subscribers {
	// 	clientAddr, err := h.repo.GetUserUDPAddress(userID)
	// 	if err != nil {
	// 		// log and continue
	// 		fmt.Printf("[UDP] no address for user %d: %v\n", userID, err)
	// 		continue
	// 	}

	// 	if clientAddr == "" {
	// 		continue
	// 	}

	// 	err = SendNewChapterNotification(
	// 		clientAddr,
	// 		mangaID,
	// 		chapter,
	// 		60*time.Second,
	// 	)
	// 	if err != nil {
	// 		fmt.Printf("[UDP] send failed to %s: %v\n", clientAddr, err)
	// 		continue
	// 	}
	// 	successCount++
	// }
	for _, userID := range subscribers {
		go func(uid int64) {
			clientAddr, err := h.repo.GetUserUDPAddress(uid)
			if err != nil || clientAddr == "" {
				return
			}

			_ = SendNewChapterNotification(
				clientAddr,
				mangaID,
				chapter,
				60*time.Second,
			)
		}(userID)
	}
	return successCount, nil
}

// helper function to send UDP message
// func SendUDPResponse(clientUDPAddr string, response UDPResponse) error {
// 	var message UDPResponse

// 	conn, err := net.DialUDP("udp", nil, client_udp_addr)
// 	if err != nil {
// 		fmt.Println("Error connecting:", err)
// 		return fmt.Errorf("error connecting: %v", err)
// 	}
// 	defer conn.Close()
// 	// Measure time
// 	start := time.Now()
// 	data, err := json.Marshal(message)
// 	if err != nil {
// 		return err
// 	}
// 	// Send to client
// 	_, err = conn.Write([]byte(data))
// 	if err != nil {
// 		fmt.Println("Error sending UDP Message", err)
// 		return fmt.Errorf("error sending UDP Message: %v", err)
// 	}
// 	totalRTT := time.Since(start)
// 	//Return result
// 	fmt.Printf("Sent subscription success to %s in %v\n", clientUDPAddr, totalRTT)
// 	return nil

// }
func SendNewChapterNotification(clientUDPAddr string, manga_id string, chapter int64, timeout time.Duration) error {
	var message Notification
	serverAddress, err := net.ResolveUDPAddr("udp", clientUDPAddr)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return fmt.Errorf("error resolving address: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, serverAddress)
	if err != nil {
		fmt.Println("Error connecting:", err)
		return fmt.Errorf("error connecting: %v", err)
	}
	defer conn.Close()
	// Measure time
	start := time.Now()
	message = Notification{MangaID: manga_id, Chapter: chapter, Timestamp: time.Now()}
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	// Send to server
	_, err = conn.Write([]byte(data))
	if err != nil {
		fmt.Println("Error sending UDP Message", err)
		return fmt.Errorf("error sending UDP Message: %v", err)
	}
	totalRTT := time.Since(start)
	//Return result
	fmt.Printf("Sent new chapter notification to %s in %v\n", clientUDPAddr, totalRTT)
	return nil

}
