package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

// Tạo raw signature từ map field
func BuildRawSignature(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+fields[k])
	}
	return strings.Join(parts, "&")
}

// Ký SHA256
func SignSHA256(raw, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}

// Gửi request MoMo
func MoMoRequest(url string, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)
	return result, nil
}

// import (
// 	"bytes"
// 	"crypto/hmac"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"time"

// 	"github.com/Huong3203/APIPodcast/models"
// 	"github.com/google/uuid"
// 	"gorm.io/gorm"
// )

// // Cấu hình MoMo
// const (
// 	MomoEndpoint         = "https://test-payment.momo.vn/v2/gateway/api/create"
// 	MomoPartnerCode      = "MOMO"
// 	MomoAccessKey        = "F8BBA842ECF85"
// 	MomoSecretKey        = "K951B6PE1waDMi640xX08PD3vg6EkVlz"
// 	AppClientRedirectURL = "https://example.com/payment-result"

// 	httpTimeout = 10 * time.Second
// )

// // Gửi request tạo thanh toán MoMo
// func CreateMoMoRequest(payload map[string]interface{}) (map[string]interface{}, error) {
// 	jsonPayload, err := json.Marshal(payload)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client := &http.Client{Timeout: httpTimeout}
// 	resp, err := client.Post(MomoEndpoint, "application/json", bytes.NewBuffer(jsonPayload))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	bodyBytes, _ := io.ReadAll(resp.Body)

// 	var momoRes map[string]interface{}
// 	if err := json.Unmarshal(bodyBytes, &momoRes); err != nil {
// 		return nil, fmt.Errorf("decode momo response error: %w", err)
// 	}

// 	return momoRes, nil
// }

// // Build raw signature đúng thứ tự
// func BuildRawSignature(fields map[string]string) string {
// 	return fmt.Sprintf(
// 		"accessKey=%s&amount=%s&extraData=%s&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=%s",
// 		fields["accessKey"],
// 		fields["amount"],
// 		fields["extraData"],
// 		fields["ipnUrl"],
// 		fields["orderId"],
// 		fields["orderInfo"],
// 		fields["partnerCode"],
// 		fields["redirectUrl"],
// 		fields["requestId"],
// 		fields["requestType"],
// 	)
// }

// // Tạo chữ ký HMAC SHA256
// func SignHmacSHA256(message string) string {
// 	h := hmac.New(sha256.New, []byte(MomoSecretKey))
// 	h.Write([]byte(message))
// 	return hex.EncodeToString(h.Sum(nil))
// }

// // Verify chữ ký trả về từ MoMo
// func VerifySignature(expectedRaw, signature string) bool {
// 	expected := SignHmacSHA256(expectedRaw)
// 	return hmac.Equal([]byte(expected), []byte(signature))
// }

// // Tạo payment + lưu DB
// func CreatePaymentAndSave(db *gorm.DB, userID, orderId string, amount int, isRecurring bool, periodMonths int) (*models.Payment, error) {
// 	p := models.Payment{
// 		ID:           GenerateUUID(),
// 		OrderID:      orderId,
// 		UserID:       userID,
// 		Amount:       amount,
// 		Status:       "pending",
// 		IsRecurring:  isRecurring,
// 		PeriodMonths: periodMonths,
// 	}

// 	if err := db.Create(&p).Error; err != nil {
// 		return nil, err
// 	}
// 	return &p, nil
// }

// // Generate UUID (GOOGLE UUID)
// func GenerateUUID() string {
// 	return uuid.NewString()
// }

// // Parse body JSON → map
// func ParseBodyToMap(body []byte) (map[string]interface{}, error) {
// 	var m map[string]interface{}
// 	if err := json.Unmarshal(body, &m); err != nil {
// 		return nil, err
// 	}
// 	return m, nil
// }
