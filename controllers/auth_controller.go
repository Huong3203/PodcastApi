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

// ==========================
// Local Register
// ==========================
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

	var existing models.NguoiDung
	if err := config.DB.Where("email = ?", input.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email đã được sử dụng"})
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(input.MatKhau), bcrypt.DefaultCost)
	user := models.NguoiDung{
		ID:       uuid.New().String(),
		Email:    input.Email,
		MatKhau:  string(hashed),
		HoTen:    input.HoTen,
		VaiTro:   "user",
		KichHoat: true,
		Provider: "local",
	}

	config.DB.Create(&user)
	user.MatKhau = ""
	c.JSON(http.StatusCreated, user)
}

// ==========================
// Local Login
// ==========================
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

	token, _ := utils.GenerateToken(user.ID, user.VaiTro, user.Provider)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}
