package controllers

import (
	"net/http"
	"os"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/clerkinc/clerk-sdk-go/clerk"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var clerkClient clerk.Client

// ✅ Khởi tạo Clerk client lấy từ ENV
func InitClerk() {
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		panic("CLERK_SECRET_KEY không tồn tại trong ENV")
	}

	client, err := clerk.NewClient(secretKey)
	if err != nil {
		panic("Clerk init failed: " + err.Error())
	}

	clerkClient = client
}

type ClerkLoginInput struct {
	ClerkToken string `json:"clerk_token" binding:"required"`
}

func ClerkLogin(c *gin.Context) {
	var input ClerkLoginInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu clerk_token"})
		return
	}

	// ✅ verify session token
	session, err := clerkClient.Sessions().Verify(input.ClerkToken)
	if err != nil || session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
		return
	}

	userID := session.UserID

	// ✅ Lấy user
	clerkUser, err := clerkClient.Users().Read(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc thông tin user từ Clerk"})
		return
	}

	// LẤY EMAIL AN TOÀN
	email := ""
	if clerkUser.EmailAddress != nil {
		email = *clerkUser.EmailAddress
	}

	//LẤY TÊN AN TOÀN (pointer string)
	firstName := ""
	lastName := ""

	if clerkUser.FirstName != nil {
		firstName = *clerkUser.FirstName
	}

	if clerkUser.LastName != nil {
		lastName = *clerkUser.LastName
	}

	fullName := firstName + " " + lastName
	if fullName == " " {
		fullName = "Người dùng"
	}

	// ----------------------
	// ✅ Avatar (pointer string)
	// ----------------------
	avatar := ""
	if clerkUser.ProfileImageURL != nil {
		avatar = *clerkUser.ProfileImageURL
	}

	// ✅ Kiểm tra user DB
	var user models.NguoiDung
	result := config.DB.Where("email = ?", email).First(&user)

	if result.Error != nil {
		// User mới
		user = models.NguoiDung{
			ID:       uuid.New().String(),
			Email:    email,
			MatKhau:  "clerk",
			HoTen:    fullName,
			VaiTro:   "user",
			Avatar:   avatar,
			KichHoat: true,
		}

		if err := config.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo user mới"})
			return
		}
	}

	// ✅ Tạo JWT
	token, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo JWT"})
		return
	}

	// ✅ Trả về client
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"ho_ten":   user.HoTen,
			"vai_tro":  user.VaiTro,
			"avatar":   user.Avatar,
			"ngay_tao": user.NgayTao,
		},
	})
}
