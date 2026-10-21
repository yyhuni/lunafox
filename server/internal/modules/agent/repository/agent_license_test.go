package repository

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/server/internal/agentlicense"
	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/gorm"
)

func agentLicensePublisher(t *testing.T) (ed25519.PrivateKey, *agentlicense.Verifier) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := agentlicense.NewVerifier(base64.RawURLEncoding.EncodeToString(public))
	if err != nil {
		t.Fatal(err)
	}
	return private, verifier
}

func signAgentLicense(t *testing.T, private ed25519.PrivateKey, limit, months int, now time.Time) (string, agentlicense.Claims) {
	t.Helper()
	claims, err := agentlicense.NewClaims(limit, months, now)
	if err != nil {
		t.Fatal(err)
	}
	key, err := agentlicense.Sign(private, claims)
	if err != nil {
		t.Fatal(err)
	}
	return key, claims
}

func licenseRepositoryForTest(t *testing.T, db *gorm.DB, verifier AgentLicenseVerifier) *AgentLicenseRepository {
	t.Helper()
	repository, err := NewAgentLicenseRepositoryWithVerifier(db, verifier)
	if err != nil {
		t.Fatal(err)
	}
	return repository
}

func TestAgentLicensePersistsFixedEntitlementsAcrossReplayAndRedeployment(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	private, verifier := agentLicensePublisher(t)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	key, claims := signAgentLicense(t, private, 10, 2, now)
	repository := licenseRepositoryForTest(t, db, verifier)
	inactive, err := repository.CurrentAt(context.Background(), now)
	if err != nil || inactive.State != "inactive" || inactive.AgentLimit != 3 || inactive.Months != nil || inactive.ExpiresAt != nil || inactive.LicenseID != nil {
		t.Fatalf("inactive=%#v err=%v", inactive, err)
	}
	for _, instant := range []time.Time{now, now.Add(24 * time.Hour)} {
		license, err := repository.ActivateAt(context.Background(), " "+key+"\n", instant)
		if err != nil || license.State != "active" || license.AgentLimit != 10 || *license.Months != 2 || license.ExpiresAt.Unix() != claims.ExpiresAt || *license.LicenseID != claims.LicenseID {
			t.Fatalf("activation=%#v err=%v", license, err)
		}
	}
	// A new repository instance models a restarted Server; another database
	// models a fresh VPS. Neither import recalculates the signed expiry.
	for _, candidate := range []*gorm.DB{db, openRegistrationTokenTestDB(t)} {
		fresh := licenseRepositoryForTest(t, candidate, verifier)
		license, err := fresh.ActivateAt(context.Background(), key, now.Add(48*time.Hour))
		if err != nil || license.ExpiresAt.Unix() != claims.ExpiresAt {
			t.Fatalf("redeploy=%#v err=%v", license, err)
		}
	}
	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if license, err := repository.CurrentAt(context.Background(), expiresAt.Add(-time.Nanosecond)); err != nil || license.State != "active" {
		t.Fatalf("before expiry=%#v err=%v", license, err)
	}
	license, err := repository.CurrentAt(context.Background(), expiresAt)
	if err != nil || license.State != "expired" || license.AgentLimit != 3 || license.ExpiresAt.Unix() != claims.ExpiresAt || *license.Months != 2 || *license.LicenseID != claims.LicenseID {
		t.Fatalf("expired=%#v err=%v", license, err)
	}
}

