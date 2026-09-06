package router

import (
	"context"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	blacklistapp "github.com/yyhuni/lunafox/server/internal/modules/blacklist/application"
	"github.com/yyhuni/lunafox/server/internal/modules/blacklist/handler"
)

type blacklistPolicyRouteServiceStub struct{}

func (blacklistPolicyRouteServiceStub) GetGlobal(context.Context) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func (blacklistPolicyRouteServiceStub) GetTarget(context.Context, int) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func (blacklistPolicyRouteServiceStub) ReplaceGlobal(context.Context, blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func (blacklistPolicyRouteServiceStub) ReplaceTarget(context.Context, int, blacklistapp.ReplaceBlacklistPolicyInput) (*blacklistapp.BlacklistPolicy, error) {
	return nil, nil
}

func TestRegisterBlacklistPolicyRoutesExposesOnlySingletonMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	protected := engine.Group("/v1")
	policyHandler, err := handler.NewBlacklistPolicyHandler(blacklistPolicyRouteServiceStub{})
	if err != nil {
		t.Fatal(err)
	}
	RegisterBlacklistPolicyRoutes(protected, policyHandler)

	routes := map[string]struct{}{}
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
		if strings.Contains(route.Path, "blacklist") && strings.Contains(route.Path, "blacklistPolicies") {
			t.Fatalf("legacy collection route registered: %s %s", route.Method, route.Path)
		}
	}
	for _, expected := range []string{
		"GET /v1/blacklistPolicy",
		"PATCH /v1/blacklistPolicy",
		"GET /v1/targets/:target/blacklistPolicy",
		"PATCH /v1/targets/:target/blacklistPolicy",
	} {
		if _, ok := routes[expected]; !ok {
			t.Fatalf("missing route %s", expected)
		}
	}
	for _, forbidden := range []string{
		"POST /v1/blacklistPolicy",
		"DELETE /v1/blacklistPolicy",
		"GET /v1/blacklistPolicies",
		"POST /v1/targets/:target/blacklistPolicy",
		"DELETE /v1/targets/:target/blacklistPolicy",
	} {
		if _, ok := routes[forbidden]; ok {
			t.Fatalf("forbidden lifecycle route registered: %s", forbidden)
		}
	}
}
