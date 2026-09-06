package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/middleware"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
	loginvisualapp "github.com/yyhuni/lunafox/server/internal/modules/loginvisual/application"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/loginvisual/dto"
)

const maxLoginVisualRequestBytes int64 = 26 << 20

type loginVisualService interface {
	IsDiscoverabilityUnlocked(context.Context, int) (bool, error)
	UnlockDiscoverability(context.Context, int) error
	GetSettings(context.Context, int) (domain.Settings, error)
	Upload(context.Context, int, []byte) (domain.Settings, error)
	Publish(context.Context, int) (domain.Settings, error)
	RestoreDefault(context.Context, int) (domain.Settings, error)
	Public(context.Context) (domain.PublicVisual, error)
	OpenPublished(context.Context, bool) (domain.Media, io.ReadCloser, error)
	OpenPreview(context.Context, int) (domain.Media, io.ReadCloser, error)
}

type LoginVisualHandler struct{ service loginVisualService }

func NewLoginVisualHandler(service loginVisualService) *LoginVisualHandler {
	if service == nil {
		panic("login visual handler service is required")
	}
	return &LoginVisualHandler{service: service}
}

// CheckDiscoverability handles GET /v1/settings/loginVisual:checkDiscoverability.
func (handler *LoginVisualHandler) CheckDiscoverability(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	unlocked, err := handler.service.IsDiscoverabilityUnlocked(c.Request.Context(), userID)
	if err != nil {
		handler.writeError(c, err, "check login visual discoverability")
		return
	}
	c.JSON(http.StatusOK, dto.DiscoverabilityResponse{Unlocked: unlocked})
}

// UnlockDiscoverability handles POST /v1/settings/loginVisual:unlockDiscoverability.
func (handler *LoginVisualHandler) UnlockDiscoverability(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	if err := handler.service.UnlockDiscoverability(c.Request.Context(), userID); err != nil {
		handler.writeError(c, err, "unlock login visual discoverability")
		return
	}
	c.JSON(http.StatusOK, dto.DiscoverabilityResponse{Unlocked: true})
}

// GetSettings handles GET /v1/settings/loginVisual.
func (handler *LoginVisualHandler) GetSettings(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	settings, err := handler.service.GetSettings(c.Request.Context(), userID)
	if err != nil {
		handler.writeError(c, err, "get login visual settings")
		return
	}
	c.JSON(http.StatusOK, settingsResponse(settings))
}

// Upload handles POST /v1/settings/loginVisual:upload.
func (handler *LoginVisualHandler) Upload(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	contents, err := readUpload(c)
	if err != nil {
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Upload one supported image or video file")
		return
	}
	settings, err := handler.service.Upload(c.Request.Context(), userID, contents)
	if err != nil {
		handler.writeError(c, err, "upload login visual")
		return
	}
	c.JSON(http.StatusOK, settingsResponse(settings))
}

// Publish handles POST /v1/settings/loginVisual:publish.
func (handler *LoginVisualHandler) Publish(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	settings, err := handler.service.Publish(c.Request.Context(), userID)
	if err != nil {
		handler.writeError(c, err, "publish login visual")
		return
	}
	c.JSON(http.StatusOK, settingsResponse(settings))
}

// RestoreDefault handles POST /v1/settings/loginVisual:restoreDefault.
func (handler *LoginVisualHandler) RestoreDefault(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	settings, err := handler.service.RestoreDefault(c.Request.Context(), userID)
	if err != nil {
		handler.writeError(c, err, "restore default login visual")
		return
	}
	c.JSON(http.StatusOK, settingsResponse(settings))
}

// Preview handles GET /v1/settings/loginVisual:preview.
func (handler *LoginVisualHandler) Preview(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		httpdto.Unauthorized(c, "Not authenticated")
		return
	}
	media, reader, err := handler.service.OpenPreview(c.Request.Context(), userID)
	if err != nil {
		handler.writeError(c, err, "read login visual preview")
		return
	}
	defer reader.Close()
	handler.writeMedia(c, media, reader, false)
}

// PublicCurrent handles GET /v1/loginVisual/current.
func (handler *LoginVisualHandler) PublicCurrent(c *gin.Context) {
	visual, err := handler.service.Public(c.Request.Context())
	if err != nil || !visual.HasVisual {
		c.JSON(http.StatusOK, dto.PublicVisualResponse{Kind: "builtIn"})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, dto.PublicVisualResponse{Kind: string(visual.Kind), MediaURL: visual.MediaURL, PosterURL: visual.PosterURL})
}

// PublicMedia handles GET /v1/loginVisual/current/media.
func (handler *LoginVisualHandler) PublicMedia(c *gin.Context) { handler.writePublicMedia(c, false) }

// PublicPoster handles GET /v1/loginVisual/current/poster.
func (handler *LoginVisualHandler) PublicPoster(c *gin.Context) { handler.writePublicMedia(c, true) }

func (handler *LoginVisualHandler) writePublicMedia(c *gin.Context, poster bool) {
	media, reader, err := handler.service.OpenPublished(c.Request.Context(), poster)
	if err != nil {
		httpdto.NotFound(c, "Published login visual is unavailable")
		return
	}
	defer reader.Close()
	handler.writeMedia(c, media, reader, poster)
}

func (handler *LoginVisualHandler) writeMedia(c *gin.Context, media domain.Media, reader io.Reader, poster bool) {
	contentType := media.ContentType
	if poster {
		contentType = "image/png"
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.DataFromReader(http.StatusOK, -1, contentType, reader, nil)
}

func readUpload(c *gin.Context) ([]byte, error) {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		return nil, fmt.Errorf("expected multipart upload")
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLoginVisualRequestBytes)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return nil, err
	}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		contents, readErr := io.ReadAll(io.LimitReader(part, maxLoginVisualRequestBytes+1))
		part.Close()
		if readErr != nil || int64(len(contents)) > maxLoginVisualRequestBytes {
			return nil, fmt.Errorf("upload too large")
		}
		return contents, nil
	}
	return nil, fmt.Errorf("missing file part")
}

func settingsResponse(settings domain.Settings) dto.SettingsResponse {
	response := dto.SettingsResponse{Draft: toMedia(settings.Draft), Published: toMedia(settings.Published)}
	if settings.Draft != nil || settings.Published != nil {
		response.PreviewURL = "/v1/settings/loginVisual:preview"
	}
	return response
}

func toMedia(media *domain.Media) *dto.Media {
	if media == nil {
		return nil
	}
	return &dto.Media{Kind: string(media.Kind), ContentType: media.ContentType, SizeBytes: media.SizeBytes, DurationMS: media.Duration.Milliseconds()}
}

func (handler *LoginVisualHandler) writeError(c *gin.Context, err error, operation string) {
	switch {
	case errors.Is(err, loginvisualapp.ErrPermissionDenied):
		httpdto.Forbidden(c, "Permission denied")
	case errors.Is(err, loginvisualapp.ErrInvalidMedia):
		httpdto.Error(c, http.StatusBadRequest, "INVALID_ARGUMENT", "Unsupported or invalid login visual media")
	case errors.Is(err, loginvisualapp.ErrNoDraft):
		httpdto.Error(c, http.StatusPreconditionFailed, "FAILED_PRECONDITION", "Upload a visual before publishing")
	case errors.Is(err, loginvisualapp.ErrMediaUnavailable):
		httpdto.NotFound(c, "Login visual media is unavailable")
	default:
		httpdto.InternalError(c, "Failed to "+operation)
	}
}
