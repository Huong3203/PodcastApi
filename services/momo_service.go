package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/google/uuid"
)

type MoMoCreateRequest struct {
	PartnerCode  string `json:"partnerCode"`
	AccessKey    string `json:"accessKey"`
	RequestID    string `json:"requestId"`
	Amount       string `json:"amount"`
	OrderID      string `json:"orderId"`
	OrderInfo    string `json:"orderInfo"`
	RedirectURL  string `json:"redirectUrl"`
	IPNUrl       string `json:"ipnUrl"`
	RequestType  string `json:"requestType"`
	ExtraData    string `json:"extraData"`
	Lang         string `json:"lang"`
	AutoCapture  bool   `json:"autoCapture"`
	OrderGroupID string `json:"orderGroupId"`
	Signature    string `json:"signature"`
}

type MoMoCreateResponse struct {
	PartnerCode  string `json:"partnerCode"`
	OrderID      string `json:"orderId"`
	RequestID    string `json:"requestId"`
	Amount       int64  `json:"amount"`
	ResponseTime int64  `json:"responseTime"`
	Message      string `json:"message"`
	ResultCode   int    `json:"resultCode"`
	PayURL       string `json:"payUrl"`
	Deeplink     string `json:"deeplink"`
	QRCodeURL    string `json:"qrCodeUrl"`
}

type MoMoIPNRequest struct {
	PartnerCode  string `json:"partnerCode"`
	OrderID      string `json:"orderId"`
	RequestID    string `json:"requestId"`
	Amount       int64  `json:"amount"`
	OrderInfo    string `json:"orderInfo"`
	OrderType    string `json:"orderType"`
	TransID      int64  `json:"transId"`
	ResultCode   int    `json:"resultCode"`
	Message      string `json:"message"`
	PayType      string `json:"payType"`
	ResponseTime int64  `json:"responseTime"`
	ExtraData    string `json:"extraData"`
	Signature    string `json:"signature"`
}

func GenerateRandomID(prefix string) string {
	return fmt.Sprintf("%s%d%03d", prefix, time.Now().Unix(), time.Now().Nanosecond()%1000)
}

func CreateMoMoSignature(rawSignature, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(rawSignature))
	return hex.EncodeToString(h.Sum(nil))
}

