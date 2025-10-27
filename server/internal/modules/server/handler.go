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

func (sc *ServerHandler) RegisterRoutes(e *echo.Echo, middlewares ...echo.MiddlewareFunc) {
	servers := e.Group("/servers", middlewares...)
	servers.POST("", sc.CreateServer)
	servers.GET("", sc.ListServers)
	servers.GET("/:id", sc.GetServer)
	servers.POST("/:id/start", sc.StartServer)
	servers.POST("/:id/stop", sc.StopServer)
	servers.DELETE("/:id", sc.DeleteServer)
}

type Server struct {
	Name string `json:"name" validate:"required,alphanum"`
}

// @Summary		Create a new Minecraft server
// @Description	Create a new Minecraft server with Docker container, proxy configuration, and DNS record
// @Tags			servers
// @Param			server	body		Server				true	"Server creation data"
// @Success		201		{object}	map[string]string	"Server created successfully"
// @Failure		400		{object}	map[string]string	"Invalid request data"
// @Failure		500		{object}	map[string]string	"Internal server error"
// @Router			/servers [post]
// @Security		BearerAuth
func (sc *ServerHandler) CreateServer(c echo.Context) error {
	server := new(Server)

	if err := c.Bind(server); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request data")
	}

	if err := c.Validate(server); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	existingServer, err := sc.serverRepo.FindByName(server.Name)
	if err == nil && existingServer != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Container with this name already exists")
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
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create container: %v", err))
	}
	dockerCreated = true

	const internalPort = 25565
	subdomain = fmt.Sprintf("%s.%s", server.Name, sc.cloudflareService.BaseDomain())

	if err := sc.infraredService.CreateProxyConfig(server.Name, subdomain); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create proxy config: %v", err))
	}
	proxyCreated = true

	if err := sc.cloudflareService.CreateDNSRecord(subdomain); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to create DNS record: %v", err))
	}
	dnsCreated = true

	createdServer, err := sc.serverRepo.Create(server.Name, internalPort, subdomain)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to save server: %v", err))
	}
	dbCreated = true

	return c.JSON(http.StatusCreated, map[string]string{
		"message":   "Server created successfully",
		"id":        createdServer.ID.String(),
		"name":      createdServer.Name,
		"subdomain": createdServer.Subdomain,
	})
}

// @Summary		List all Minecraft servers
// @Description	Get a list of all configured Minecraft servers
// @Tags			servers
// @Success		200	{array}		model.Server	"List of servers"
// @Failure		500	{object}	map[string]string	"Internal server error"
// @Router			/servers [get]
// @Security		BearerAuth
func (sc *ServerHandler) ListServers(c echo.Context) error {
	servers, err := sc.serverRepo.FindAll()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to list servers: %v", err))
	}

	return c.JSON(http.StatusOK, servers)
}

// @Summary		Get server details
// @Description	Get details of a specific Minecraft server by ID
// @Tags			servers
// @Success		200	{object}	model.Server		"Server details"
// @Failure		400	{object}	map[string]string	"Invalid server ID"
// @Failure		404	{object}	map[string]string	"Server not found"
// @Router			/servers/{id} [get]
// @Security		BearerAuth
func (sc *ServerHandler) GetServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid server ID")
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Server not found")
	}

	return c.JSON(http.StatusOK, server)
}

// @Summary		Start a Minecraft server
// @Description	Start a stopped Minecraft server container
// @Tags			servers
// @Success		200	{object}	map[string]string	"Server started successfully"
// @Failure		400	{object}	map[string]string	"Invalid server ID"
// @Failure		404	{object}	map[string]string	"Server not found"
// @Failure		500	{object}	map[string]string	"Internal server error"
// @Router			/servers/{id}/start [post]
// @Security		BearerAuth
func (sc *ServerHandler) StartServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid server ID")
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Server not found")
	}

	if server.Status == "running" {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Server is already running",
			"status":  server.Status,
		})
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "starting"); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to update server status: %v", err))
	}

	if err := sc.dockerService.Start(server.Name); err != nil {
		sc.serverRepo.UpdateStatus(serverID, "stopped")
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to start container: %v", err))
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "running"); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to update server status: %v", err))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Server started successfully",
		"status":  "running",
	})
}

// @Summary		Stop a Minecraft server
// @Description	Stop a running Minecraft server container
// @Tags			servers
// @Success		200	{object}	map[string]string	"Server stopped successfully"
// @Failure		400	{object}	map[string]string	"Invalid server ID"
// @Failure		404	{object}	map[string]string	"Server not found"
// @Failure		500	{object}	map[string]string	"Internal server error"
// @Router			/servers/{id}/stop [post]
// @Security		BearerAuth
func (sc *ServerHandler) StopServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid server ID")
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Server not found")
	}

	if server.Status == "stopped" {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "Server is already stopped",
			"status":  server.Status,
		})
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "stopping"); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to update server status: %v", err))
	}

	if err := sc.dockerService.Stop(server.Name); err != nil {
		sc.serverRepo.UpdateStatus(serverID, "running")
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to stop container: %v", err))
	}

	if err := sc.serverRepo.UpdateStatus(serverID, "stopped"); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to update server status: %v", err))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Server stopped successfully",
		"status":  "stopped",
	})
}

// @Summary		Delete a Minecraft server
// @Description	Delete a Minecraft server and all its associated resources (container, proxy config, DNS record)
// @Tags			servers
// @Success		200	{object}	map[string]string	"Server deleted successfully"
// @Failure		400	{object}	map[string]string	"Invalid server ID"
// @Failure		404	{object}	map[string]string	"Server not found"
// @Failure		500	{object}	map[string]string	"Internal server error"
// @Router			/servers/{id} [delete]
// @Security		BearerAuth
func (sc *ServerHandler) DeleteServer(c echo.Context) error {
	id := c.Param("id")
	serverID, err := uuid.Parse(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid server ID")
	}

	server, err := sc.serverRepo.FindByID(serverID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Server not found")
	}

	if err := sc.dockerService.Delete(server.Name); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to delete container: %v", err))
	}

	if err := sc.infraredService.DeleteProxyConfig(server.Name); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to delete proxy config: %v", err))
	}

	if err := sc.cloudflareService.DeleteDNSRecord(server.Subdomain); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to delete DNS record: %v", err))
	}

	if err := sc.serverRepo.Delete(serverID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("failed to delete server record: %v", err))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Server deleted successfully",
	})
}
