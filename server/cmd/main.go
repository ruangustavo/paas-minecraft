package main

import (
	"log"
	"net/http"
	"os"
	"paas-minecraft/internal/modules/server"
	"paas-minecraft/internal/modules/server/repository"
	"paas-minecraft/internal/modules/server/service"
	"paas-minecraft/internal/shared/database"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Warning: .env file not found or could not be loaded:", err)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalln("Failed to connect to database:", err)
	}

	e := echo.New()

	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	dockerService := service.NewDockerService()
	infraredService := service.NewInfraredService("./config/infrared/proxies", dockerService)

	cfToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	cfZone := os.Getenv("CLOUDFLARE_ZONE_ID")
	serverIP := os.Getenv("SERVER_PUBLIC_IP")
	baseDomain := os.Getenv("BASE_DOMAIN")

	if cfToken == "" || cfZone == "" || serverIP == "" || baseDomain == "" {
		log.Fatalln("One or more required environment variables (CLOUDFLARE_API_TOKEN, CLOUDFLARE_ZONE_ID, SERVER_PUBLIC_IP, BASE_DOMAIN) are missing")
	}

	cloudflareService := service.NewCloudflareService(cfToken, cfZone, serverIP, baseDomain)
	serverRepo := repository.NewServerRepository()

	serverHandler := server.NewServerHandler(dockerService, infraredService, serverRepo, cloudflareService)
	serverHandler.RegisterRoutes(e)

	e.Logger.Fatal(e.Start(":8080"))
}
