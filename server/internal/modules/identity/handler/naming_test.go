package handler

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserHandlerExposesCanonicalListMethod(t *testing.T) {
	h := &UserHandler{}
	_ = gin.HandlerFunc(h.List)
}

func TestOrganizationHandlerExposesCanonicalListMethod(t *testing.T) {
	h := &OrganizationHandler{}
	_ = gin.HandlerFunc(h.List)
}
