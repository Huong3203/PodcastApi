package controllers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sony/sonyflake"
	"gorm.io/gorm"
)

type MomoRequest struct {
	PodcastID   string `json:"podcast_id"`
	Amount      int    `json:"amount"`
	OrderInfo   string `json:"orderInfo"`
	RedirectUrl string `json:"redirectUrl"`
	IpnUrl      string `json:"ipnUrl"`
}

// Tạo đơn Momo
func CreateMomoPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MomoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		flake := sonyflake.NewSonyflake(sonyflake.Settings{})
		a, _ := flake.NextID()
		b, _ := flake.NextID()
		orderId := strconv.FormatUint(a, 16)
		requestId := strconv.FormatUint(b, 16)

		partnerCode := "MOMOIQA420180417"
		accessKey := "SvDmj2cOTYZmQQ3H"
		secretKey := "PPuDXq1KowPT1ftR8DvlQTHhC03aul17"
		requestType := "captureWallet"
		extraData := ""

		rawSignature := fmt.Sprintf(
			"accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
			accessKey, req.Amount, extraData, req.IpnUrl, orderId, req.OrderInfo, partnerCode, req.RedirectUrl, requestId, requestType,
		)

		h := hmac.New(sha256.New, []byte(secretKey))
		h.Write([]byte(rawSignature))
		signature := hex.EncodeToString(h.Sum(nil))

		payload := map[string]interface{}{
			"partnerCode": partnerCode,
			"accessKey":   accessKey,
			"requestId":   requestId,
			"amount":      strconv.Itoa(req.Amount),
			"orderId":     orderId,
			"orderInfo":   req.OrderInfo,
			"redirectUrl": req.RedirectUrl,
			"ipnUrl":      req.IpnUrl,
			"extraData":   extraData,
			"requestType": requestType,
			"signature":   signature,
		}

		jsonPayload, _ := json.Marshal(payload)
		endpoint := "https://test-payment.momo.vn/v2/gateway/api/create"

		resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Momo payment"})
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		// Lưu Payment vào DB
		payment := models.Payment{
			ID:        uuid.NewString(),
			PodcastID: req.PodcastID,
			Amount:    req.Amount,
			Status:    "pending",
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Println("DB create payment error:", err)
		}

		c.JSON(http.StatusOK, result)
	}
}

// IPN Momo
func MomoIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ipnData map[string]interface{}
		if err := c.ShouldBindJSON(&ipnData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fmt.Println("IPN received:", ipnData)

		orderId, ok := ipnData["orderId"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing orderId"})
			return
		}

		// TODO: kiểm tra signature từ Momo trước khi cập nhật

		// Cập nhật status Payment
		db.Model(&models.Payment{}).Where("id = ?", orderId).Update("status", "success")

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
