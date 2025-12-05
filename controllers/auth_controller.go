package controllers

import (
	"net/http"

	"github.com/Huong3203/APIPodcast/config"
	"github.com/Huong3203/APIPodcast/models"
	"github.com/Huong3203/APIPodcast/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	Email   string `json:"email" binding:"required,email"`
	MatKhau string `json:"mat_khau" binding:"required,min=6"`
	HoTen   string `json:"ho_ten" binding:"required"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Kiểm tra email + provider = local
	var existing models.NguoiDung
	if err := config.DB.Where("email = ? AND provider = ?", input.Email, "local").First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã được sử dụng"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.MatKhau), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mã hoá mật khẩu"})
		return
	}

	newUser := models.NguoiDung{
		ID:       uuid.New().String(),
		Email:    input.Email,
		MatKhau:  string(hashedPassword),
		HoTen:    input.HoTen,
		VaiTro:   "user",
		KichHoat: true,
		Provider: "local",
	}

	if err := config.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tạo người dùng"})
		return
	}

	token, _ := utils.GenerateToken(newUser.ID, newUser.VaiTro)
	newUser.MatKhau = ""

	c.JSON(http.StatusCreated, gin.H{
		"user":     newUser,
		"token":    token,
		"provider": "local",
	})
}

type LoginInput struct {
	Email   string `json:"email" binding:"required,email"`
	MatKhau string `json:"mat_khau" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.NguoiDung
	if err := config.DB.Where("email = ? AND provider = ?", input.Email, "local").First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.MatKhau), []byte(input.MatKhau)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
		return
	}

	token, _ := utils.GenerateToken(user.ID, user.VaiTro)
	user.MatKhau = ""

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"token":    token,
		"provider": "local",
	})
}

func RegisterAdmin(c *gin.Context) {
	type AdminRegisterInput struct {
		Email   string `json:"email" binding:"required,email"`
		MatKhau string `json:"mat_khau" binding:"required,min=6"`
		HoTen   string `json:"ho_ten" binding:"required"`
	}

	var input AdminRegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1️⃣ Kiểm tra đã có admin nào chưa
	var adminCount int64
	config.DB.Model(&models.NguoiDung{}).Where("vai_tro = ?", "admin").Count(&adminCount)

	isFirstAdmin := adminCount == 0

	// 2️⃣ Nếu không phải admin đầu tiên → cần token admin
	if !isFirstAdmin {
		role, exists := c.Get("vai_tro")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ admin mới có quyền truy cập"})
			return
		}
	}

	// 3️⃣ Kiểm tra email
	var existing models.NguoiDung
	if err := config.DB.Where("email = ? AND provider = ?", input.Email, "local").First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã được sử dụng"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.MatKhau), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mã hoá mật khẩu"})
		return
	}

	newAdmin := models.NguoiDung{
		ID:       uuid.New().String(),
		Email:    input.Email,
		MatKhau:  string(hashedPassword),
		HoTen:    input.HoTen,
		VaiTro:   "admin",
		KichHoat: true,
		Provider: "local",
	}

	if err := config.DB.Create(&newAdmin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tạo admin"})
		return
	}

	token, _ := utils.GenerateToken(newAdmin.ID, newAdmin.VaiTro)
	newAdmin.MatKhau = ""

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Tạo admin thành công",
		"first_admin": isFirstAdmin,
		"admin":       newAdmin,
		"token":       token,
	})
}
