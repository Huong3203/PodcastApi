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
// Khuyến nghị: lấy 3 giá trị này trong https://business.momo.vn (mục Thông tin tích hợp)
// và thay vào đây. Bộ "MOMO/F8BBA..." là demo cũ nên có thể lỗi với /v2/gateway/api/create.
const (
	momoEndpoint    = "https://test-payment.momo.vn/v2/gateway/api/create"
	momoPartnerCode = "MOMO"                             // TODO: đổi thành partnerCode TEST của bạn
	momoAccessKey   = "F8BBA842ECF85"                    // TODO: đổi thành accessKey TEST của bạn
	momoSecretKey   = "K951B6PE1waDMi640xX08PD3vg6EkVlz" // TODO: đổi thành secretKey TEST của bạn
)

// ================== REQUEST BODY TỪ FE ==================
//
// FE gửi JSON:
//
//	{
//	  "user_id": "xxx",
//	  "amount": 10000,
//	  "orderInfo": "Mua gói VIP Sonify",
//	  "redirectUrl": "https://frontend.com/momo-return",
//	  "ipnUrl": "https://backend.com/momo/vip/ipn"
//	}
type MomoVIPRequest struct {
	UserID      string `json:"user_id"`     // user nâng cấp VIP
	Amount      int    `json:"amount"`      // số tiền (VND)
	OrderInfo   string `json:"orderInfo"`   // nội dung hiển thị trong MoMo
	RedirectUrl string `json:"redirectUrl"` // URL redirect sau thanh toán
	IpnUrl      string `json:"ipnUrl"`      // IPN callback URL (public)
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
		//
		// raw = "accessKey=" + accessKey +
		//       "&amount=" + amount +
		//       "&extraData=" + extraData +
		//       "&ipnUrl=" + ipnUrl +
		//       "&orderId=" + orderId +
		//       "&orderInfo=" + orderInfo +
		//       "&partnerCode=" + partnerCode +
		//       "&redirectUrl=" + redirectUrl +
		//       "&requestId=" + requestId +
		//       "&requestType=" + requestType;
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

		log.Println("MoMo signature:", signature)

		// --------- Payload gửi MoMo ---------
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
		// Model Payment cần có:
		// OrderID string `json:"order_id"`
		payment := models.Payment{
			ID:      uuid.NewString(),
			OrderID: orderId,
			UserID:  req.UserID,
			Amount:  req.Amount,
			Status:  "pending",
		}
		if err := db.Create(&payment).Error; err != nil {
			log.Println("DB create VIP payment error:", err)
		}

		// Thêm orderId/requestId vào response cho FE (chuẩn hóa)
		momoRes["orderId"] = orderId
		momoRes["requestId"] = requestId

		// --------- Trả kết quả cho FE ---------
		// FE sẽ lấy momoRes["payUrl"] hoặc deeplink để mở MoMo
		c.JSON(http.StatusOK, momoRes)
	}
}

// ================== IPN TỪ MOMO ==================
//
// Route: POST /momo/vip/ipn
// MoMo gọi vào URL này sau khi user thanh toán (kể cả thành công/ thất bại).
func MomoVIPIPN(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var ipnData map[string]interface{}
		if err := c.ShouldBindJSON(&ipnData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Println("MoMo IPN data:", ipnData)

		orderId, ok := ipnData["orderId"].(string)
		if !ok || orderId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing orderId"})
			return
		}

		// resultCode: 0 = thanh toán thành công
		resultCodeFloat, _ := ipnData["resultCode"].(float64)
		resultCode := int(resultCodeFloat)

		// TODO: nên verify thêm chữ ký m2signature trong ipnData nếu cần bảo mật tối đa

		// --------- Tìm payment theo OrderID ---------
		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderId).Error; err != nil {
			log.Println("Payment not found for orderId:", orderId)
			// vẫn trả 204 để MoMo không retry hoài
			c.Status(http.StatusNoContent)
			return
		}

		// --------- Cập nhật trạng thái payment + VIP user ---------
		if resultCode == 0 {
			// THANH TOÁN THÀNH CÔNG → chuyển từ pending → paid
			payment.Status = "paid"
			if err := db.Save(&payment).Error; err != nil {
				log.Println("Update payment success error:", err)
			}

			// Cập nhật VIP cho user (giả sử models.NguoiDung có field vip bool)
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

		// Chuẩn MoMo: IPN nên trả 204 No Content
		c.Status(http.StatusNoContent)
	}
}

// ================== API KIỂM TRA TRẠNG THÁI THANH TOÁN ==================
//
// Route: GET /payments/status/:orderId
// FE sau khi được redirect từ MoMo về redirectUrl có thể gọi API này để check kết quả.
func GetPaymentStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("orderId")
		if orderId == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing orderId"})
			return
		}

		var payment models.Payment
		if err := db.First(&payment, "order_id = ?", orderId).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"order_id": payment.OrderID,
			"user_id":  payment.UserID,
			"amount":   payment.Amount,
			"status":   payment.Status, // pending / paid / failed
		})
	}
}
