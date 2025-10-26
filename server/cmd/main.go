package main

import (
	"log"
	"net/http"
	"paas-minecraft/internal/modules/server"
	"paas-minecraft/internal/modules/server/repository"
	"paas-minecraft/internal/modules/server/service"
	"paas-minecraft/internal/shared/database"

	"github.com/go-playground/validator/v10"
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
	if err := database.InitDB(); err != nil {
		log.Fatalln("Failed to connect to database:", err)
	}

	e := echo.New()

	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	dockerService := service.NewDockerService()
	serverRepo := repository.NewServerRepository()

	serverHandler := server.NewServerHandler(dockerService, serverRepo)
	serverHandler.RegisterRoutes(e)

	e.Logger.Fatal(e.Start(":8080"))
}
