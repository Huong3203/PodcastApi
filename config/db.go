package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Huong3203/APIPodcast/models"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Load biến môi trường từ file .env
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Không tìm thấy file .env, dùng biến môi trường hệ thống.")
	}
}

func ConnectDB() {
	LoadEnv()

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	// ✅ DSN chuẩn PostgreSQL cho Render
	// dsn := fmt.Sprintf(
	// 	"host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Asia/Ho_Chi_Minh",
	// 	host, user, password, dbname, port,
	// )

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=UTC",
		host, user, password, dbname, port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Kết nối cơ sở dữ liệu thất bại: %v", err)
	}

	DB = db

	// ✅ Tự động migrate các model
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

	fmt.Println("✅ Đã kết nối PostgreSQL & migrate thành công!")

	// ✅ Cấu hình connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ Không thể lấy đối tượng *sql.DB: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
}
