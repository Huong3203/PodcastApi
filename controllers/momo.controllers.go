package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sony/sonyflake"
	"gorm.io/gorm"
)

const (
	momoPartnerCode = services.MomoPartnerCode
	momoAccessKey   = services.MomoAccessKey
	momoSecretKey   = services.MomoSecretKey
)

// CreateMomoVIPPayment: tạo payment request
func CreateMomoVIPPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID       string `json:"user_id"`
			Amount       int    `json:"amount"`
			OrderInfo    string `json:"orderInfo"`
			RedirectUrl  string `json:"redirectUrl"`
			IpnUrl       string `json:"ipnUrl"`
			AutoRenew    bool   `json:"auto_renew"`
			PeriodMonths int    `json:"period_months"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.UserID == "" || req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		if req.PeriodMonths <= 0 {
			req.PeriodMonths = 1
		}

		log.Printf("📝 Creating payment - UserID: %s, Amount: %d, Period: %d months",
			req.UserID, req.Amount, req.PeriodMonths)

		// Verify user exists
		var user models.NguoiDung
		if err := db.First(&user, "id = ?", req.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Generate orderId/requestId
		flake := sonyflake.NewSonyflake(sonyflake.Settings{})
		if flake == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sonyflake init failed"})
			return
		}
		orderNum, _ := flake.NextID()
		requestNum, _ := flake.NextID()
		orderId := strconv.FormatUint(orderNum, 10)
		requestId := strconv.FormatUint(requestNum, 10)

		log.Printf("🆔 Generated OrderID: %s, RequestID: %s", orderId, requestId)

		requestType := "captureWallet"
		extraData := ""

		// Build signature
		fields := map[string]string{
			"accessKey":   momoAccessKey,
			"amount":      strconv.Itoa(req.Amount),
			"extraData":   extraData,
			"ipnUrl":      req.IpnUrl,
			"orderId":     orderId,
			"orderInfo":   req.OrderInfo,
			"partnerCode": momoPartnerCode,
			"redirectUrl": req.RedirectUrl,
			"requestId":   requestId,
			"requestType": requestType,
		}
		raw := services.BuildRawSignature(fields)
		sign := services.SignHmacSHA256(raw)

		payload := map[string]interface{}{
			"partnerCode": momoPartnerCode,
			"accessKey":   momoAccessKey,
			"requestId":   requestId,
			"amount":      req.Amount,
			"orderId":     orderId,
			"orderInfo":   req.OrderInfo,
			"redirectUrl": req.RedirectUrl,
			"ipnUrl":      req.IpnUrl,
			"extraData":   extraData,
			"requestType": requestType,
			"lang":        "vi",
			"signature":   sign,
		}

		momoRes, err := services.CreateMoMoRequest(payload)
		if err != nil {
			log.Println("❌ MoMo create error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "momo create error"})
			return
		}

		// Lưu pending payment vào DB
		pendingPayment := models.Payment{
			ID:           uuid.NewString(),
			OrderID:      orderId,
			UserID:       req.UserID,
			Amount:       req.Amount,
			Status:       "pending",
			IsRecurring:  req.AutoRenew,
			PeriodMonths: req.PeriodMonths,
		}
		if err := db.Create(&pendingPayment).Error; err != nil {
			log.Println("❌ DB create pending payment error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db create error"})
			return
		}

		log.Printf("✅ Payment created successfully - OrderID: %s", orderId)

		momoRes["orderId"] = orderId
		momoRes["requestId"] = requestId
		c.JSON(http.StatusOK, momoRes)
	}
}

// MomoVIPIPN: xử lý IPN callback từ MoMo
func MomoVIPIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("🔔 ========== IPN CALLBACK RECEIVED ==========")
		log.Printf("🕐 Time: %s", time.Now().Format("2006-01-02 15:04:05"))

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Println("❌ Cannot read IPN body:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
			return
		}

		log.Printf("📦 IPN Raw Body: %s", string(bodyBytes))
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var ipn map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &ipn); err != nil {
			log.Println("❌ IPN decode error:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ipn body"})
			return
		}

		log.Printf("📊 IPN Parsed - OrderID: %v, ResultCode: %v, Amount: %v, TransID: %v",
			ipn["orderId"], ipn["resultCode"], ipn["amount"], ipn["transId"])

		signature, _ := ipn["signature"].(string)

		// Verify signature
		raw := fmt.Sprintf(
			"accessKey=%s&amount=%v&extraData=%v&orderId=%v&orderInfo=%v&partnerCode=%s&requestId=%v&responseTime=%v&resultCode=%v&transId=%v",
			services.MomoAccessKey,
			ipn["amount"],
			ipn["extraData"],
			ipn["orderId"],
			ipn["orderInfo"],
			services.MomoPartnerCode,
			ipn["requestId"],
			ipn["responseTime"],
			ipn["resultCode"],
			ipn["transId"],
		)

		ok := services.VerifySignature(raw, signature)
		if !ok {
			log.Println("❌ IPN signature verification FAILED")
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature"})
			return
		}
		log.Println("✅ IPN signature verified")

		orderID, _ := ipn["orderId"].(string)
		resultCodeF := ipn["resultCode"]
		var resultCode int
		switch v := resultCodeF.(type) {
		case float64:
			resultCode = int(v)
		case int:
			resultCode = v
		default:
			resultCode = -1
		}

		// Tìm payment
		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderID).Error; err != nil {
			log.Printf("❌ Payment not found in IPN - OrderID: %s", orderID)
			c.Status(http.StatusNoContent)
			return
		}

		log.Printf("📌 Current payment status: %s", payment.Status)

		if resultCode == 0 {
			// THÀNH CÔNG
			log.Printf("✅ Payment SUCCESS - OrderID: %s", orderID)
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("❌ Update payment error:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "update payment failed"})
				return
			}

			// Set VIP
			if err := setUserVIPByPayment(db, &payment); err != nil {
				log.Println("❌ Set VIP error:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "set vip failed"})
				return
			}

			log.Printf("🎉 Payment completed - UserID: %s, Amount: %d", payment.UserID, payment.Amount)
		} else {
			// THẤT BẠI
			log.Printf("❌ Payment FAILED - OrderID: %s, ResultCode: %d", orderID, resultCode)
			payment.Status = "failed"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("❌ Update payment failed error:", err)
			}
		}

		log.Println("🔔 ========== IPN CALLBACK END ==========")
		c.Status(http.StatusOK)
	}
}

// MomoVIPReturn: xử lý redirect từ MoMo
func MomoVIPReturn(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Query("orderId")
		resultCodeStr := c.Query("resultCode")
		message := c.Query("message")

		log.Printf("🔄 MoMo RETURN - OrderID: %s, ResultCode: %s, Message: %s",
			orderId, resultCodeStr, message)

		if orderId == "" {
			c.String(http.StatusBadRequest, "Missing orderId")
			return
		}

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderId).Error; err != nil {
			log.Printf("❌ Payment not found in RETURN - OrderID: %s", orderId)
			redirect := fmt.Sprintf("sonifyapp://payment-result?orderId=%s&resultCode=%s&status=not_found",
				orderId, resultCodeStr)
			c.Redirect(http.StatusFound, redirect)
			return
		}

		if resultCodeStr == "0" {
			// THÀNH CÔNG - Update status
			log.Printf("✅ RETURN SUCCESS - OrderID: %s", orderId)
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("❌ Update payment error:", err)
			}

			// Set VIP
			if err := setUserVIPByPayment(db, &payment); err != nil {
				log.Println("❌ Set VIP error:", err)
			}

			redirect := fmt.Sprintf("sonifyapp://payment-result?orderId=%s&resultCode=%s",
				orderId, resultCodeStr)
			c.Redirect(http.StatusFound, redirect)
			return
		}

		// THẤT BẠI
		log.Printf("❌ RETURN FAILED - OrderID: %s", orderId)
		payment.Status = "failed"
		_ = db.Save(&payment)

		redirect := fmt.Sprintf("sonifyapp://payment-result?orderId=%s&resultCode=%s&message=%s",
			orderId, resultCodeStr, message)
		c.Redirect(http.StatusFound, redirect)
	}
}

// CheckPaymentStatus: Check payment status
func CheckPaymentStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("orderId")
		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
			return
		}

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}

		log.Printf("📊 Check status - OrderID: %s, Status: %s, Created: %s",
			orderID, payment.Status, payment.CreatedAt.Format("2006-01-02 15:04:05"))

		c.JSON(http.StatusOK, gin.H{
			"order_id":      payment.OrderID,
			"user_id":       payment.UserID,
			"amount":        payment.Amount,
			"status":        payment.Status,
			"period_months": payment.PeriodMonths,
			"is_recurring":  payment.IsRecurring,
			"created_at":    payment.CreatedAt,
			"updated_at":    payment.UpdatedAt,
		})
	}
}

// VerifyPaymentAndSetVIP: Verify và set VIP
func VerifyPaymentAndSetVIP(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("orderId")
		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
			return
		}

		log.Printf("🔍 Verify payment - OrderID: %s", orderID)

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}

		// ✅ Check if payment is pending for too long (>1 minute in sandbox)
		if payment.Status == "pending" {
			elapsed := time.Since(payment.CreatedAt)
			log.Printf("⏰ Payment pending for %.0f seconds", elapsed.Seconds())

			// Trong sandbox, sau 60s vẫn pending thì có thể do IPN chưa được gọi
			if elapsed > 60*time.Second {
				log.Printf("⚠️ Payment stuck in pending state - consider manual verification")
			}
		}

		if payment.Status == "success" {
			log.Printf("✅ Payment already success - OrderID: %s", orderID)

			var user models.NguoiDung
			if err := db.First(&user, "id = ?", payment.UserID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}

			// Kiểm tra user đã VIP chưa
			now := time.Now()
			needUpdate := false

			if !user.VIP {
				needUpdate = true
				log.Printf("👤 User %s is not VIP yet", user.ID)
			} else if user.VIPExpires == nil || user.VIPExpires.Before(now) {
				needUpdate = true
				log.Printf("👤 User %s VIP expired", user.ID)
			}

			if needUpdate {
				if err := setUserVIPByPayment(db, &payment); err != nil {
					log.Println("❌ Set VIP error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set VIP"})
					return
				}
				log.Printf("✅ VIP set for user %s", user.ID)
			}

			// Fetch updated user
			if err := db.First(&user, "id = ?", payment.UserID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":     "payment verified",
				"status":      "success",
				"user_id":     user.ID,
				"vip":         user.VIP,
				"vip_expires": user.VIPExpires,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "payment not completed",
			"status":  payment.Status,
		})
	}
}

// ✅ NEW: ForceCompletePayment - Manual complete payment (FOR TESTING ONLY)
func ForceCompletePayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("orderId")
		if orderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
			return
		}

		log.Printf("🔧 [MANUAL] Force completing payment - OrderID: %s", orderID)

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}

		log.Printf("📌 Current status: %s", payment.Status)

		// Force update to success
		payment.Status = "success"
		if err := db.Save(&payment).Error; err != nil {
			log.Println("❌ Update payment error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
			return
		}

		// Set VIP
		if err := setUserVIPByPayment(db, &payment); err != nil {
			log.Println("❌ Set VIP error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "set VIP failed"})
			return
		}

		// Fetch updated user
		var user models.NguoiDung
		if err := db.First(&user, "id = ?", payment.UserID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
			return
		}

		log.Printf("✅ [MANUAL] Payment force completed - OrderID: %s, UserID: %s", orderID, payment.UserID)

		c.JSON(http.StatusOK, gin.H{
			"message":       "Payment manually completed",
			"status":        "success",
			"order_id":      payment.OrderID,
			"user_id":       payment.UserID,
			"vip":           user.VIP,
			"vip_expires":   user.VIPExpires,
			"period_months": payment.PeriodMonths,
		})
	}
}

// setUserVIPByPayment: Set VIP cho user
func setUserVIPByPayment(db *gorm.DB, p *models.Payment) error {
	var user models.NguoiDung
	if err := db.First(&user, "id = ?", p.UserID).Error; err != nil {
		return err
	}

	now := time.Now()
	var newExpiry time.Time

	// Gia hạn từ expires nếu còn VIP, nếu không thì từ bây giờ
	if user.VIPExpires != nil && user.VIPExpires.After(now) {
		newExpiry = user.VIPExpires.AddDate(0, p.PeriodMonths, 0)
		log.Printf("📅 Extending VIP from %s to %s",
			user.VIPExpires.Format("2006-01-02"), newExpiry.Format("2006-01-02"))
	} else {
		newExpiry = now.AddDate(0, p.PeriodMonths, 0)
		log.Printf("📅 New VIP from now to %s", newExpiry.Format("2006-01-02"))
	}

	// Update user
	if err := db.Model(&models.NguoiDung{}).Where("id = ?", p.UserID).Updates(map[string]interface{}{
		"vip":         true,
		"vip_expires": newExpiry,
		"auto_renew":  p.IsRecurring,
	}).Error; err != nil {
		return err
	}

	log.Printf("🎉 User %s is now VIP until %s", p.UserID, newExpiry.Format("2006-01-02 15:04:05"))
	return nil
}