func CreateMoMoPayment(userID string, amount int64, vipDuration int, orderInfo string) (*MoMoCreateResponse, *models.Payment, error) {
	cfg := config.GetMoMoConfig()

	orderID := GenerateRandomID("inv-")
	requestID := GenerateRandomID("req-")
	amountStr := fmt.Sprintf("%d", amount)

	payment := &models.Payment{
		ID:          uuid.New().String(),
		UserID:      userID,
		OrderID:     orderID,
		RequestID:   requestID,
		Amount:      amount,
		Status:      "PENDING",
		PaymentType: "VIP_UPGRADE",
		VIPDuration: vipDuration,
	}

	if err := config.DB.Create(payment).Error; err != nil {
		return nil, nil, fmt.Errorf("không thể tạo bản ghi thanh toán: %w", err)
	}

	rawSignature := fmt.Sprintf(
		"accessKey=%s&amount=%s&extraData=&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=captureWallet",
		cfg.AccessKey, amountStr, cfg.IPNUrl, orderID, orderInfo, cfg.PartnerCode, cfg.RedirectURL, requestID,
	)
	signature := CreateMoMoSignature(rawSignature, cfg.SecretKey)

	reqBody := MoMoCreateRequest{
		PartnerCode:  cfg.PartnerCode,
		AccessKey:    cfg.AccessKey,
		RequestID:    requestID,
		Amount:       amountStr,
		OrderID:      orderID,
		OrderInfo:    orderInfo,
		RedirectURL:  cfg.RedirectURL,
		IPNUrl:       cfg.IPNUrl,
		RequestType:  "captureWallet",
		ExtraData:    "",
		Lang:         "vi",
		AutoCapture:  true,
		OrderGroupID: "",
		Signature:    signature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("không thể marshal request: %w", err)
	}

	resp, err := http.Post(cfg.CreateEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("không thể kết nối MoMo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("không thể đọc response: %w", err)
	}

	var momoResp MoMoCreateResponse
	if err := json.Unmarshal(body, &momoResp); err != nil {
		return nil, nil, fmt.Errorf("không thể parse response: %w", err)
	}

	paymentInfo, _ := json.Marshal(momoResp)
	config.DB.Model(payment).Updates(map[string]interface{}{
		"payment_info": string(paymentInfo),
	})

	if momoResp.ResultCode != 0 {
		config.DB.Model(payment).Update("status", "FAILED")
		return nil, nil, fmt.Errorf("MoMo error: %s (code: %d)", momoResp.Message, momoResp.ResultCode)
	}

	return &momoResp, payment, nil
}

func VerifyMoMoSignature(ipnReq *MoMoIPNRequest, secretKey string) bool {
	rawSignature := fmt.Sprintf(
		"accessKey=%s&amount=%d&extraData=%s&message=%s&orderId=%s&orderInfo=%s&orderType=%s&partnerCode=%s&payType=%s&requestId=%s&responseTime=%d&resultCode=%d&transId=%d",
		config.GetMoMoConfig().AccessKey,
		ipnReq.Amount,
		ipnReq.ExtraData,
		ipnReq.Message,
		ipnReq.OrderID,
		ipnReq.OrderInfo,
		ipnReq.OrderType,
		ipnReq.PartnerCode,
		ipnReq.PayType,
		ipnReq.RequestID,
		ipnReq.ResponseTime,
		ipnReq.ResultCode,
		ipnReq.TransID,
	)
	expectedSignature := CreateMoMoSignature(rawSignature, secretKey)
	return expectedSignature == ipnReq.Signature
}

func ProcessMoMoIPN(ipnReq *MoMoIPNRequest) error {
	cfg := config.GetMoMoConfig()

	if !VerifyMoMoSignature(ipnReq, cfg.SecretKey) {
		return errors.New("chữ ký không hợp lệ")
	}

	var payment models.Payment
	if err := config.DB.Where("order_id = ?", ipnReq.OrderID).First(&payment).Error; err != nil {
		return fmt.Errorf("không tìm thấy đơn hàng: %w", err)
	}

	if payment.Status == "SUCCESS" {
		return nil
	}

	if ipnReq.ResultCode == 0 {
		payment.Status = "SUCCESS"

		var user models.NguoiDung
		if err := config.DB.Where("id = ?", payment.UserID).First(&user).Error; err != nil {
			return fmt.Errorf("không tìm thấy người dùng: %w", err)
		}

		var vipExpires time.Time
		if user.VIP && user.VIPExpires != nil && user.VIPExpires.After(time.Now()) {
			vipExpires = user.VIPExpires.AddDate(0, 0, payment.VIPDuration)
		} else {
			vipExpires = time.Now().AddDate(0, 0, payment.VIPDuration)
		}

		if err := config.DB.Model(&user).Updates(map[string]interface{}{
			"vip":         true,
			"vip_expires": vipExpires,
		}).Error; err != nil {
			return fmt.Errorf("không thể cập nhật VIP: %w", err)
		}

		message := fmt.Sprintf("Bạn đã nâng cấp VIP thành công! Hạn sử dụng đến %s", vipExpires.Format("02/01/2006"))
		if err := CreateNotification(payment.UserID, "", "vip_upgrade", message); err != nil {
			fmt.Println("⚠️ Lỗi tạo thông báo:", err)
		}
	} else {
		payment.Status = "FAILED"
	}

	paymentInfo, _ := json.Marshal(ipnReq)
	payment.PaymentInfo = string(paymentInfo)

	if err := config.DB.Save(&payment).Error; err != nil {
		return fmt.Errorf("không thể cập nhật payment: %w", err)
	}

	return nil
}

// ✅ HÀM MỚI: Check status từ query params (cho MoMo UAT)
func CheckPaymentStatusByOrderID(orderID string) (*models.Payment, error) {
	var payment models.Payment
	if err := config.DB.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		return nil, fmt.Errorf("không tìm thấy đơn hàng: %w", err)
	}
	return &payment, nil
}

// package services

// import (
// 	"bytes"
// 	"crypto/hmac"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"net/http"
// 	"time"

// 	"github.com/Huong3203/APIPodcast/config"
// 	"github.com/Huong3203/APIPodcast/models"
// 	"github.com/google/uuid"
// )

// type MoMoCreateRequest struct {
// 	PartnerCode  string `json:"partnerCode"`
// 	AccessKey    string `json:"accessKey"`
// 	RequestID    string `json:"requestId"`
// 	Amount       string `json:"amount"`
// 	OrderID      string `json:"orderId"`
// 	OrderInfo    string `json:"orderInfo"`
// 	RedirectURL  string `json:"redirectUrl"`
// 	IPNUrl       string `json:"ipnUrl"`
// 	RequestType  string `json:"requestType"`
// 	ExtraData    string `json:"extraData"`
// 	Lang         string `json:"lang"`
// 	AutoCapture  bool   `json:"autoCapture"`
// 	OrderGroupID string `json:"orderGroupId"`
// 	Signature    string `json:"signature"`
// }

