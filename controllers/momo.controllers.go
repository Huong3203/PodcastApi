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

// CreateMomoVIPPayment: tạo payment request (chưa lưu DB)
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
			log.Println("momo create error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "momo create error"})
			return
		}

		// Lưu thông tin pending vào cache/session hoặc temporary table
		// để khi IPN callback có thể lấy thông tin UserID, PeriodMonths, AutoRenew
		pendingPayment := models.Payment{
			ID:           uuid.NewString(),
			OrderID:      orderId,
			UserID:       req.UserID,
			Amount:       req.Amount,
			Status:       "pending",
			IsRecurring:  req.AutoRenew,
			PeriodMonths: req.PeriodMonths,
		}
		// Lưu vào DB với status pending để tracking
		if err := db.Create(&pendingPayment).Error; err != nil {
			log.Println("db create pending payment error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db create error"})
			return
		}

		// Trả về response MoMo cho FE mở payUrl
		momoRes["orderId"] = orderId
		momoRes["requestId"] = requestId
		c.JSON(http.StatusOK, momoRes)
	}
}

// MomoVIPIPN: xử lý IPN từ MoMo - CHỈ LƯU KHI THÀNH CÔNG
func MomoVIPIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var ipn map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &ipn); err != nil {
			log.Println("ipn decode err:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ipn body"})
			return
		}

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
			log.Println("IPN signature verification failed")
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid signature"})
			return
		}

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

		// Tìm pending payment
		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderID).Error; err != nil {
			log.Println("payment not found in ipn:", orderID)
			c.Status(http.StatusNoContent)
			return
		}

		if resultCode == 0 {
			// THÀNH CÔNG: Cập nhật status và set VIP cho user
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("update payment err:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "update payment failed"})
				return
			}

			// Set VIP cho user
			if err := setUserVIPByPayment(db, &payment); err != nil {
				log.Println("set user vip err:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "set vip failed"})
				return
			}

			log.Printf("Payment success: OrderID=%s, UserID=%s, Amount=%d", orderID, payment.UserID, payment.Amount)
		} else {
			// THẤT BẠI: Cập nhật status hoặc xóa record
			payment.Status = "failed"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("update payment failed err:", err)
			}
			log.Printf("Payment failed: OrderID=%s, ResultCode=%d", orderID, resultCode)
		}

		c.Status(http.StatusOK)
	}
}

// MomoVIPReturn: xử lý redirect user từ MoMo
func MomoVIPReturn(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Query("orderId")
		resultCodeStr := c.Query("resultCode")
		message := c.Query("message")
		transId := c.Query("transId")
		signature := c.Query("signature")

		log.Println("MoMo RETURN:", "orderId=", orderId, "resultCode=", resultCodeStr, "message=", message)

		// Optional verify signature
		if signature != "" {
			raw := fmt.Sprintf("accessKey=%s&orderId=%s&partnerCode=%s&requestId=%s&amount=%s&transId=%s&resultCode=%s",
				services.MomoAccessKey,
				orderId,
				services.MomoPartnerCode,
				c.Query("requestId"),
				c.Query("amount"),
				transId,
				resultCodeStr,
			)
			if !services.VerifySignature(raw, signature) {
				log.Println("return signature invalid")
			}
		}

		if orderId == "" {
			c.String(http.StatusBadRequest, "Missing orderId")
			return
		}

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderId).Error; err != nil {
			log.Println("Payment not found in RETURN for orderId:", orderId)
			redirect := fmt.Sprintf("%s?orderId=%s&status=not_found", services.AppClientRedirectURL, orderId)
			c.Redirect(http.StatusFound, redirect)
			return
		}

		if resultCodeStr == "0" {
			// THÀNH CÔNG
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("update payment err:", err)
			}

			// Set VIP cho user
			if err := setUserVIPByPayment(db, &payment); err != nil {
				log.Println("set user vip err:", err)
			}

			redirect := fmt.Sprintf("%s?orderId=%s&status=success", services.AppClientRedirectURL, orderId)
			c.Redirect(http.StatusFound, redirect)
			return
		}

		// THẤT BẠI
		payment.Status = "failed"
		_ = db.Save(&payment)
		redirect := fmt.Sprintf("%s?orderId=%s&status=failed&message=%s", services.AppClientRedirectURL, orderId, message)
		c.Redirect(http.StatusFound, redirect)
	}
}

// CheckPaymentStatus: API để FE check trạng thái thanh toán
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

		// Trả về thông tin payment
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

// VerifyPaymentAndSetVIP: API để FE verify và force set VIP (nếu cần)
func VerifyPaymentAndSetVIP(db *gorm.DB) gin.HandlerFunc {
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

		// Nếu payment đã success nhưng user chưa được set VIP
		if payment.Status == "success" {
			var user models.NguoiDung
			if err := db.First(&user, "id = ?", payment.UserID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}

			// Kiểm tra xem user đã VIP chưa
			now := time.Now()
			needUpdate := false

			if !user.VIP {
				needUpdate = true
			} else if user.VIPExpires == nil || user.VIPExpires.Before(now) {
				needUpdate = true
			}

			// Nếu cần update, set VIP
			if needUpdate {
				if err := setUserVIPByPayment(db, &payment); err != nil {
					log.Println("set user vip err:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set VIP"})
					return
				}
			}

			// Lấy thông tin user mới nhất
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

		// Nếu payment chưa success
		c.JSON(http.StatusOK, gin.H{
			"message": "payment not completed",
			"status":  payment.Status,
		})
	}
}

// setUserVIPByPayment: Set VIP=true và VIPExpires cho user
func setUserVIPByPayment(db *gorm.DB, p *models.Payment) error {
	var user models.NguoiDung
	if err := db.First(&user, "id = ?", p.UserID).Error; err != nil {
		return err
	}

	now := time.Now()
	var newExpiry time.Time

	// Nếu user còn VIP (VIPExpires > now), gia hạn từ expires
	// Nếu không, tính từ bây giờ
	if user.VIPExpires != nil && user.VIPExpires.After(now) {
		newExpiry = user.VIPExpires.AddDate(0, p.PeriodMonths, 0)
	} else {
		newExpiry = now.AddDate(0, p.PeriodMonths, 0)
	}

	// Cập nhật VIP = true và VIPExpires
	if err := db.Model(&models.NguoiDung{}).Where("id = ?", p.UserID).Updates(map[string]interface{}{
		"vip":         true,
		"vip_expires": newExpiry,
		"auto_renew":  p.IsRecurring, // Cập nhật auto_renew setting
	}).Error; err != nil {
		return err
	}

	log.Printf("User %s is now VIP until %s", p.UserID, newExpiry.Format("2006-01-02 15:04:05"))
	return nil
}
