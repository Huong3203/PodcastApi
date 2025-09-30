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
	err := godotenv.Load() //Load để đọc file .env, không truyền vào gì thì nó tự động mặc định kiếm .env để gán vào biến môi trường và nếu file .env bị lỗi thì trả về false nghĩa là ko phải nil
	if err != nil {
		log.Fatal("❌ Load .env failed") //Dùng để ghi log dạng lỗi nghiêm trọng rồi thoát khỏi chương trình ngay lập tức (giống như panic()).
	}
}

func ConnectDB() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	var err error
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{}) //mở kết nối đến DB qua driver mysql.
	// gor.Open Hàm của GORM để mở kết nối CSDL. Trả về *gorm.DB và error.
	// &gorm.Config{} Cấu hình thêm cho GORM (bạn có thể để trống hoặc tùy chỉnh sâu hơn).
	if err != nil {
		log.Fatal("❌ Failed to connect DB:", err)
	}
	DB = db
	// ✅ Auto migrate các bảng
	err = DB.AutoMigrate(
		&models.NguoiDung{},
		&models.TaiLieu{},
		&models.Podcast{},
		&models.DanhMuc{},
	)
	if err != nil {
		log.Fatal("❌ Auto migration failed:", err)
	}

	fmt.Println("✅ Connected to DB and Migrated!")

	// Cài đặt trước của Connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Không thể lấy dữ được đối tượng cơ sở dữ liệu: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)  // số kết nối rảnh tối đa (không dùng nhưng giữ lại)
	sqlDB.SetMaxOpenConns(100) // tổng số kết nối mở tối đa
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

}
