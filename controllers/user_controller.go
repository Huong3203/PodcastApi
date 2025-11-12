package controllers

import (
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// 🔹 GET /api/users/profile

func GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")

	var user models.NguoiDung
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	user.MatKhau = ""
	c.JSON(http.StatusOK, user)
}

// 🔹 PUT /api/users/profile

type UpdateProfileInput struct {
	HoTen  string                `form:"ho_ten" binding:"required"`
	Email  string                `form:"email" binding:"required,email"`
	Avatar *multipart.FileHeader `form:"avatar"` // avatar có thể upload
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}

	var input UpdateProfileInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check email đã tồn tại
	var existingUser models.NguoiDung
	if err := config.DB.Where("email = ? AND id != ?", input.Email, userID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã được sử dụng"})
		return
	}

	updateData := map[string]interface{}{
		"ho_ten": input.HoTen,
		"email":  input.Email,
	}

	// Nếu upload avatar mới
	if input.Avatar != nil {
		avatarURL, err := utils.UploadAvatarToSupabase(input.Avatar, fmt.Sprintf("avatar_%s", userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể upload avatar"})
			return
		}
		updateData["avatar"] = avatarURL
	}

	// Update DB
	tx := config.DB.Model(&models.NguoiDung{}).Where("id = ?", userID).Updates(updateData)
	if tx.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cập nhật thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật thành công", "avatar": updateData["avatar"]})
}

// 🔹 POST /api/users/change-password

type ChangePasswordInput struct {
	MatKhauCu  string `json:"mat_khau_cu" binding:"required"`
	MatKhauMoi string `json:"mat_khau_moi" binding:"required,min=6"`
}

func ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")

	var input ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.NguoiDung
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.MatKhau), []byte(input.MatKhauCu)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu cũ không đúng"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.MatKhauMoi), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mã hoá mật khẩu"})
		return
	}

	if err := config.DB.Model(&user).Update("mat_khau", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đổi mật khẩu thất bại"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đổi mật khẩu thành công"})
}

// 🔹 GET /api/admin/users

func GetAllUsers(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền truy cập"})
		return
	}

	var users []models.NguoiDung
	if err := config.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách người dùng"})
		return
	}

	for i := range users {
		users[i].MatKhau = ""
	}

	c.JSON(http.StatusOK, gin.H{"total": len(users), "users": users})
}

// ==========================
// 🔹 PATCH /api/admin/users/:id/role
// ==========================
func UpdateUserRole(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền đổi vai trò"})
		return
	}

	id := c.Param("id")
	var input struct {
		VaiTro string `json:"vai_tro"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if input.VaiTro != "admin" && input.VaiTro != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vai trò không hợp lệ"})
		return
	}

	if err := config.DB.Model(&models.NguoiDung{}).Where("id = ?", id).Update("vai_tro", input.VaiTro).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vai trò"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật vai trò thành công"})
}

// ==========================
// 🔹 PATCH /api/admin/users/:id/toggle-active
// ==========================
func ToggleUserActivation(c *gin.Context) {
	role, _ := c.Get("vai_tro")
	if role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền cập nhật trạng thái"})
		return
	}

	id := c.Param("id")
	var user models.NguoiDung
	if err := config.DB.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	newStatus := !user.KichHoat
	if err := config.DB.Model(&user).Update("kich_hoat", newStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật trạng thái thành công", "kich_hoat": newStatus})
}
