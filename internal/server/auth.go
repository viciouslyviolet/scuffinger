package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"scuffinger/internal/auth"
	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
)

// AuthHandler handles /api/auth/* endpoints for the GitHub OAuth device flow.
type AuthHandler struct {
	clientID string
	log      *logging.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(clientID string, log *logging.Logger) *AuthHandler {
	return &AuthHandler{clientID: clientID, log: log}
}

// RegisterRoutes implements RouteRegistrar.
func (h *AuthHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/auth", h.StartDeviceFlow)
	}
}

// StartDeviceFlow initiates the GitHub OAuth device flow and returns
// the verification URI and user code the client must present to the user.
func (h *AuthHandler) StartDeviceFlow(c *gin.Context) {
	if h.clientID == "" {
		h.log.Error(i18n.Get(i18n.ErrAuthNoClientID))
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": i18n.Get(i18n.ErrAuthNoClientID),
		})
		return
	}

	scopes := []string{"repo", "read:org", "workflow"}

	h.log.Debug("Starting GitHub device flow")

	dcr, err := auth.RequestDeviceCode(h.clientID, scopes)
	if err != nil {
		h.log.Error(i18n.Get(i18n.ErrAuthDeviceCode), "error", err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   i18n.Get(i18n.ErrAuthDeviceCode),
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verification_uri": dcr.VerificationURI,
		"user_code":        dcr.UserCode,
		"device_code":      dcr.DeviceCode,
		"expires_in":       dcr.ExpiresIn,
		"interval":         dcr.Interval,
	})
}
