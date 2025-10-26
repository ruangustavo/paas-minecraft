package server

import (
	"net/http"
	"paas-minecraft/internal/modules/server/repository"
	"paas-minecraft/internal/modules/server/service"

	"github.com/labstack/echo/v4"
)

type ServerHandler struct {
	dockerService *service.DockerService
	serverRepo    *repository.ServerRepository
}

func NewServerHandler(dockerService *service.DockerService, serverRepo *repository.ServerRepository) *ServerHandler {
	return &ServerHandler{
		dockerService: dockerService,
		serverRepo:    serverRepo,
	}
}

func (sc *ServerHandler) RegisterRoutes(e *echo.Echo) {
	e.POST("/servers", sc.CreateServer)
}

type Server struct {
	Name string `json:"name" validate:"required,alphanum"`
}

// TODO: think how to know which port the container is running? maybe persisting is not a option because the container can be down, idk...
func (sc *ServerHandler) CreateServer(c echo.Context) error {
	server := new(Server)

	if err := c.Bind(server); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request data",
		})
	}

	if err := c.Validate(server); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	existingServer, err := sc.serverRepo.FindByName(server.Name)

	if err == nil && existingServer != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Container with this name already exists",
		})
	}

	if err := sc.dockerService.Create(server.Name); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	createdServer, err := sc.serverRepo.Create(server.Name)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Server created successfully",
		"id":      createdServer.ID.String(),
		"name":    createdServer.Name,
	})
}
