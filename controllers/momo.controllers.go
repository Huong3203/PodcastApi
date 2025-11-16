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

// Request từ frontend để mua VIP 1 lần
type MomoVIPRequest struct {
	UserID      string `json:"user_id"`     // ID user mua VIP
	Amount      int    `json:"amount"`      // Số tiền
	OrderInfo   string `json:"orderInfo"`   // Nội dung thanh toán
	RedirectUrl string `json:"redirectUrl"` // URL redirect sau khi thanh toán
	IpnUrl      string `json:"ipnUrl"`      // IPN callback URL
}

// Tạo payment VIP chỉ theo user
func CreateMomoVIPPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MomoVIPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Tạo orderId và requestId
		flake := sonyflake.NewSonyflake(sonyflake.Settings{})
		orderNum, _ := flake.NextID()
		requestNum, _ := flake.NextID()
		orderId := strconv.FormatUint(orderNum, 10)     // base10
		requestId := strconv.FormatUint(requestNum, 10) // base10

		// Thông tin Momo
		partnerCode := "MOMOIQA420180417"
		accessKey := "SvDmj2cOTYZmQQ3H"
		secretKey := "PPuDXq1KowPT1ftR8DvlQTHhC03aul17"
		requestType := "captureWallet"
		extraData := ""

		// Build raw signature theo đúng thứ tự Momo yêu cầu
		rawSignature := fmt.Sprintf(
			"accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
			accessKey, req.Amount, extraData, req.IpnUrl, orderId, req.OrderInfo, partnerCode, req.RedirectUrl, requestId, requestType,
		)

		// Tạo HMAC SHA256
		h := hmac.New(sha256.New, []byte(secretKey))
		h.Write([]byte(rawSignature))
		signature := hex.EncodeToString(h.Sum(nil))

		// Payload gửi lên Momo
		payload := map[string]interface{}{
			"partnerCode": partnerCode,
			"accessKey":   accessKey,
			"requestId":   requestId,
			"amount":      req.Amount, // số nguyên
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Momo VIP payment"})
			return
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Momo response"})
			return
		}

		// Lưu Payment VIP vào DB (chỉ user, không podcast)
		payment := models.Payment{
			ID:     uuid.NewString(),
			UserID: req.UserID,
			Amount: req.Amount,
			Status: "pending",
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Println("DB create VIP payment error:", err)
		}

		c.JSON(http.StatusOK, result)
	}
}

// IPN callback Momo VIP
func MomoVIPIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ipnData map[string]interface{}
		if err := c.ShouldBindJSON(&ipnData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		orderId, ok := ipnData["orderId"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing orderId"})
			return
		}

		// TODO: kiểm tra signature từ Momo trước khi cập nhật trạng thái

		// Cập nhật Payment status thành success
		var payment models.Payment
		if err := db.First(&payment, "id = ?", orderId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}
		payment.Status = "success"
		db.Save(&payment)

		// Cập nhật cột VIP của user
		db.Model(&models.NguoiDung{}).Where("id = ?", payment.UserID).Update("vip", true)

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
