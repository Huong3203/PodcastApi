package controllers

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sony/sonyflake"
	"gorm.io/gorm"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/services"
)

// ENV
var (
	momoPartnerCode = os.Getenv("MOMO_PARTNER_CODE")
	momoAccessKey   = os.Getenv("MOMO_ACCESS_KEY")
	momoSecretKey   = os.Getenv("MOMO_SECRET_KEY")
	momoEndpoint    = "https://test-payment.momo.vn/v2/gateway/api/create"
)

// ─────────────────────────────────────────────
// Create MoMo Payment (VIP)
// ─────────────────────────────────────────────
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

		// Check user
		var user models.NguoiDung
		if err := db.First(&user, "id = ?", req.UserID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user_not_found"})
			return
		}

		// Generate orderId / requestId
		flake := sonyflake.NewSonyflake(sonyflake.Settings{})
		oid, _ := flake.NextID()
		rid, _ := flake.NextID()
		orderId := strconv.FormatUint(oid, 10)
		requestId := strconv.FormatUint(rid, 10)

		amountStr := strconv.Itoa(req.Amount)

		// If ipn null -> dùng redirect luôn
		ipnUrl := req.IpnUrl
		if ipnUrl == "" {
			ipnUrl = req.RedirectUrl
		}

		// MoMo signature fields
		fields := map[string]string{
			"accessKey":   momoAccessKey,
			"amount":      amountStr,
			"extraData":   "",
			"ipnUrl":      ipnUrl,
			"orderId":     orderId,
			"orderInfo":   req.OrderInfo,
			"partnerCode": momoPartnerCode,
			"redirectUrl": req.RedirectUrl,
			"requestId":   requestId,
			"requestType": "captureWallet",
		}

		rawSignature := services.BuildRawSignature(fields)
		signature := services.SignSHA256(rawSignature, momoSecretKey)

		body := map[string]interface{}{
			"partnerCode": momoPartnerCode,
			"accessKey":   momoAccessKey,
			"requestId":   requestId,
			"amount":      amountStr,
			"orderId":     orderId,
			"orderInfo":   req.OrderInfo,
			"redirectUrl": req.RedirectUrl,
			"ipnUrl":      ipnUrl,
			"extraData":   "",
			"requestType": "captureWallet",
			"signature":   signature,
			"lang":        "vi",
		}

		// Call MoMo
		res, err := services.MoMoRequest(momoEndpoint, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "momo_failed", "detail": err.Error()})
			return
		}

		// Save payment
		db.Create(&models.Payment{
			ID:           uuid.NewString(),
			OrderID:      orderId,
			UserID:       req.UserID,
			Amount:       req.Amount,
			Status:       "pending",
			IsRecurring:  req.AutoRenew,
			PeriodMonths: req.PeriodMonths,
		})

		res["orderId"] = orderId
		res["requestId"] = requestId

		c.JSON(http.StatusOK, res)
	}
}

// ─────────────────────────────────────────────
// MoMo IPN Callback
// ─────────────────────────────────────────────
func MomoIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ipn map[string]interface{}
		if err := c.ShouldBindJSON(&ipn); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_ipn"})
			return
		}

		// Build raw signature (MoMo spec)
		raw := fmt.Sprintf(
			"accessKey=%s&amount=%v&extraData=%v&message=%v&orderId=%v&orderInfo=%v&orderType=%v&partnerCode=%s&payType=%v&requestId=%v&responseTime=%v&resultCode=%v&transId=%v",
			momoAccessKey,
			ipn["amount"],
			ipn["extraData"],
			ipn["message"],
			ipn["orderId"],
			ipn["orderInfo"],
			ipn["orderType"],
			momoPartnerCode,
			ipn["payType"],
			ipn["requestId"],
			ipn["responseTime"],
			ipn["resultCode"],
			ipn["transId"],
		)

		sign := services.SignSHA256(raw, momoSecretKey)
		if sign != ipn["signature"] {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid_signature"})
			return
		}

		orderID := ipn["orderId"].(string)
		resultCode := int(ipn["resultCode"].(float64))

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"message": "payment_not_found"})
			return
		}

		// FAILED
		if resultCode != 0 {
			payment.Status = "failed"
			db.Save(&payment)
			c.JSON(http.StatusOK, gin.H{"message": "ipn_received"})
			return
		}

		// SUCCESS
		payment.Status = "success"
		payment.UpdatedAt = time.Now()
		payment.ExpiresAt = time.Now().AddDate(0, payment.PeriodMonths, 0)
		db.Save(&payment)

		// Upgrade user to VIP
		db.Model(&models.NguoiDung{}).
			Where("id = ?", payment.UserID).
			Updates(map[string]interface{}{
				"is_vip":      true,
				"vip_expires": payment.ExpiresAt,
			})

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

// ─────────────────────────────────────────────
// MoMo Redirect URL Handler
// ─────────────────────────────────────────────
func MomoReturnURL() gin.HandlerFunc {
	return func(c *gin.Context) {
		resultCode := c.Query("resultCode")
		orderId := c.Query("orderId")

		if resultCode == "0" {
			c.Redirect(http.StatusFound, "sonify://payment-success?orderId="+orderId)
		} else {
			c.Redirect(http.StatusFound, "sonify://payment-failed?orderId="+orderId)
		}
	}
}
