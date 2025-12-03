package controllers

import (
	"fmt"
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/services"
	"github.com/gin-gonic/gin"
)

// POST /api/payment/momo/create
type CreateMoMoPaymentInput struct {
	Amount      int64  `json:"amount" binding:"required,min=10000"`
	VIPDuration int    `json:"vip_duration" binding:"required,oneof=30 90 365"`
	OrderInfo   string `json:"order_info"`
}

func CreateMoMoPayment(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}

	var input CreateMoMoPaymentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Tạo orderInfo nếu không có
	if input.OrderInfo == "" {
		input.OrderInfo = fmt.Sprintf("Nâng cấp VIP %d ngày", input.VIPDuration)
	}

	// Gọi service tạo payment
	momoResp, payment, err := services.CreateMoMoPayment(userID, input.Amount, input.VIPDuration, input.OrderInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Tạo link thanh toán thành công",
		"pay_url":    momoResp.PayURL,
		"deeplink":   momoResp.Deeplink,
		"qr_code":    momoResp.QRCodeURL,
		"order_id":   payment.OrderID,
		"request_id": payment.RequestID,
	})
}

// GET /api/payment/momo/callback (redirect từ MoMo)
func MoMoCallback(c *gin.Context) {
	orderID := c.Query("orderId")
	resultCode := c.Query("resultCode")

	var payment models.Payment
	if err := config.DB.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		c.HTML(http.StatusOK, "payment_error.html", gin.H{
			"message": "Không tìm thấy đơn hàng",
		})
		return
	}

	if resultCode == "0" {
		c.HTML(http.StatusOK, "payment_success.html", gin.H{
			"message":  "Thanh toán thành công! Tài khoản VIP của bạn đã được kích hoạt.",
			"order_id": orderID,
		})
	} else {
		c.HTML(http.StatusOK, "payment_error.html", gin.H{
			"message": "Thanh toán thất bại hoặc đã bị hủy",
		})
	}
}

// POST /api/payment/momo/ipn (webhook từ MoMo)
func MoMoIPN(c *gin.Context) {
	var ipnReq services.MoMoIPNRequest
	if err := c.ShouldBindJSON(&ipnReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Xử lý IPN
	if err := services.ProcessMoMoIPN(&ipnReq); err != nil {
		fmt.Printf("❌ Lỗi xử lý IPN: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Trả về response cho MoMo
	c.JSON(http.StatusOK, gin.H{
		"partnerCode": ipnReq.PartnerCode,
		"orderId":     ipnReq.OrderID,
		"requestId":   ipnReq.RequestID,
		"resultCode":  0,
		"message":     "success",
	})
}

// GET /api/payment/history (lịch sử thanh toán)
func GetPaymentHistory(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}

	var payments []models.Payment
	if err := config.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy lịch sử thanh toán"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    len(payments),
		"payments": payments,
	})
}

// GET /api/payment/packages (gói VIP)
func GetVIPPackages(c *gin.Context) {
	packages := []gin.H{
		{
			"id":          "vip_30",
			"name":        "VIP 1 tháng",
			"duration":    30,
			"price":       50000,
			"description": "Truy cập tất cả podcast VIP trong 30 ngày",
		},
		{
			"id":          "vip_90",
			"name":        "VIP 3 tháng",
			"duration":    90,
			"price":       120000,
			"description": "Truy cập tất cả podcast VIP trong 90 ngày (Tiết kiệm 20%)",
			"discount":    "20%",
		},
		{
			"id":          "vip_365",
			"name":        "VIP 1 năm",
			"duration":    365,
			"price":       400000,
			"description": "Truy cập tất cả podcast VIP trong 365 ngày (Tiết kiệm 33%)",
			"discount":    "33%",
		},
	}

	c.JSON(http.StatusOK, gin.H{"packages": packages})
}
