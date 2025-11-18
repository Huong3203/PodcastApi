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

// ================== CẤU HÌNH MOMO SANDBOX ==================
//
// GỢI Ý: Lấy 3 giá trị này trong business.momo.vn (thông tin tích hợp)
const (
	momoEndpoint    = "https://test-payment.momo.vn/v2/gateway/api/create"
	momoPartnerCode = "MOMO"                             // TODO: đổi thành partnerCode TEST của bạn
	momoAccessKey   = "F8BBA842ECF85"                    // TODO: đổi thành accessKey TEST của bạn
	momoSecretKey   = "K951B6PE1waDMi640xX08PD3vg6EkVlz" // TODO: đổi thành secretKey TEST của bạn

	// URL app/frontend sau khi backend xử lý xong (FE/app của bạn)
	appClientRedirectURL = "https://example.com/payment-result" // hoặc deep link: "sonify://momo-result"
)

// ================== REQUEST BODY TỪ FE ==================

type MomoVIPRequest struct {
	UserID      string `json:"user_id"`     // user nâng cấp VIP
	Amount      int    `json:"amount"`      // số tiền (VND)
	OrderInfo   string `json:"orderInfo"`   // nội dung hiển thị trong MoMo
	RedirectUrl string `json:"redirectUrl"` // NÊN: https://api.yourdomain.com/momo/vip/return
	IpnUrl      string `json:"ipnUrl"`      // NÊN: https://api.yourdomain.com/momo/vip/ipn
}

// ================== TẠO GIAO DỊCH VIP ==================
//
// Route: POST /momo/vip/create
func CreateMomoVIPPayment(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MomoVIPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.UserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}
		if req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be > 0"})
			return
		}

		requestType := "captureWallet"
		extraData := "" // nếu muốn kèm thêm info thì encode base64 JSON

		// --------- Sinh orderId & requestId duy nhất ---------
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

		// --------- Build rawSignature đúng format MoMo v2 ---------
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

		log.Println("MoMo rawSignature:", rawSignature)

		// --------- Tạo HMAC SHA256 ---------
		h := hmac.New(sha256.New, []byte(momoSecretKey))
		h.Write([]byte(rawSignature))
		signature := hex.EncodeToString(h.Sum(nil))

		// --------- Payload gửi MoMo ---------
		payload := map[string]interface{}{
			"partnerCode": momoPartnerCode,
			"accessKey":   momoAccessKey,
			"requestId":   requestId,
			"amount":      req.Amount,
			"orderId":     orderId,
			"orderInfo":   req.OrderInfo,
			"redirectUrl": req.RedirectUrl, // /momo/vip/return
			"ipnUrl":      req.IpnUrl,      // /momo/vip/ipn
			"extraData":   extraData,
			"requestType": requestType,
			"lang":        "vi",
			"signature":   signature,
		}

		jsonPayload, _ := json.Marshal(payload)

		// --------- Gọi API MoMo ---------
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

		log.Println("MoMo response:", momoRes)

		// --------- Lưu Payment gắn với userID + orderId ---------
		payment := models.Payment{
			ID:      uuid.NewString(),
			OrderID: orderId,
			UserID:  req.UserID, // VIP theo user
			Amount:  req.Amount,
			Status:  "pending",
			// PodcastID để nil vì đây là gói VIP
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Println("DB create VIP payment error:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB create payment failed"})
			return
		}

		// Thêm orderId/requestId vào response cho FE
		momoRes["orderId"] = orderId
		momoRes["requestId"] = requestId

		// FE nhận payUrl → mở MoMo
		c.JSON(http.StatusOK, momoRes)
	}
}

// ================== IPN TỪ MOMO (SERVER-TO-SERVER) ==================
//
// Route: POST /momo/vip/ipn
func MomoVIPIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ipnData struct {
			OrderID    string `json:"orderId"`
			RequestID  string `json:"requestId"`
			ResultCode int    `json:"resultCode"`
			Message    string `json:"message"`
		}
		if err := c.ShouldBindJSON(&ipnData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Println("MoMo IPN data:", ipnData)

		if ipnData.OrderID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing orderId"})
			return
		}

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", ipnData.OrderID).Error; err != nil {
			log.Println("Payment not found for orderId:", ipnData.OrderID)
			c.Status(http.StatusNoContent)
			return
		}

		if ipnData.ResultCode == 0 {
			// THANH TOÁN THÀNH CÔNG
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("Update payment success error (IPN):", err)
			}
			// Set VIP user
			if err := db.Model(&models.NguoiDung{}).
				Where("id = ?", payment.UserID).
				Update("vip", true).Error; err != nil {
				log.Println("Update user VIP error (IPN):", err)
			}
		} else {
			payment.Status = "failed"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("Update payment failed error (IPN):", err)
			}
		}

		c.Status(http.StatusNoContent)
	}
}

// ================== RETURN URL: TRẢ VỀ APP + CẬP NHẬT STATUS ==================
//
// Route: GET /momo/vip/return
// Đây là URL bạn gán vào redirectUrl khi tạo payment.
func MomoVIPReturn(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Query("orderId")
		resultCodeStr := c.Query("resultCode")
		message := c.Query("message")

		log.Println("MoMo RETURN:", "orderId=", orderId, "resultCode=", resultCodeStr, "message=", message)

		if orderId == "" {
			c.String(http.StatusBadRequest, "Missing orderId")
			return
		}

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderId).Error; err != nil {
			log.Println("Payment not found in RETURN for orderId:", orderId)
			redirect := fmt.Sprintf("%s?orderId=%s&status=not_found", appClientRedirectURL, orderId)
			c.Redirect(http.StatusFound, redirect)
			return
		}

		if resultCodeStr == "0" {
			payment.Status = "success"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("Update payment success error (RETURN):", err)
			}
			if err := db.Model(&models.NguoiDung{}).
				Where("id = ?", payment.UserID).
				Update("vip", true).Error; err != nil {
				log.Println("Update user VIP error (RETURN):", err)
			}
			// Redirect về app với status success
			redirect := fmt.Sprintf("%s?orderId=%s&status=success", appClientRedirectURL, orderId)
			c.Redirect(http.StatusFound, redirect)
			return
		}

		payment.Status = "failed"
		if err := db.Save(&payment).Error; err != nil {
			log.Println("Update payment failed error (RETURN):", err)
		}
		redirect := fmt.Sprintf("%s?orderId=%s&status=failed&message=%s", appClientRedirectURL, orderId, message)
		c.Redirect(http.StatusFound, redirect)
	}
}
