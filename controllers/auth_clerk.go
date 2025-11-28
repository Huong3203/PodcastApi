package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/middleware"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"
	"github.com/gin-gonic/gin"
)

// ✅ Input nhận từ frontend
type ClerkLoginInput struct {
	SessionID string `json:"session_id" binding:"required"` // Clerk session ID
	Email     string `json:"email" binding:"required"`      // Email từ Clerk
	HoTen     string `json:"ho_ten"`                        // Optional
	Avatar    string `json:"avatar"`                        // Optional
}

func LoginWithClerk(c *gin.Context) {
	fmt.Println("🔵 [LoginWithClerk] Bắt đầu đăng nhập Clerk")

	var input ClerkLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fmt.Println("❌ Lỗi bind JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu session_id hoặc email"})
		return
	}

	fmt.Printf("📩 Received data: session_id=%s, email=%s\n", input.SessionID, input.Email)

	// 1. ✅ Verify session với Clerk
	sess, err := middleware.ClerkClient.Sessions().Read(input.SessionID)
	if err != nil {
		fmt.Println("❌ Session không hợp lệ:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session không hợp lệ hoặc đã hết hạn"})
		return
	}

	fmt.Printf("✅ Session verified: UserID=%s\n", sess.UserID)

	// 2. ✅ Lấy thông tin user từ Clerk (để verify)
	clerkUser, err := middleware.ClerkClient.Users().Read(sess.UserID)
	if err != nil {
		fmt.Println("❌ Không lấy được user từ Clerk:", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Không lấy được thông tin user"})
		return
	}

	// 3. ✅ Verify email khớp với Clerk
	clerkEmail := ""
	if len(clerkUser.EmailAddresses) > 0 {
		clerkEmail = clerkUser.EmailAddresses[0].EmailAddress
	}

	if clerkEmail != input.Email {
		fmt.Printf("⚠️ Email mismatch: Clerk=%s, Input=%s\n", clerkEmail, input.Email)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email không khớp với Clerk"})
		return
	}

	// 4. ✅ Dùng thông tin từ Clerk (ưu tiên) hoặc từ input
	hoTen := input.HoTen
	if hoTen == "" {
		if clerkUser.FirstName != nil {
			hoTen = *clerkUser.FirstName
		}
		if clerkUser.LastName != nil {
			if hoTen != "" {
				hoTen += " "
			}
			hoTen += *clerkUser.LastName
		}
	}

	avatar := input.Avatar
	if avatar == "" {
		avatar = clerkUser.ProfileImageURL
	}

	// 5. ✅ Kiểm tra user trong DB
	var user models.NguoiDung
	err = config.DB.Where("email = ?", input.Email).First(&user).Error

	if err != nil {
		// User chưa tồn tại → tạo mới
		if IsRecordNotFound(err) {
			fmt.Println("ℹ️ User chưa tồn tại → tạo mới")

			user = models.NguoiDung{
				ID:       clerkUser.ID, // Dùng Clerk ID
				Email:    input.Email,
				HoTen:    hoTen,
				Avatar:   avatar,
				VaiTro:   "user",
				KichHoat: true,
				Provider: "clerk",
				NgayTao:  time.Now(),
			}

			if err := config.DB.Create(&user).Error; err != nil {
				fmt.Println("❌ Không thể tạo user:", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":  "Không thể tạo user mới",
					"detail": err.Error(),
				})
				return
			}

			fmt.Println("✅ User mới đã được tạo:", user.ID)
		} else {
			// Lỗi DB khác
			fmt.Println("❌ Lỗi DB:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
			return
		}
	} else {
		// User đã tồn tại → update thông tin
		fmt.Println("ℹ️ User đã tồn tại → cập nhật thông tin")

		user.HoTen = hoTen
		user.Avatar = avatar
		user.Provider = "clerk"

		if err := config.DB.Save(&user).Error; err != nil {
			fmt.Println("❌ Không thể cập nhật user:", err)
		} else {
			fmt.Println("✅ User đã được cập nhật")
		}
	}

	// 6. ✅ Tạo JWT token của hệ thống
	token, err := utils.GenerateToken(user.ID, user.VaiTro)
	if err != nil {
		fmt.Println("❌ Không thể tạo JWT token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo token"})
		return
	}

	fmt.Println("✅ JWT token đã được tạo:", token[:20]+"...")

	// 7. ✅ Trả về response
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":      user.ID,
			"email":   user.Email,
			"ho_ten":  user.HoTen,
			"avatar":  user.Avatar,
			"vai_tro": user.VaiTro,
		},
		"token": token,
	})

	fmt.Println("✅ [LoginWithClerk] Hoàn tất")
}

// Helper kiểm tra RecordNotFound
func IsRecordNotFound(err error) bool {
	// GORM v2
	return err != nil && err.Error() == "record not found"
}
