package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jtsang4/nettune/internal/server/service"
	"github.com/jtsang4/nettune/internal/shared/types"
)

// ProfileHandler handles profile-related HTTP endpoints
type ProfileHandler struct {
	profileService *service.ProfileService
}

// NewProfileHandler creates a new ProfileHandler
func NewProfileHandler(profileService *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// List handles GET /profiles
func (h *ProfileHandler) List(c *gin.Context) {
	profiles, err := h.profileService.List()
	if err != nil {
		internalError(c, err.Error())
		return
	}

	success(c, gin.H{
		"profiles": profiles,
	})
}

// Get handles GET /profiles/:id
func (h *ProfileHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		badRequest(c, "profile id is required")
		return
	}

	profile, err := h.profileService.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrProfileNotFound) {
			notFound(c, "profile not found")
			return
		}
		internalError(c, err.Error())
		return
	}

	success(c, profile)
}

func notFound(c *gin.Context, message string) {
	c.JSON(404, gin.H{"success": false, "error": gin.H{"code": "NOT_FOUND", "message": message}})
}
