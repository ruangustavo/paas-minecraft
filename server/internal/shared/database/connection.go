package database

import (
	"fmt"
	"log"
	"os"
	"paas-minecraft/internal/modules/server/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	host := getEnv("DB_HOST", "localhost")
	user := getEnv("DB_USER", "docker")
	password := getEnv("DB_PASSWORD", "qwe123")
	dbname := getEnv("DB_NAME", "paas_minecraft")
	port := getEnv("DB_PORT", "5432")
	sslmode := getEnv("DB_SSLMODE", "disable")
	timezone := getEnv("DB_TIMEZONE", "UTC")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		host, user, password, dbname, port, sslmode, timezone)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database connected and migrated successfully")
	return nil
}

func runMigrations() error {
	return DB.AutoMigrate(&model.Server{})
}

func GetDB() *gorm.DB {
	return DB
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
