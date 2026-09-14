package router

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/application"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/upgrade/handler"
)

type routeUpgradeServiceStub struct{}

func (routeUpgradeServiceStub) CheckForUpdates(context.Context, int) (application.CheckForUpdatesResult, error) {
	return application.CheckForUpdatesResult{}, nil
}

func (routeUpgradeServiceStub) CreateOperation(context.Context, int, application.CreateUpgradeOperationInput) (*domain.Operation, bool, error) {
	return nil, false, nil
}

func (routeUpgradeServiceStub) GetOperation(context.Context, int, string) (*domain.Operation, error) {
	return nil, nil
}

func (routeUpgradeServiceStub) RetryOperation(context.Context, int, string, bool) (*domain.Operation, error) {
	return nil, nil
}

func TestRegisterUpgradeRoutesUsesCanonicalBoundaries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	RegisterUpgradeRoutes(protected, handler.NewUpgradeHandler(routeUpgradeServiceStub{}))

	routes := map[string]struct{}{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		"POST /v1/system:checkForUpdates",
		"POST /v1/upgradeOperations",
		"GET /v1/upgradeOperations/:upgradeOperation",
		"POST /v1/upgradeOperations/*upgradeOperationAction",
	} {
		if _, ok := routes[expected]; !ok {
			t.Fatalf("missing canonical upgrade route %s", expected)
		}
	}
}
