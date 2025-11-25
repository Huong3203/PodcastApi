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
	// copy lại từ services hoặc import config chung
	momoPartnerCode = services.MomoPartnerCode
	momoAccessKey   = services.MomoAccessKey
	momoSecretKey   = services.MomoSecretKey
)

// CreateMomoVIPPayment: tạo payment (FE gọi)
func CreateMomoVIPPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID       string `json:"user_id"`
			Amount       int    `json:"amount"`
			OrderInfo    string `json:"orderInfo"`
			RedirectUrl  string `json:"redirectUrl"`
			IpnUrl       string `json:"ipnUrl"`
			AutoRenew    bool   `json:"auto_renew"`    // nếu FE gửi muốn auto renew
			PeriodMonths int    `json:"period_months"` // số tháng gói (mặc định 1)
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

		// generate orderId/requestId
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
		extraData := "" // nếu cần

		// build raw signature và sign
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

		// Lưu payment
		payment := models.Payment{
			ID:           uuid.NewString(),
			OrderID:      orderId,
			UserID:       req.UserID,
			Amount:       req.Amount,
			Status:       "pending",
			IsRecurring:  req.AutoRenew,
			PeriodMonths: req.PeriodMonths,
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Println("db create payment error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db create error"})
			return
		}

		// Nếu FE muốn lưu setting auto-renew trên user
		if req.AutoRenew {
			if err := db.Model(&models.NguoiDung{}).Where("id = ?", req.UserID).Update("auto_renew", true).Error; err != nil {
				log.Println("update user auto_renew err:", err)
			}
		}

		// trả về response MoMo cho FE mở payUrl
		momoRes["orderId"] = orderId
		momoRes["requestId"] = requestId
		c.JSON(http.StatusOK, momoRes)
	}
}

// MomoVIPIPN: xử lý IPN từ MoMo
func MomoVIPIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// đọc body thô để có thể verify signature
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
			return
		}
		// restore body để dùng gin.Bind nếu cần
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var ipn map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &ipn); err != nil {
			log.Println("ipn decode err:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ipn body"})
			return
		}

		// lấy signature từ body
		signature, _ := ipn["signature"].(string)

		// Build raw string theo MoMo spec: có thể khác nhau tuỳ payload của MoMo IPN
		// Common các field: partnerCode, orderId, requestId, amount, responseTime, resultCode, message, transId, extraData
		// Hãy đảm bảo thứ tự giống MoMo gửi — dưới đây là 1 ví dụ, bạn cần sửa nếu MoMo spec khác
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
			// trả 400 cho MoMo hoặc 403; MoMo có thể retry — tùy policy
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

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderID).Error; err != nil {
			log.Println("payment not found ipn:", orderID)
			c.Status(http.StatusNoContent)
			return
		}

		if resultCode == 0 {
			// success
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("update payment err:", err)
			}
			// set VIP và VIPExpires theo PeriodMonths
			if err := setUserVIPByPayment(db, &payment); err != nil {
				log.Println("set user vip err:", err)
			}
		} else {
			payment.Status = "failed"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("update payment failed err:", err)
			}
		}

		// MoMo thường yêu cầu trả 200 OK hoặc empty 204
		c.Status(http.StatusOK)
	}
}

// MomoVIPReturn: xử lý redirect user từ MoMo (FE sẽ nhận query params)
func MomoVIPReturn(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Query("orderId")
		resultCodeStr := c.Query("resultCode")
		message := c.Query("message")
		transId := c.Query("transId")
		signature := c.Query("signature") // có thể kèm theo

		log.Println("MoMo RETURN:", "orderId=", orderId, "resultCode=", resultCodeStr, "message=", message)

		// optional verify query signature nếu MoMo gửi
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
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("update payment err:", err)
			}
			if err := setUserVIPByPayment(db, &payment); err != nil {
				log.Println("set user vip err:", err)
			}
			redirect := fmt.Sprintf("%s?orderId=%s&status=success", services.AppClientRedirectURL, orderId)
			c.Redirect(http.StatusFound, redirect)
			return
		}

		// failed
		payment.Status = "failed"
		_ = db.Save(&payment)
		redirect := fmt.Sprintf("%s?orderId=%s&status=failed&message=%s", services.AppClientRedirectURL, orderId, message)
		c.Redirect(http.StatusFound, redirect)
	}
}

// setUserVIPByPayment set VIP true và set VIPExpires theo PeriodMonths của payment
func setUserVIPByPayment(db *gorm.DB, p *models.Payment) error {
	var user models.NguoiDung
	if err := db.First(&user, "id = ?", p.UserID).Error; err != nil {
		return err
	}
	now := time.Now()
	var newExpiry time.Time
	if user.VIPExpires != nil && user.VIPExpires.After(now) {
		// nếu còn VIP, gia hạn từ expires
		newExpiry = user.VIPExpires.AddDate(0, p.PeriodMonths, 0)
	} else {
		newExpiry = now.AddDate(0, p.PeriodMonths, 0)
	}

	if err := db.Model(&models.NguoiDung{}).Where("id = ?", p.UserID).Updates(map[string]interface{}{
		"vip":         true,
		"vip_expires": newExpiry,
	}).Error; err != nil {
		return err
	}
	return nil
}