// type MoMoCreateResponse struct {
// 	PartnerCode  string `json:"partnerCode"`
// 	OrderID      string `json:"orderId"`
// 	RequestID    string `json:"requestId"`
// 	Amount       int64  `json:"amount"`
// 	ResponseTime int64  `json:"responseTime"`
// 	Message      string `json:"message"`
// 	ResultCode   int    `json:"resultCode"`
// 	PayURL       string `json:"payUrl"`
// 	Deeplink     string `json:"deeplink"`
// 	QRCodeURL    string `json:"qrCodeUrl"`
// }

// type MoMoIPNRequest struct {
// 	PartnerCode  string `json:"partnerCode"`
// 	OrderID      string `json:"orderId"`
// 	RequestID    string `json:"requestId"`
// 	Amount       int64  `json:"amount"`
// 	OrderInfo    string `json:"orderInfo"`
// 	OrderType    string `json:"orderType"`
// 	TransID      int64  `json:"transId"`
// 	ResultCode   int    `json:"resultCode"`
// 	Message      string `json:"message"`
// 	PayType      string `json:"payType"`
// 	ResponseTime int64  `json:"responseTime"`
// 	ExtraData    string `json:"extraData"`
// 	Signature    string `json:"signature"`
// }

// func GenerateRandomID(prefix string) string {
// 	return fmt.Sprintf("%s%d%03d", prefix, time.Now().Unix(), time.Now().Nanosecond()%1000)
// }

// func CreateMoMoSignature(rawSignature, secretKey string) string {
// 	h := hmac.New(sha256.New, []byte(secretKey))
// 	h.Write([]byte(rawSignature))
// 	return hex.EncodeToString(h.Sum(nil))
// }

// func CreateMoMoPayment(userID string, amount int64, vipDuration int, orderInfo string) (*MoMoCreateResponse, *models.Payment, error) {
// 	cfg := config.GetMoMoConfig()

// 	orderID := GenerateRandomID("inv-")
// 	requestID := GenerateRandomID("req-")
// 	amountStr := fmt.Sprintf("%d", amount)

// 	// Tạo bản ghi Payment trong DB
// 	payment := &models.Payment{
// 		ID:          uuid.New().String(),
// 		UserID:      userID,
// 		OrderID:     orderID,
// 		RequestID:   requestID,
// 		Amount:      amount,
// 		Status:      "PENDING",
// 		PaymentType: "VIP_UPGRADE",
// 		VIPDuration: vipDuration,
// 	}

// 	if err := config.DB.Create(payment).Error; err != nil {
// 		return nil, nil, fmt.Errorf("không thể tạo bản ghi thanh toán: %w", err)
// 	}

// 	// Tạo chữ ký
// 	rawSignature := fmt.Sprintf(
// 		"accessKey=%s&amount=%s&extraData=&ipnUrl=%s&orderId=%s&orderInfo=%s&partnerCode=%s&redirectUrl=%s&requestId=%s&requestType=captureWallet",
// 		cfg.AccessKey, amountStr, cfg.IPNUrl, orderID, orderInfo, cfg.PartnerCode, cfg.RedirectURL, requestID,
// 	)
// 	signature := CreateMoMoSignature(rawSignature, cfg.SecretKey)

// 	// Tạo request body
// 	reqBody := MoMoCreateRequest{
// 		PartnerCode:  cfg.PartnerCode,
// 		AccessKey:    cfg.AccessKey,
// 		RequestID:    requestID,
// 		Amount:       amountStr,
// 		OrderID:      orderID,
// 		OrderInfo:    orderInfo,
// 		RedirectURL:  cfg.RedirectURL,
// 		IPNUrl:       cfg.IPNUrl,
// 		RequestType:  "captureWallet",
// 		ExtraData:    "",
// 		Lang:         "vi",
// 		AutoCapture:  true,
// 		OrderGroupID: "",
// 		Signature:    signature,
// 	}

// 	jsonData, err := json.Marshal(reqBody)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("không thể marshal request: %w", err)
// 	}

