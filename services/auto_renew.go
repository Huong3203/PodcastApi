package services

// import (
// 	"fmt"
// 	"log"
// 	"time"

// 	"github.com/Huong3203/APIPodcast/models"
// 	"gorm.io/gorm"
// )

// // StartAutoRenewScheduler chạy goroutine kiểm tra và tạo payment tự động (thực tế: tạo invoice, gửi notify FE)
// func StartAutoRenewScheduler(db *gorm.DB, checkInterval time.Duration, thresholdDays int) {
// 	go func() {
// 		ticker := time.NewTicker(checkInterval)
// 		defer ticker.Stop()
// 		for {
// 			select {
// 			case <-ticker.C:
// 				processAutoRenew(db, thresholdDays)
// 			}
// 		}
// 	}()
// }

// func processAutoRenew(db *gorm.DB, thresholdDays int) {
// 	// Tìm users có auto_renew = true và vip_expires within thresholdDays
// 	now := time.Now()
// 	threshold := now.Add(time.Duration(thresholdDays) * 24 * time.Hour)

// 	var users []models.NguoiDung
// 	if err := db.Where("auto_renew = ? AND vip_expires IS NOT NULL AND vip_expires <= ?", true, threshold).Find(&users).Error; err != nil {
// 		log.Println("auto renew query err:", err)
// 		return
// 	}

// 	for _, u := range users {
// 		// xác định gói: lấy last successful recurring payment period_months nếu có
// 		var lastPayment models.Payment
// 		err := db.Where("user_id = ? AND status = ? AND is_recurring = ?", u.ID, "success", true).
// 			Order("created_at desc").
// 			First(&lastPayment).Error
// 		periodMonths := 1
// 		amount := 99000 // default; bạn có thể lấy price từ config
// 		isRecurring := true
// 		if err == nil {
// 			periodMonths = lastPayment.PeriodMonths
// 			amount = lastPayment.Amount
// 		}

// 		// Tạo order mới (pending) và gửi notify / tạo payUrl -> FE sẽ mở để thanh toán
// 		orderId := fmt.Sprintf("auto-%d", time.Now().UnixNano())
// 		p := models.Payment{
// 			ID:           GenerateUUID(),
// 			OrderID:      orderId,
// 			UserID:       u.ID,
// 			Amount:       amount,
// 			Status:       "pending",
// 			IsRecurring:  isRecurring,
// 			PeriodMonths: periodMonths,
// 		}
// 		if err := db.Create(&p).Error; err != nil {
// 			log.Println("create auto renewal payment err:", err)
// 			continue
// 		}

// 		// TODO: gửi noti cho user hoặc gọi MoMo create payment để sinh payUrl tự động
// 		// Option A: Tự động call MoMo create để sinh payUrl và gửi email/notification
// 		// Option B: Gửi notification đến FE -> FE hiển thị payUrl/QR cho user

// 		// Ở đây mình log để bạn triển khai notify
// 		log.Printf("Auto-renew invoice created for user %s order %s amount %d\n", u.ID, orderId, amount)
// 	}
// }
