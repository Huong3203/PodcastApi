package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Map client theo userID để gửi badge realtime
var badgeClients = make(map[*websocket.Conn]string)

// Channel gửi số badge
var badgeBroadcast = make(chan BadgeMessage, 50)

type BadgeMessage struct {
	UserID string `json:"user_id"`
	Count  int64  `json:"count"`
}

var badgeUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WS dùng để cập nhật badge realtime
func HandleBadgeWS(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	conn, err := badgeUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("❌ Badge WS upgrade error:", err)
		return
	}
	defer conn.Close()

	// Gắn connection vào map
	badgeClients[conn] = userID

	log.Printf("🎉 Badge WS connected: %s\n", userID)

	for {
		// WS badge không cần nhận dữ liệu — chỉ nhận ping để giữ kết nối
		var tmp interface{}
		if err := conn.ReadJSON(&tmp); err != nil {
			log.Println("⚠️ Badge WS disconnected:", err)
			delete(badgeClients, conn)
			return
		}
	}
}

// Goroutine gửi badge realtime
func HandleBadgeMessages() {
	for {
		msg := <-badgeBroadcast

		for conn, uid := range badgeClients {
			if uid == msg.UserID {
				err := conn.WriteJSON(msg)
				if err != nil {
					log.Println("⚠️ Badge WS send error:", err)
					conn.Close()
					delete(badgeClients, conn)
				}
			}
		}
	}
}

// Hàm để controller gọi
func SendBadgeUpdate(userID string, count int64) {
	badgeBroadcast <- BadgeMessage{
		UserID: userID,
		Count:  count,
	}
}
