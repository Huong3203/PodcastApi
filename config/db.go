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

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("❌ Load .env failed")
	}
}

func ConnectDB() {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
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
		&models.DanhGia{}, // nếu bạn có bảng rating
	)
	if err != nil {
		log.Fatal("❌ Auto migration failed:", err)
	}

	fmt.Println("✅ Connected to PostgreSQL DB and Migrated!")

	// Connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Không thể lấy đối tượng database: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
}