// 	// Gọi API MoMo
// 	resp, err := http.Post(cfg.CreateEndpoint, "application/json", bytes.NewBuffer(jsonData))
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("không thể kết nối MoMo: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("không thể đọc response: %w", err)
// 	}

// 	var momoResp MoMoCreateResponse
// 	if err := json.Unmarshal(body, &momoResp); err != nil {
// 		return nil, nil, fmt.Errorf("không thể parse response: %w", err)
// 	}

// 	// Lưu thông tin response vào DB
// 	paymentInfo, _ := json.Marshal(momoResp)
// 	config.DB.Model(payment).Updates(map[string]interface{}{
// 		"payment_info": string(paymentInfo),
// 	})

// 	if momoResp.ResultCode != 0 {
// 		config.DB.Model(payment).Update("status", "FAILED")
// 		return nil, nil, fmt.Errorf("MoMo error: %s (code: %d)", momoResp.Message, momoResp.ResultCode)
// 	}

// 	return &momoResp, payment, nil
// }

// func VerifyMoMoSignature(ipnReq *MoMoIPNRequest, secretKey string) bool {
// 	rawSignature := fmt.Sprintf(
// 		"accessKey=%s&amount=%d&extraData=%s&message=%s&orderId=%s&orderInfo=%s&orderType=%s&partnerCode=%s&payType=%s&requestId=%s&responseTime=%d&resultCode=%d&transId=%d",
// 		config.GetMoMoConfig().AccessKey,
// 		ipnReq.Amount,
// 		ipnReq.ExtraData,
// 		ipnReq.Message,
// 		ipnReq.OrderID,
// 		ipnReq.OrderInfo,
// 		ipnReq.OrderType,
// 		ipnReq.PartnerCode,
// 		ipnReq.PayType,
// 		ipnReq.RequestID,
// 		ipnReq.ResponseTime,
// 		ipnReq.ResultCode,
// 		ipnReq.TransID,
// 	)
// 	expectedSignature := CreateMoMoSignature(rawSignature, secretKey)
// 	return expectedSignature == ipnReq.Signature
// }

// func ProcessMoMoIPN(ipnReq *MoMoIPNRequest) error {
// 	cfg := config.GetMoMoConfig()

// 	// Xác thực chữ ký
// 	if !VerifyMoMoSignature(ipnReq, cfg.SecretKey) {
// 		return errors.New("chữ ký không hợp lệ")
// 	}

// 	// Tìm payment trong DB
// 	var payment models.Payment
// 	if err := config.DB.Where("order_id = ?", ipnReq.OrderID).First(&payment).Error; err != nil {
// 		return fmt.Errorf("không tìm thấy đơn hàng: %w", err)
// 	}

// 	// Kiểm tra nếu đã xử lý rồi
// 	if payment.Status == "SUCCESS" {
// 		return nil // Đã xử lý rồi, bỏ qua
// 	}

// 	// Cập nhật trạng thái payment
// 	if ipnReq.ResultCode == 0 {
// 		payment.Status = "SUCCESS"

// 		// Nâng cấp tài khoản VIP cho user
// 		var user models.NguoiDung
// 		if err := config.DB.Where("id = ?", payment.UserID).First(&user).Error; err != nil {
// 			return fmt.Errorf("không tìm thấy người dùng: %w", err)
// 		}

// 		// Tính thời gian hết hạn VIP
// 		var vipExpires time.Time
// 		if user.VIP && user.VIPExpires != nil && user.VIPExpires.After(time.Now()) {
// 			// Nếu đang VIP, cộng thêm thời gian
// 			vipExpires = user.VIPExpires.AddDate(0, 0, payment.VIPDuration)
// 		} else {
// 			// Nếu chưa VIP hoặc đã hết hạn, tính từ bây giờ
// 			vipExpires = time.Now().AddDate(0, 0, payment.VIPDuration)
// 		}

// 		// Cập nhật user
// 		if err := config.DB.Model(&user).Updates(map[string]interface{}{
// 			"vip":         true,
// 			"vip_expires": vipExpires,
// 		}).Error; err != nil {
// 			return fmt.Errorf("không thể cập nhật VIP: %w", err)
// 		}

// 		// Tạo thông báo
// 		message := fmt.Sprintf("Bạn đã nâng cấp VIP thành công! Hạn sử dụng đến %s", vipExpires.Format("02/01/2006"))
// 		if err := CreateNotification(payment.UserID, "", "vip_upgrade", message); err != nil {
// 			fmt.Println("⚠️ Lỗi tạo thông báo:", err)
// 		}
// 	} else {
// 		payment.Status = "FAILED"
// 	}

// 	// Lưu payment
// 	paymentInfo, _ := json.Marshal(ipnReq)
// 	payment.PaymentInfo = string(paymentInfo)

// 	if err := config.DB.Save(&payment).Error; err != nil {
// 		return fmt.Errorf("không thể cập nhật payment: %w", err)
// 	}

// 	return nil
// }
