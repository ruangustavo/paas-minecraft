package server

import (
	"net/http"
	"paas-minecraft/internal/modules/server/service"

	"github.com/labstack/echo/v4"
)

type ServerController struct {
	dockerService *service.DockerService
}

func NewServerController(dockerService *service.DockerService) *ServerController {
	return &ServerController{
		dockerService: dockerService,
	}
}

func (sc *ServerController) RegisterRoutes(e *echo.Echo) {
	e.POST("/servers", sc.CreateServer)
}

type Server struct {
	Name string `json:"name" validate:"required"`
}

func (sc *ServerController) CreateServer(c echo.Context) error {
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

	if err := sc.dockerService.Create(server.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusCreated)
}
