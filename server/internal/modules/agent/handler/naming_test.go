package handler

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAgentHandlerExposesCanonicalListMethod(t *testing.T) {
	h := &AgentHandler{}
	_ = gin.HandlerFunc(h.List)
}
