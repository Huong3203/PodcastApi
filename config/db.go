package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("⚠️  Không tìm thấy file .env, dùng biến môi trường hệ thống.")
	}
}

func ConnectDB() {
	LoadEnv()

	// ✅ Cấu trúc DSN MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// ✅ Kết nối MySQL
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Kết nối cơ sở dữ liệu thất bại: %v", err)
	}
	DB = db

	// ✅ Tự động migrate các bảng
	err = DB.AutoMigrate(
		&models.NguoiDung{},
		&models.TaiLieu{},
		&models.Podcast{},
		&models.DanhMuc{},
		&models.DanhGia{},
	)
	if err != nil {
		log.Fatalf("❌ Auto migration thất bại: %v", err)
	}

	fmt.Println("✅ Đã kết nối MySQL thành công và AutoMigrate xong!")

	// ✅ Connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Không thể lấy đối tượng database: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
}
