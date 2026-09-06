package screenshot

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestScreenshotHandlerExposesCanonicalListMethod(t *testing.T) {
	h := &ScreenshotHandler{}
	_ = gin.HandlerFunc(h.List)
}
