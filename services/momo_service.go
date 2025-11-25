package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/models"
	"gorm.io/gorm"
)

// Cấu hình MoMo — đổi thành của bạn
const (
	MomoEndpoint         = "https://test-payment.momo.vn/v2/gateway/api/create"
	MomoPartnerCode      = "MOMO"
	MomoAccessKey        = "F8BBA842ECF85"
	MomoSecretKey        = "K951B6PE1waDMi640xX08PD3vg6EkVlz"
	AppClientRedirectURL = "https://example.com/payment-result"
	// timeout HTTP
	httpTimeout = 10 * time.Second
)

// CreateMoMoRequest gửi request tạo payment lên MoMo (trả về response body as map)
func CreateMoMoRequest(payload map[string]interface{}) (map[string]interface{}, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Post(MomoEndpoint, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	var momoRes map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &momoRes); err != nil {
		return nil, fmt.Errorf("decode momo response error: %w", err)
	}
	return momoRes, nil
}

// BuildRawSignature tạo raw string theo spec MoMo v2 (phải đúng thứ tự)
func BuildRawSignature(fields map[string]string) string {
	// fields map phải chứa các key cần thiết; caller đảm bảo sắp xếp đúng
	// Ví dụ: accessKey, amount, extraData, ipnUrl, orderId, orderInfo, partnerCode, redirectUrl, requestId, requestType
	return fmt.Sprintf(
		"accessKey=%s&amount=%s&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
		fields["accessKey"],
		fields["amount"],
		fields["extraData"],
		fields["ipnUrl"],
		fields["orderId"],
		fields["orderInfo"],
		fields["partnerCode"],
		fields["redirectUrl"],
		fields["requestId"],
		fields["requestType"],
	)
}

// SignHmacSHA256
func SignHmacSHA256(message string) string {
	h := hmac.New(sha256.New, []byte(MomoSecretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature kiểm tra signature từ MoMo (IPN/RETURN)
// expectedRaw được build theo spec (caller phải build đúng)
func VerifySignature(expectedRaw, signature string) bool {
	expected := SignHmacSHA256(expectedRaw)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// CreatePaymentAndSave: helper tạo payment DB record
func CreatePaymentAndSave(db *gorm.DB, userID, orderId string, amount int, isRecurring bool, periodMonths int) (*models.Payment, error) {
	p := models.Payment{
		ID:           GenerateUUID(),
		OrderID:      orderId,
		UserID:       userID,
		Amount:       amount,
		Status:       "pending",
		IsRecurring:  isRecurring,
		PeriodMonths: periodMonths,
	}
	if err := db.Create(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// GenerateUUID dùng github.com/google/uuid hoặc tương tự
func GenerateUUID() string {
	return fmt.Sprintf("%s", models.UUIDString()) // bạn có thể thay bằng uuid.NewString()
}

// ParseBodyToMap tiện ích
func ParseBodyToMap(body []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}
