package auth

import (
	"net/http"
	"paas-minecraft/internal/modules/auth/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	authService *service.AuthService
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Name     string `json:"name" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

type UserDTO struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// @Summary		Register a new user
// @Description	Register a new user account with email, password, and name
// @Tags			auth
// @Param			request	body		RegisterRequest		true	"User registration data"
// @Success		201		{object}	AuthResponse		"User registered successfully"
// @Failure		400		{object}	map[string]string	"Invalid request data"
// @Router			/auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	user, err := h.authService.Register(req.Email, req.Password, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	token, err := h.authService.GenerateToken(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User: UserDTO{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
	})
}

// @Summary		User login
// @Description	Authenticate user with email and password, returns JWT token
// @Tags			auth
// @Param			request	body		LoginRequest		true	"User login credentials"
// @Success		200		{object}	AuthResponse		"Login successful"
// @Failure		400		{object}	map[string]string	"Invalid request data"
// @Failure		401		{object}	map[string]string	"Invalid credentials"
// @Router			/auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.Validate(req); err != nil {
		return err
	}

	token, user, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User: UserDTO{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
	})
}

func (h *AuthHandler) RegisterRoutes(e *echo.Echo) {
	auth := e.Group("/auth")
	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
}
