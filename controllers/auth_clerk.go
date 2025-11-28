package controllers

import (
	"fmt"
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

// Input khi client gửi session_id từ Clerk/Google
type ClerkLoginInput struct {
	SessionID string `json:"session_id" binding:"required"`
}

// API đăng nhập với Clerk
func LoginWithClerk(c *gin.Context) {

	fmt.Println("🔵 [LoginWithClerk] Bắt đầu xử lý đăng nhập Clerk")

	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fmt.Println("❌ Lỗi bind JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id bắt buộc"})
		return
	}

	fmt.Println("📥 session_id nhận từ client:", input.SessionID)

	// 1. Lấy session từ Clerk
	fmt.Println("🔎 Đang gọi Clerk API để lấy session...")
	sess, err := middleware.ClerkClient.Sessions().Read(input.SessionID)
	if err != nil {
		fmt.Println("❌ Clerk trả về lỗi khi lấy session:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session không hợp lệ"})
		return
	}

	fmt.Println("✅ Session hợp lệ:", sess.ID, " | UserID:", sess.UserID)

	// 2. Lấy user từ session
	fmt.Println("🔎 Đang lấy thông tin user từ Clerk...")
	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		fmt.Println("❌ Lỗi Clerk khi lấy user:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được user từ Clerk"})
		return
	}

	fmt.Println("✅ Lấy user Clerk thành công! UserID:", clerkUser.ID)

	// 3. Lấy email
	email := ""
	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}
	fmt.Println("📧 Email:", email)

	// 4. Lấy tên đầy đủ và avatar
	hoTen := ""
	if clerkUser.FirstName != nil && clerkUser.LastName != nil {
		hoTen = *clerkUser.FirstName + " " + *clerkUser.LastName
	}
	fmt.Println("👤 Họ tên:", hoTen)

	avatar := ""
	if clerkUser.ProfileImageURL != "" {
		avatar = clerkUser.ProfileImageURL
	}
	fmt.Println("🖼 Avatar URL:", avatar)

	// 5. Lưu user nếu chưa tồn tại
	fmt.Println("🔎 Kiểm tra user trong database...")

	var user models.NguoiDung
	result := config.DB.Where("id = ?", clerkUser.ID).First(&user)

	if result.Error != nil {
		fmt.Println("ℹ User chưa tồn tại → Tạo mới trong DB")

		user = models.NguoiDung{
			ID:       clerkUser.ID,
			Email:    email,
			VaiTro:   "user",
			KichHoat: true,
			Provider: "clerk",
			HoTen:    hoTen,
			Avatar:   avatar,
		}

		if err := config.DB.Create(&user).Error; err != nil {
			fmt.Println("❌ Lỗi tạo user mới:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo mới user"})
			return
		}

		fmt.Println("✅ Đã tạo user mới:", user.ID)
	} else {
		fmt.Println("✅ User đã tồn tại trong DB:", user.ID)
	}

	// 6. Tạo token JWT
	fmt.Println("🔐 Đang tạo JWT token...")
	token, err := utils.GenerateToken(user.ID, user.VaiTro, "clerk")
	if err != nil {
		fmt.Println("❌ Lỗi tạo token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tạo được token"})
		return
	}

	fmt.Println("✅ Token đã tạo thành công")

	// 7. Trả về client
	fmt.Println("🚀 Đăng nhập Clerk hoàn tất — trả dữ liệu cho client")

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":      user.ID,
			"email":   user.Email,
			"ho_ten":  user.HoTen,
			"vai_tro": user.VaiTro,
			"avatar":  user.Avatar,
		},
		"token": token,
	})
}
