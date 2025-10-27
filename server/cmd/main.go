package main

import (
	"log"
	"net/http"
	"os"
	"paas-minecraft/internal/modules/auth"
	authMiddleware "paas-minecraft/internal/modules/auth/middleware"
	authRepo "paas-minecraft/internal/modules/auth/repository"
	authService "paas-minecraft/internal/modules/auth/service"
	"paas-minecraft/internal/modules/server"
	"paas-minecraft/internal/modules/server/repository"
	"paas-minecraft/internal/modules/server/service"
	"paas-minecraft/internal/shared/database"

	_ "paas-minecraft/docs"

	echoSwagger "github.com/swaggo/echo-swagger"

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

// @title           Paas Minecraft
// @version         1.0.0
// @description     Minecraft PaaS API
// @host            localhost:8080
// @BasePath        /
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
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	dockerService := service.NewDockerService()
	infraredService := service.NewInfraredService("./config/infrared/proxies", dockerService)

	cfToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	cfZone := os.Getenv("CLOUDFLARE_ZONE_ID")
	serverIP := os.Getenv("SERVER_PUBLIC_IP")
	baseDomain := os.Getenv("BASE_DOMAIN")

	cloudflareService := service.NewCloudflareService(cfToken, cfZone, serverIP, baseDomain)
	serverRepo := repository.NewServerRepository()

	userRepo := authRepo.NewUserRepository()
	authSvc := authService.NewAuthService(userRepo)
	authHandler := auth.NewAuthHandler(authSvc)
	authHandler.RegisterRoutes(e)

	jwtMiddleware := authMiddleware.JWTAuth(authSvc)
	serverHandler := server.NewServerHandler(dockerService, infraredService, serverRepo, cloudflareService)
	serverHandler.RegisterRoutes(e, jwtMiddleware)

	e.Logger.Fatal(e.Start(":8080"))
}
