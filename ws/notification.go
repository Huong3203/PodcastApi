package ws

import (
	"log"
	"net/http"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/gorilla/websocket"
)

var notificationClients = make(map[*websocket.Conn]string) // map[conn]userID
var notificationBroadcast = make(chan models.Notification)

var notificationUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ⚡ Kết nối WS thông báo
func HandleNotificationWS(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	conn, err := notificationUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("❌ Notification WS upgrade error:", err)
		return
	}
	defer conn.Close()

	notificationClients[conn] = userID
	log.Printf("✅ Notification WS connected: %s", userID)

	for {
		var msg models.Notification
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("⚠️ Notification WS read error:", err)
			delete(notificationClients, conn)
			break
		}
	}
}

// ⚡ Luồng xử lý gửi thông báo
func HandleNotificationMessages() {
	for {
		noti := <-notificationBroadcast
		for conn, userID := range notificationClients {
			if userID == noti.UserID {
				err := conn.WriteJSON(noti)
				if err != nil {
					log.Println("⚠️ Notification WS send error:", err)
					conn.Close()
					delete(notificationClients, conn)
				}
			}
		}
	}
}

// ⚡ Hàm public cho controller gọi
func SendNotification(noti models.Notification) {
	notificationBroadcast <- noti
}
