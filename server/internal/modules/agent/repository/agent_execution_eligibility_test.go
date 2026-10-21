package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	agentdomain "github.com/yyhuni/lunafox/server/internal/modules/agent/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/agent/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/dbtx"
	"gorm.io/gorm"
)

func TestAgentExecutionEligibilityUsesOldestIdentitiesAndRestoresOnRenewal(t *testing.T) {
	ctx := context.Background()
	db := openRegistrationTokenTestDB(t)
	private, verifier := agentLicensePublisher(t)
	repository := licenseRepositoryForTest(t, db, verifier)
	now := time.Now().UTC()
	insertRegistrationToken(t, db, 1, "aaaaaaaa", now.Add(time.Hour))
	for _, id := range []int{5, 2, 4, 1, 3} {
		record := domainAgentToModel(quotaAgent(id, 1))
		record.ID, record.CreatedAt = id, now
		record.Status = "offline"
		if id == 1 {
			record.CreatedAt = now.Add(time.Minute)
		}
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	assertIDs := func(want []int) {
		t.Helper()
		ids, err := repository.EligibleAgentIDsAt(ctx, now)
		if err != nil || !reflect.DeepEqual(ids, want) {
			t.Fatalf("eligible identities=%v err=%v want=%v", ids, err, want)
		}
	}
	assertIDs([]int{2, 3, 4})
	if eligibility, err := repository.ExecutionEligibilityAt(ctx, 1, now); err != nil || eligibility != agentdomain.AgentExecutionLicenseRestricted {
		t.Fatalf("new identity=%q err=%v", eligibility, err)
	}
	if err := NewAgentRepository(db).Delete(ctx, 2); err != nil {
		t.Fatal(err)
	}
	assertIDs([]int{3, 4, 5})
	key, _ := signAgentLicense(t, private, 5, 1, now.Add(-time.Minute))
	if _, err := repository.ActivateAt(ctx, key, now); err != nil {
		t.Fatal(err)
	}
	assertIDs([]int{3, 4, 5, 1})
	if eligibility, err := repository.ExecutionEligibilityAt(ctx, 1, now); err != nil || eligibility != agentdomain.AgentExecutionEligible {
		t.Fatalf("renewed identity=%q err=%v", eligibility, err)
	}
	if _, err := repository.ExecutionEligibilityAt(ctx, 99, now); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing identity=%v", err)
	}
}

func TestAgentExecutionEligibilityParticipatesInTransactionAndRollsBack(t *testing.T) {
	db := openRegistrationTokenTestDB(t)
	_, verifier := agentLicensePublisher(t)
	repository := licenseRepositoryForTest(t, db, verifier)
	rollback := errors.New("rollback")
	err := repository.WithinIdentityChange(context.Background(), func(ctx context.Context) error {
		if err := dbtx.Resolve(ctx, db).Create(&model.AgentLicense{Singleton: true, SignedKey: "corrupt"}).Error; err != nil {
			return err
		}
		if _, err := repository.CurrentAt(ctx, time.Now()); err == nil {
			t.Fatal("CurrentAt ignored transaction-local corrupt Key")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	if license, err := repository.CurrentAt(context.Background(), time.Now()); err != nil || license.State != "inactive" {
		t.Fatalf("rollback lost atomicity: %#v %v", license, err)
	}
	if _, err := repository.ExecutionEligibilityAt(context.Background(), 0, time.Now()); err == nil {
		t.Fatal("invalid identity accepted")
	}
	if err := repository.WithinClaim(context.Background(), nil); err == nil {
		t.Fatal("nil callback accepted")
	}
}
