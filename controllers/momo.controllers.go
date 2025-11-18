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
	"os"
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
	Amount      int    `json:"amount"`      // Số tiền (VND)
	OrderInfo   string `json:"orderInfo"`   // Nội dung thanh toán
	RedirectUrl string `json:"redirectUrl"` // URL redirect sau khi thanh toán thành công/thất bại
	IpnUrl      string `json:"ipnUrl"`      // URL callback (IPN) của backend
}

const momoEndpoint = "https://test-payment.momo.vn/v2/gateway/api/create"

// Tạo payment VIP theo user (gọi từ FE)
func CreateMomoVIPPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MomoVIPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Lấy key từ ENV (khuyến nghị)
		partnerCode := os.Getenv("MOMO_PARTNER_CODE")
		accessKey := os.Getenv("MOMO_ACCESS_KEY")
		secretKey := os.Getenv("MOMO_SECRET_KEY")

		// Nếu chưa set ENV thì fallback về sandbox demo (chỉ dùng test)
		if partnerCode == "" || accessKey == "" || secretKey == "" {
			partnerCode = "MOMOIQA420180417"
			accessKey = "SvDmj2cOTYZmQQ3H"
			secretKey = "PPuDXq1KowPT1ftR8DvlQTHhC03aul17"
		}

		requestType := "captureWallet"
		extraData := "" // nếu cần truyền thêm thông tin thì encode base64 JSON

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

		// Build rawSignature đúng format MoMo
		rawSignature := fmt.Sprintf(
			"accessKey=%s&amount=%d&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
			accessKey,
			req.Amount,
			extraData,
			req.IpnUrl,
			orderId,
			req.OrderInfo,
			partnerCode,
			req.RedirectUrl,
			requestId,
			requestType,
		)

		h := hmac.New(sha256.New, []byte(secretKey))
		h.Write([]byte(rawSignature))
		signature := hex.EncodeToString(h.Sum(nil))

		// Payload gửi lên MoMo
		payload := map[string]interface{}{
			"partnerCode": partnerCode,
			"accessKey":   accessKey,
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

		// Lưu payment: nhớ map OrderID với orderId của MoMo
		payment := models.Payment{
			ID:     uuid.NewString(),
			UserID: req.UserID,
			Amount: req.Amount,
			Status: "pending",
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Println("DB create VIP payment error:", err)
		}

		// Trả kết quả cho FE (bao gồm payUrl, deeplink, …)
		momoRes["orderId"] = orderId
		momoRes["requestId"] = requestId

		c.JSON(http.StatusOK, momoRes)
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
		if !ok || orderId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing orderId"})
			return
		}

		// resultCode: 0 => thanh toán thành công
		resultCodeFloat, _ := ipnData["resultCode"].(float64)
		resultCode := int(resultCodeFloat)

		// TODO: verify chữ ký m2signature nếu muốn chuẩn security theo docs

		// Lấy payment theo OrderID
		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderId).Error; err != nil {
			log.Println("Payment not found for orderId:", orderId)
			// nên trả 204 để MoMo không retry quá nhiều
			c.Status(http.StatusNoContent)
			return
		}

		if resultCode == 0 {
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("Update payment success error:", err)
			}

			// Cập nhật VIP user
			if err := db.Model(&models.NguoiDung{}).
				Where("id = ?", payment.UserID).
				Update("vip", true).Error; err != nil {
				log.Println("Update user VIP error:", err)
			}
		} else {
			payment.Status = "failed"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("Update payment failed error:", err)
			}
		}

		// Chuẩn docs: 204, không body
		c.Status(http.StatusNoContent)
	}
}
