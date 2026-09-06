package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	securityapp "github.com/yyhuni/lunafox/server/internal/modules/security/application"
	securitydomain "github.com/yyhuni/lunafox/server/internal/modules/security/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/security/handler"
)

type vulnerabilityRouterStoreStub struct{}

func (stub vulnerabilityRouterStoreStub) List(page, pageSize int, filter, orderBy string) ([]securitydomain.Vulnerability, int64, error) {
	return nil, 0, nil
}

func (stub vulnerabilityRouterStoreStub) GetByID(id int) (*securitydomain.Vulnerability, error) {
	return &securitydomain.Vulnerability{ID: id}, nil
}

func (stub vulnerabilityRouterStoreStub) ListByTargetID(targetID, page, pageSize int, filter, orderBy string) ([]securitydomain.Vulnerability, int64, error) {
	return nil, 0, nil
}

func (stub vulnerabilityRouterStoreStub) ListFilterOptions(field string) ([]securitydomain.FilterOption, error) {
	return nil, nil
}

func (stub vulnerabilityRouterStoreStub) ListFilterOptionsByTargetID(targetID int, field string) ([]securitydomain.FilterOption, error) {
	return nil, nil
}

func (stub vulnerabilityRouterStoreStub) GetGlobalVulnerabilityStatistics() (total, pending, reviewed int64, err error) {
	return 0, 0, 0, nil
}

func (stub vulnerabilityRouterStoreStub) GetVulnerabilityStatisticsByTargetID(targetID int) (total, pending, reviewed int64, err error) {
	return 0, 0, 0, nil
}

func (stub vulnerabilityRouterStoreStub) BatchCreate(vulnerabilities []securitydomain.Vulnerability) (int64, error) {
	return int64(len(vulnerabilities)), nil
}

func (stub vulnerabilityRouterStoreStub) BatchCreateContext(_ context.Context, vulnerabilities []securitydomain.Vulnerability) (int64, error) {
	return stub.BatchCreate(vulnerabilities)
}

func (stub vulnerabilityRouterStoreStub) BatchDelete(ids []int) (int64, error) {
	return int64(len(ids)), nil
}

func (stub vulnerabilityRouterStoreStub) MarkAsReviewed(id int) error {
	return nil
}

func (stub vulnerabilityRouterStoreStub) MarkAsUnreviewed(id int) error {
	return nil
}

func (stub vulnerabilityRouterStoreStub) BatchMarkAsReviewed(ids []int) (int64, error) {
	return int64(len(ids)), nil
}

func (stub vulnerabilityRouterStoreStub) BatchMarkAsUnreviewed(ids []int) (int64, error) {
	return int64(len(ids)), nil
}

func (stub vulnerabilityRouterStoreStub) BatchUpdateReviewStates(reviewedIDs []int, unreviewedIDs []int) (int64, error) {
	return int64(len(reviewedIDs) + len(unreviewedIDs)), nil
}

type vulnerabilityRouterTargetStub struct{}

func (stub vulnerabilityRouterTargetStub) GetActiveByID(id int) (*securitydomain.TargetRef, error) {
	return &securitydomain.TargetRef{ID: id}, nil
}

func TestVulnerabilityBatchUpdateRouteReplacesReviewAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/v1")
	vulnerabilityHandler := handler.NewVulnerabilityHandler(securityapp.NewVulnerabilityFacade(vulnerabilityRouterStoreStub{}, vulnerabilityRouterTargetStub{}))
	registerVulnerabilityRoutes(group, vulnerabilityHandler)

	tests := []struct {
		path string
		want int
	}{
		{path: "/v1/vulnerabilities:batchUpdate", want: http.StatusBadRequest},
		{path: "/v1/vulnerabilities:batchReview", want: http.StatusNotFound},
		{path: "/v1/vulnerabilities:batchUnreview", want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, tt.path, nil)
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, request)
			if recorder.Code != tt.want {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}
