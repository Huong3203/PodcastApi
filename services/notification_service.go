package services

import (
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/ws"
	"github.com/google/uuid"
)

// Tạo + lưu thông báo + gửi realtime
func CreateNotification(userID, podcastID, action, message string) error {
	noti := models.Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		PodcastID: podcastID,
		Action:    action,
		Message:   message,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	// Lưu DB
	if err := config.DB.Create(&noti).Error; err != nil {
		return err
	}

	// Gửi realtime đến frontend (qua WebSocket)
	ws.SendNotification(noti)
	return nil
}
