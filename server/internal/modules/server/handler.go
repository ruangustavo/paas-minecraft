package server

import (
	"fmt"
	"net/http"
	"paas-minecraft/internal/modules/server/repository"
	"paas-minecraft/internal/modules/server/service"

	"github.com/labstack/echo/v4"
)

type ServerHandler struct {
	dockerService     *service.DockerService
	infraredService   *service.InfraredService
	cloudflareService *service.CloudflareService
	serverRepo        *repository.ServerRepository
}

func NewServerHandler(dockerService *service.DockerService, infraredService *service.InfraredService, serverRepo *repository.ServerRepository, cloudflareService *service.CloudflareService) *ServerHandler {
	return &ServerHandler{
		dockerService:     dockerService,
		infraredService:   infraredService,
		cloudflareService: cloudflareService,
		serverRepo:        serverRepo,
	}
}

func (sc *ServerHandler) RegisterRoutes(e *echo.Echo) {
	e.POST("/servers", sc.CreateServer)
}

type Server struct {
	Name string `json:"name" validate:"required,alphanum"`
}

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

	var (
		dockerCreated bool
		proxyCreated  bool
		dnsCreated    bool
		dbCreated     bool
		subdomain     string
	)

	defer func() {
		if !dbCreated {
			if dnsCreated {
				sc.cloudflareService.DeleteDNSRecord(subdomain)
			}
			if proxyCreated {
				sc.infraredService.DeleteProxyConfig(server.Name)
			}
			if dockerCreated {
				sc.dockerService.Delete(server.Name)
			}
		}
	}()

	if err := sc.dockerService.Create(server.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create container: %v", err),
		})
	}
	dockerCreated = true

	const internalPort = 25565
	subdomain = fmt.Sprintf("%s.%s", server.Name, sc.cloudflareService.BaseDomain())

	if err := sc.infraredService.CreateProxyConfig(server.Name, subdomain); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create proxy config: %v", err),
		})
	}
	proxyCreated = true

	if err := sc.cloudflareService.CreateDNSRecord(subdomain); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create DNS record: %v", err),
		})
	}
	dnsCreated = true

	createdServer, err := sc.serverRepo.Create(server.Name, internalPort, subdomain)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save server: %v", err),
		})
	}
	dbCreated = true

	return c.JSON(http.StatusCreated, map[string]string{
		"message":   "Server created successfully",
		"id":        createdServer.ID.String(),
		"name":      createdServer.Name,
		"subdomain": createdServer.Subdomain,
	})
}