func TestAgentLicenseRejectedImportsKeepCurrentAuthorization(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	private, verifier := agentLicensePublisher(t)
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	key, claims := signAgentLicense(t, private, 10, 2, now)
	repository := licenseRepositoryForTest(t, db, verifier)
	if _, err := repository.ActivateAt(context.Background(), key, now); err != nil {
		t.Fatal(err)
	}
	expired, _ := signAgentLicense(t, private, 12, 1, now.AddDate(0, -2, 0))
	future, _ := signAgentLicense(t, private, 12, 1, now.Add(time.Hour))
	otherPrivate, _ := agentLicensePublisher(t)
	foreign, _ := signAgentLicense(t, otherPrivate, 12, 1, now)
	for _, sample := range []struct {
		name, key string
		want      error
	}{
		{"required", " \n", agentdomain.ErrAgentLicenseKeyRequired},
		{"invalid", "LF-DEMO-1M", agentdomain.ErrAgentLicenseKeyInvalid},
		{"oversize", strings.Repeat("x", 4097), agentdomain.ErrAgentLicenseKeyInvalid},
		{"foreign", foreign, agentdomain.ErrAgentLicenseKeyInvalid},
		{"expired", expired, agentdomain.ErrAgentLicenseKeyExpired},
		{"future", future, agentdomain.ErrAgentLicenseKeyNotYetValid},
	} {
		t.Run(sample.name, func(t *testing.T) {
			if _, err := repository.ActivateAt(context.Background(), sample.key, now); !errors.Is(err, sample.want) {
				t.Fatalf("err=%v want=%v", err, sample.want)
			}
			license, err := repository.CurrentAt(context.Background(), now)
			if err != nil || *license.LicenseID != claims.LicenseID || license.AgentLimit != 10 {
				t.Fatalf("old authorization changed: %#v err=%v", license, err)
			}
		})
	}
	newKey, newClaims := signAgentLicense(t, private, 20, 3, now)
	license, err := repository.ActivateAt(context.Background(), newKey, now)
	if err != nil || license.AgentLimit != 20 || *license.LicenseID != newClaims.LicenseID {
		t.Fatalf("replace=%#v err=%v", license, err)
	}
	var count int64
	if err := db.Model(&model.AgentLicense{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("singleton count=%d err=%v", count, err)
	}
}

func TestAgentLicenseStorageFaultsFailClosed(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	_, verifier := agentLicensePublisher(t)
	repository := licenseRepositoryForTest(t, db, verifier)
	now := time.Now().UTC()
	if err := db.Create(&model.AgentLicense{Singleton: true, SignedKey: "corrupt"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CurrentAt(context.Background(), now); err == nil {
		t.Fatal("corrupt Key became inactive")
	}
	insertRegistrationToken(t, db, 1, "aaaaaaaa", now.Add(time.Hour))
	agents, err := NewAgentRepositoryWithLicenseVerifier(db, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if err := agents.Create(context.Background(), quotaAgent(1, 1)); err == nil || errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatalf("corrupt Key registration error=%v", err)
	}
	if err := db.Exec("DROP TABLE agent_license").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CurrentAt(context.Background(), now); err == nil {
		t.Fatal("missing table became inactive")
	}
	if err := agents.Create(context.Background(), quotaAgent(1, 1)); err == nil {
		t.Fatal("registration ignored missing table")
	}
	if _, err := NewAgentLicenseRepositoryWithVerifier(db, nil); err == nil {
		t.Fatal("nil verifier accepted")
	}
	if _, err := NewAgentLicenseRepositoryWithVerifier(nil, verifier); err == nil {
		t.Fatal("nil db accepted")
	}
	if _, err := NewAgentLicenseRepository(nil).CurrentAt(context.Background(), now); err == nil {
		t.Fatal("nil db read accepted")
	}
}

func TestAgentLicenseExpandedQuotaAndExpiryPreserveExistingAgents(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	private, verifier := agentLicensePublisher(t)
	now := time.Now().UTC()
	key, _ := signAgentLicense(t, private, 5, 1, now.Add(-time.Minute))
	licenses := licenseRepositoryForTest(t, db, verifier)
	if _, err := licenses.ActivateAt(context.Background(), key, now); err != nil {
		t.Fatal(err)
	}
	agents, err := NewAgentRepositoryWithLicenseVerifier(db, verifier)
	if err != nil {
		t.Fatal(err)
	}
	insertRegistrationToken(t, db, 1, "aaaaaaaa", now.Add(time.Hour))
	for index := 1; index <= 5; index++ {
		if err := agents.Create(context.Background(), quotaAgent(index, 1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := agents.Create(context.Background(), quotaAgent(6, 1)); !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatal(err)
	}
	expired, _ := signAgentLicense(t, private, 5, 1, now.AddDate(0, -2, 0))
	if err := db.Model(&model.AgentLicense{}).Where("singleton = ?", true).Update("signed_key", expired).Error; err != nil {
		t.Fatal(err)
	}
	if err := agents.Create(context.Background(), quotaAgent(6, 1)); !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatal(err)
	}
	var retained model.Agent
	if err := db.First(&retained, "instance_id = ?", "quota-5").Error; err != nil || retained.InstanceID != "quota-5" {
		t.Fatalf("existing agent disappeared: %v", err)
	}
	var count int64
	if err := db.Model(&model.Agent{}).Count(&count).Error; err != nil || count != 5 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
