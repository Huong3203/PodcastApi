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

// ====== CẤU HÌNH MOMO SANDBOX (KEY GỐC DEMO) ======
const (
	momoEndpoint    = "https://test-payment.momo.vn/v2/gateway/api/create"
	momoPartnerCode = "MOMOIQA420180417"
	momoAccessKey   = "SvDmj2cOTYZmQQ3H"
	momoSecretKey   = "PPuDXq1KowPT1ftR8DvlQTHhC03aul17"
)

// Body từ FE, KHÔNG cần userId (lấy từ URL)
type MomoVIPRequest struct {
	Amount      int    `json:"amount"`      // số tiền (VND)
	OrderInfo   string `json:"orderInfo"`   // nội dung thanh toán
	RedirectUrl string `json:"redirectUrl"` // URL redirect sau thanh toán
	IpnUrl      string `json:"ipnUrl"`      // IPN callback URL
}

// POST /users/:userId/momo/vip/create
func CreateMomoVIPPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy userId từ URL
		userID := c.Param("userId")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId in URL"})
			return
		}

		var req MomoVIPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		requestType := "captureWallet"
		extraData := "" // nếu muốn kèm thêm info thì encode base64 JSON

		// Tạo orderId & requestId
		flake := sonyflake.NewSonyflake(sonyflake.Settings{})
		if flake == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Sonyflake init error"})
			return
		}
		orderNum, err := flake.NextID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Generate orderId failed"})
			return
		}
		requestNum, err := flake.NextID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Generate requestId failed"})
			return
		}
		orderId := strconv.FormatUint(orderNum, 10)
		requestId := strconv.FormatUint(requestNum, 10)

		// Build rawSignature đúng format tài liệu MoMo
		rawSignature := fmt.Sprintf(
			"accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
			momoAccessKey,
			req.Amount,
			extraData,
			req.IpnUrl,
			orderId,
			req.OrderInfo,
			momoPartnerCode,
			req.RedirectUrl,
			requestId,
			requestType,
		)

		// Tạo HMAC SHA256
		h := hmac.New(sha256.New, []byte(momoSecretKey))
		h.Write([]byte(rawSignature))
		signature := hex.EncodeToString(h.Sum(nil))

		// Payload gửi MoMo
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
			"signature":   signature,
		}

		jsonPayload, _ := json.Marshal(payload)

		resp, err := http.Post(momoEndpoint, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			log.Println("MoMo create error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Momo VIP payment"})
			return
		}
		defer resp.Body.Close()

		var momoRes map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&momoRes); err != nil {
			log.Println("Decode MoMo response error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Momo response"})
			return
		}

		// Lưu Payment gắn với userID
		payment := models.Payment{
			ID:     uuid.NewString(),
			UserID: userID,
			Amount: req.Amount,
			Status: "pending",
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Println("DB create VIP payment error:", err)
		}

		// Trả về cho FE cả payUrl/deeplink
		momoRes["orderId"] = orderId
		momoRes["requestId"] = requestId

		c.JSON(http.StatusOK, momoRes)
	}
}

// IPN callback MoMo VIP (không cần key, trừ khi bạn verify chữ ký)
func MomoVIPIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ipnData map[string]interface{}
		if err := c.ShouldBindJSON(&ipnData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		orderId, ok := ipnData["orderId"].(string)
		if !ok || orderId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing orderId"})
			return
		}

		resultCodeFloat, _ := ipnData["resultCode"].(float64)
		resultCode := int(resultCodeFloat)

		// TODO: nếu muốn chuẩn security thì verify m2signature ở đây

		// Tìm payment theo OrderID
		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderId).Error; err != nil {
			log.Println("Payment not found for orderId:", orderId)
			c.Status(http.StatusNoContent)
			return
		}

		if resultCode == 0 {
			payment.Status = "success"
			db.Save(&payment)

			// Set VIP đúng user
			db.Model(&models.NguoiDung{}).
				Where("id = ?", payment.UserID).
				Update("vip", true)
		} else {
			payment.Status = "failed"
			db.Save(&payment)
		}

		c.Status(http.StatusNoContent)
	}
}
