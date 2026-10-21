package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"gorm.io/gorm"
)

func TestAgentQuotaPostgresExpandedConcurrentLastSlot(t *testing.T) {
	db := openAgentQuotaPostgresDB(t)
	private, verifier := agentLicensePublisher(t)
	key, _ := signAgentLicense(t, private, 5, 1, time.Now().UTC().Add(-time.Minute))
	licenses := licenseRepositoryForTest(t, db, verifier)
	if _, err := licenses.ActivateAt(context.Background(), key, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	for index := 1; index < 5; index++ {
		repository, err := NewAgentRepositoryWithLicenseVerifier(db, verifier)
		if err != nil {
			t.Fatal(err)
		}
		if err := repository.Create(context.Background(), quotaAgent(index, 1)); err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	results := make(chan error, 8)
	for index := 5; index < 13; index++ {
		go func(index int) {
			<-start
			repository, err := NewAgentRepositoryWithLicenseVerifier(db.Session(&gorm.Session{NewDB: true}), verifier)
			if err == nil {
				err = repository.Create(context.Background(), quotaAgent(index, 1+index%2))
			}
			results <- err
		}(index)
	}
	close(start)
	successes := 0
	for index := 0; index < 8; index++ {
		if err := <-results; err == nil {
			successes++
		} else if !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&model.Agent{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if successes != 1 || count != 5 {
		t.Fatalf("successes=%d count=%d", successes, count)
	}
	// Expiry changes only authorization for new identities, never existing rows.
	expired, _ := signAgentLicense(t, private, 5, 1, time.Now().UTC().AddDate(0, -2, 0))
	if err := db.Model(&model.AgentLicense{}).Where("singleton = ?", true).Update("signed_key", expired).Error; err != nil {
		t.Fatal(err)
	}
	repository, err := NewAgentRepositoryWithLicenseVerifier(db, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), quotaAgent(99, 1)); !errors.Is(err, agentdomain.ErrAgentQuotaExceeded) {
		t.Fatal(err)
	}
	license, err := licenses.CurrentAt(context.Background(), time.Now().UTC())
	if err != nil || license.State != "expired" || license.AgentLimit != 3 {
		t.Fatalf("license=%#v err=%v", license, err)
	}
	if err := db.Model(&model.Agent{}).Count(&count).Error; err != nil || count != 5 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestAgentQuotaPostgresActivationSharesLockAndRegistrationReadsCommittedKey(t *testing.T) {
	db := openAgentQuotaPostgresDB(t)
	private, verifier := agentLicensePublisher(t)
	key, _ := signAgentLicense(t, private, 6, 1, time.Now().UTC().Add(-time.Minute))
	licenses := licenseRepositoryForTest(t, db, verifier)
	for index := 1; index <= 3; index++ {
		if err := NewAgentRepository(db).Create(context.Background(), quotaAgent(index, 1)); err != nil {
			t.Fatal(err)
		}
	}
	holder := db.Begin()
	defer holder.Rollback()
	if err := lockAgentQuota(holder); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := licenses.ActivateAt(ctx, key, time.Now().UTC())
	cancel()
	if err == nil {
		t.Fatal("activation did not wait for registration quota lock")
	}
	var count int64
	if err := db.Model(&model.AgentLicense{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("cancelled activation count=%d err=%v", count, err)
	}
	// Write a valid replacement inside the lock holder, then let a registration
	// wait on that same lock. It must read the just-committed Key, not base=3.
	if err := holder.Create(&model.AgentLicense{Singleton: true, SignedKey: key}).Error; err != nil {
		t.Fatal(err)
	}
	repository, err := NewAgentRepositoryWithLicenseVerifier(db.Session(&gorm.Session{NewDB: true}), verifier)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- repository.Create(context.Background(), quotaAgent(4, 2)) }()
	select {
	case err := <-result:
		t.Fatalf("registration skipped held quota lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := holder.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("registration did not resume")
	}
	if err := db.Model(&model.Agent{}).Count(&count).Error; err != nil || count != 4 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestAgentQuotaPostgresBootstrapTransactionUsesExpandedQuota(t *testing.T) {
	db := openAgentQuotaPostgresDB(t)
	private, verifier := agentLicensePublisher(t)
	key, _ := signAgentLicense(t, private, 5, 1, time.Now().UTC().Add(-time.Minute))
	licenses := licenseRepositoryForTest(t, db, verifier)
	if _, err := licenses.ActivateAt(context.Background(), key, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	// Compose bootstrap owns an outer transaction and repository.Create opens a
	// nested transaction. The same entitlement and advisory lock apply there.
	err := db.Transaction(func(tx *gorm.DB) error {
		repository, err := NewAgentRepositoryWithLicenseVerifier(tx, verifier)
		if err != nil {
			return err
		}
		for index := 1; index <= 5; index++ {
			if err := repository.Create(context.Background(), quotaAgent(index, 1)); err != nil {
				return fmt.Errorf("bootstrap identity %d: %w", index, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAgentQuotaPostgresActivationRechecksExpiryAfterLockWait(t *testing.T) {
	db := openAgentQuotaPostgresDB(t)
	private, verifier := agentLicensePublisher(t)
	licenses := licenseRepositoryForTest(t, db, verifier)
	currentKey, currentClaims := signAgentLicense(t, private, 10, 2, time.Now().UTC().Add(-time.Minute))
	if _, err := licenses.ActivateAt(context.Background(), currentKey, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	// Calendar dates in this fixture stay away from month-end clamping. The
	// incoming valid Key expires while its transaction waits for the quota lock.
	expiresAt := time.Now().UTC().Truncate(time.Second).Add(2 * time.Second)
	if expiresAt.Day() > 28 {
		t.Skip("expiry-wait fixture requires a day shared by adjacent months")
	}
	key, claims := signAgentLicense(t, private, 20, 1, expiresAt.AddDate(0, -1, 0))
	if claims.ExpiresAt != expiresAt.Unix() {
		t.Fatal("fixture has unexpected calendar expiry")
	}
	holder := db.Begin()
	defer holder.Rollback()
	if err := lockAgentQuota(holder); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := licenses.ActivateAt(context.Background(), key, time.Now().UTC()); result <- err }()
	timer := time.NewTimer(time.Until(expiresAt) + 50*time.Millisecond)
	defer timer.Stop()
	select {
	case err := <-result:
		t.Fatalf("activation skipped held quota lock: %v", err)
	case <-timer.C:
	}
	if err := holder.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, agentdomain.ErrAgentLicenseKeyExpired) {
			t.Fatalf("lock-wait expiry error=%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("activation did not resume")
	}
	license, err := licenses.CurrentAt(context.Background(), time.Now().UTC())
	if err != nil || *license.LicenseID != currentClaims.LicenseID || license.AgentLimit != 10 {
		t.Fatalf("old authorization changed: %#v err=%v", license, err)
	}
}
