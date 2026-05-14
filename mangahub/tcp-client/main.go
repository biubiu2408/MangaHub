package tcpclient

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type ProgressUpdateMessage struct {
	Type          string    `json:"type"`
	MangaID       string    `json:"manga_id"`
	Previous      int       `json:"previous_chapter"`
	Current       int       `json:"current_chapter"`
	UpdatedAt     time.Time `json:"updated_at"`
	DevicesSynced int       `json:"devices_synced"`
}

func StartSync(token, deviceID, serverIP string) error {
	conn, err := net.Dial("tcp", serverIP+":9090")
	if err != nil {
		return err
	}

	// handshake
	json.NewEncoder(conn).Encode(map[string]string{
		"token":     token,
		"device_id": deviceID,
	})

	// listen for updates
	for {
		var msg ProgressUpdateMessage
		if err := json.NewDecoder(conn).Decode(&msg); err != nil {
			return err
		}

		fmt.Printf(
			"SYNC UPDATE: %s → chapter %d\n Device online: %d\n",
			msg.MangaID,
			msg.Current,
			msg.DevicesSynced,
		)
	}

}
