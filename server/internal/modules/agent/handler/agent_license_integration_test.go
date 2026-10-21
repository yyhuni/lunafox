package handler

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyhuni/lunafox/server/internal/agentlicense"
	agentapp "github.com/yyhuni/lunafox/server/internal/modules/agent/application"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/agent/dto"
	agentinfra "github.com/yyhuni/lunafox/server/internal/modules/agent/infrastructure"
	agentrepo "github.com/yyhuni/lunafox/server/internal/modules/agent/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAgentLicenseActivationIntegratesHTTPPersistenceSummaryAndRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "activation.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// This fixture implements the actual repository SQL surface; no entitlement,
	// summary, or registration port is stubbed in this cross-layer test.
	if err := db.Exec(`
		CREATE TABLE agent_license (singleton BOOLEAN PRIMARY KEY CHECK(singleton = TRUE), signed_key TEXT NOT NULL);
		CREATE TABLE registration_token (id INTEGER PRIMARY KEY AUTOINCREMENT, token TEXT UNIQUE NOT NULL, expires_at DATETIME NOT NULL, ever_attributed_at DATETIME, created_at DATETIME);
		CREATE TABLE agent (id INTEGER PRIMARY KEY AUTOINCREMENT, instance_id TEXT UNIQUE NOT NULL, display_name TEXT NOT NULL, authentication_token TEXT UNIQUE NOT NULL, status TEXT, max_tasks INTEGER, cpu_threshold INTEGER, mem_threshold INTEGER, disk_threshold INTEGER, registration_token_id INTEGER NOT NULL, created_at DATETIME, updated_at DATETIME);
		CREATE TABLE agent_runtime_status (agent_id INTEGER PRIMARY KEY, health_state TEXT, last_heartbeat DATETIME, task_slots_used INTEGER);
		CREATE TABLE agent_location (agent_id INTEGER PRIMARY KEY, latitude REAL, longitude REAL, accuracy_radius_km REAL, provider_key TEXT, resolved_at DATETIME);
	`).Error; err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := agentlicense.NewVerifier(base64.RawURLEncoding.EncodeToString(public))
	if err != nil {
		t.Fatal(err)
	}
	agents, err := agentrepo.NewAgentRepositoryWithLicenseVerifier(db, verifier)
	if err != nil {
		t.Fatal(err)
	}
	licenses, err := agentrepo.NewAgentLicenseRepositoryWithVerifier(db, verifier)
	if err != nil {
		t.Fatal(err)
	}
	clock := agentinfra.NewSystemClock()
	licenseService, err := agentapp.NewAgentLicenseService(licenses, clock)
	if err != nil {
		t.Fatal(err)
	}
	summaryService, err := agentapp.NewAgentClusterSummaryService(agents, clock, licenses)
	if err != nil {
		t.Fatal(err)
	}
	registrations := agentapp.NewAgentRegistrationService(agents, agentrepo.NewRegistrationTokenRepository(db), clock, agentinfra.NewCryptoTokenGenerator())
	token, err := registrations.CreateRegistrationToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 3; index++ {
		if _, err := registrations.RegisterAgent(context.Background(), token.Token, fmt.Sprintf("node-%d", index), "1.0.0", agentdomain.AgentRegistrationOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := registrations.RegisterAgent(context.Background(), token.Token, "fourth", "1.0.0", agentdomain.AgentRegistrationOptions{}); !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatalf("base quota error=%v", err)
	}
	router := gin.New()
	router.POST("/v1/admin/agentLicenses/:licenseAction", NewAgentLicenseHandler(licenseService).Activate)
	router.GET("/v1/admin/agentClusterSummaries/current", NewAgentClusterSummaryHandler(summaryService).Current)
	claims, err := agentlicense.NewClaims(10, 2, time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	key, err := agentlicense.Sign(private, claims)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]string{"key": key})
	if err != nil {
		t.Fatal(err)
	}
	activation := httptest.NewRecorder()
	router.ServeHTTP(activation, httptest.NewRequest(http.MethodPost, "/v1/admin/agentLicenses/current:activate", strings.NewReader(string(body))))
	if activation.Code != http.StatusOK {
		t.Fatalf("activation status=%d body=%s", activation.Code, activation.Body.String())
	}
	var license dto.AgentLicenseResponse
	if err := json.Unmarshal(activation.Body.Bytes(), &license); err != nil {
		t.Fatal(err)
	}
	if license.Name != "agentLicenses/current" || license.State != "active" || license.AgentLimit != 10 || license.ExpiresAt.Unix() != claims.ExpiresAt {
		t.Fatalf("license=%#v", license)
	}
	if _, err := registrations.RegisterAgent(context.Background(), token.Token, "fourth", "1.0.0", agentdomain.AgentRegistrationOptions{}); err != nil {
		t.Fatalf("expanded fourth registration failed: %v", err)
	}
	summary := httptest.NewRecorder()
	router.ServeHTTP(summary, httptest.NewRequest(http.MethodGet, "/v1/admin/agentClusterSummaries/current", nil))
	if summary.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", summary.Code, summary.Body.String())
	}
	var projection dto.AgentClusterSummaryResponse
	if err := json.Unmarshal(summary.Body.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	if projection.AgentLimit != 10 || projection.TotalNodes != 4 || projection.License.State != "active" || projection.License.AgentLimit != 10 || *projection.License.LicenseID != claims.LicenseID || projection.License.ExpiresAt.Unix() != claims.ExpiresAt {
		t.Fatalf("summary=%#v", projection)
	}
	if strings.Contains(summary.Body.String(), key) || strings.Contains(activation.Body.String(), key) {
		t.Fatal("signed Key leaked into entitlement read model")
	}
}
