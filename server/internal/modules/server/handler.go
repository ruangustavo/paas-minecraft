package server

import (
	"fmt"
	"net/http"
	"paas-minecraft/internal/modules/server/repository"
	"paas-minecraft/internal/modules/server/service"

	"github.com/google/uuid"
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
	e.GET("/servers", sc.ListServers)
	e.GET("/servers/:id", sc.GetServer)
	e.POST("/servers/:id/start", sc.StartServer)
	e.POST("/servers/:id/stop", sc.StopServer)
	e.DELETE("/servers/:id", sc.DeleteServer)
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

func (sc *ServerHandler) ListServers(c echo.Context) error {
	servers, err := sc.serverRepo.FindAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to list servers: %v", err),
		})
	}

	return c.JSON(http.StatusOK, servers)
}

func (sc *ServerHandler) GetServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid server ID",
		})
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
		})
	}

	return c.JSON(http.StatusOK, server)
}

func (sc *ServerHandler) StartServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid server ID",
		})
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
		})
	}

	if server.Status == "running" {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Server is already running",
			"status":  server.Status,
		})
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "starting"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to update server status: %v", err),
		})
	}

	if err := sc.dockerService.Start(server.Name); err != nil {
		sc.serverRepo.UpdateStatus(serverID, "stopped")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to start container: %v", err),
		})
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "running"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to update server status: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Server started successfully",
		"status":  "running",
	})
}

func (sc *ServerHandler) StopServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid server ID",
		})
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
		})
	}

	if server.Status == "stopped" {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Server is already stopped",
			"status":  server.Status,
		})
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "stopping"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to update server status: %v", err),
		})
	}

	if err := sc.dockerService.Stop(server.Name); err != nil {
		sc.serverRepo.UpdateStatus(serverID, "running")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to stop container: %v", err),
		})
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "stopped"); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to update server status: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Server stopped successfully",
		"status":  "stopped",
	})
}

func (sc *ServerHandler) DeleteServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := parseUUID(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid server ID",
		})
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
		})
	}

	if err := sc.dockerService.Delete(server.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to delete container: %v", err),
		})
	}

	if err := sc.infraredService.DeleteProxyConfig(server.Name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to delete proxy config: %v", err),
		})
	}

	if err := sc.cloudflareService.DeleteDNSRecord(server.Subdomain); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to delete DNS record: %v", err),
		})
	}

	if err := sc.serverRepo.Delete(serverID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to delete server record: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Server deleted successfully",
	})
}
