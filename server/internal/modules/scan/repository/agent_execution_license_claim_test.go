package repository

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/yyhuni/lunafox/contracts/resourcenames"
	"github.com/yyhuni/lunafox/server/internal/agentlicense"
	agentrepo "github.com/yyhuni/lunafox/server/internal/modules/agent/repository"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

func signedClaimTestGuard(t *testing.T, db *gorm.DB) (*agentrepo.AgentLicenseRepository, ed25519.PrivateKey) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := agentlicense.NewVerifier(base64.RawURLEncoding.EncodeToString(public))
	if err != nil {
		t.Fatal(err)
	}
	guard, err := agentrepo.NewAgentLicenseRepositoryWithVerifier(db, verifier)
	if err != nil {
		t.Fatal(err)
	}
	return guard, private
}

func claimTestKey(t *testing.T, private ed25519.PrivateKey, issuedAt time.Time) string {
	t.Helper()
	claims, err := agentlicense.NewClaims(10, 1, issuedAt)
	if err != nil {
		t.Fatal(err)
	}
	key, err := agentlicense.Sign(private, claims)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func seedOlderClaimIdentities(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, id := range []int{1, 2, 3} {
		if err := db.Exec("INSERT INTO agent (id, created_at) VALUES (?, ?)", id, time.Now().UTC().Add(-time.Hour)).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestSavedPlanLicenseClaimPostgresExpiryReplayCompletionAndRenewal(t *testing.T) {
	db := openSavedPlanLeaseConcurrencyPostgres(t)
	_ = savedPlanTestExecutionGuard(t, db)
	resetSavedPlanLeaseConcurrencyFixture(t, db)
	seedOlderClaimIdentities(t, db)
	guard, private := signedClaimTestGuard(t, db)
	repository := NewScanTaskRepository(db, guard)
	ctx := context.Background()
	active := claimTestKey(t, private, time.Now().UTC().Add(-time.Minute))
	if _, err := guard.ActivateAt(ctx, active, time.Now()); err != nil {
		t.Fatal(err)
	}
	const requestID = "550e8400-e29b-41d4-a716-446655440041"
	first, err := repository.ClaimNextCompatibleSavedExecutionPlan(ctx, 42, "session-a", 9, requestID, []uint32{2})
	if err != nil || first == nil {
		t.Fatalf("initial expanded claim=%v err=%v", first, err)
	}
	second := mustSavedPlanForTask(t, resourcenames.Task(7, 2), resourcenames.Scan(7), "engine.lunafox.port_scan", "ports", "second")
	if err := db.Exec("INSERT INTO scan_task (id, scan_id, stage_order, created_at, status, resolved_execution_plan) VALUES (2, 7, 2, CURRENT_TIMESTAMP, 'pending', ?)", second).Error; err != nil {
		t.Fatal(err)
	}
	expired := claimTestKey(t, private, time.Now().UTC().AddDate(0, -2, 0))
	if err := db.Exec("UPDATE agent_license SET signed_key = ?", expired).Error; err != nil {
		t.Fatal(err)
	}
	replay, err := repository.ClaimNextCompatibleSavedExecutionPlan(ctx, 42, "session-a", 9, requestID, []uint32{2})
	if err != nil || !proto.Equal(first, replay) {
		t.Fatalf("expired assignment replay=%v err=%v", replay, err)
	}
	const nextRequestID = "550e8400-e29b-41d4-a716-446655440042"
	next, err := repository.ClaimNextCompatibleSavedExecutionPlan(ctx, 42, "session-a", 9, nextRequestID, []uint32{2})
	if err != nil || next != nil {
		t.Fatalf("expired new claim=%v err=%v", next, err)
	}
	if committed, err := repository.CommitScanTaskTerminalStatusForSession(ctx, 1, 42, "session-a", 9, taskStatusSucceeded, nil); err != nil || !committed {
		t.Fatalf("expired running completion=%t err=%v", committed, err)
	}
	if _, err := guard.ActivateAt(ctx, active, time.Now()); err != nil {
		t.Fatal(err)
	}
	next, err = repository.ClaimNextCompatibleSavedExecutionPlan(ctx, 42, "session-a", 9, nextRequestID, []uint32{2})
	if err != nil || next == nil || next.GetTask() != resourcenames.Task(7, 2) {
		t.Fatalf("renewed pending claim=%v err=%v", next, err)
	}
	if err := db.Exec("UPDATE agent_license SET signed_key = 'corrupt'").Error; err != nil {
		t.Fatal(err)
	}
	if next, err := repository.ClaimNextCompatibleSavedExecutionPlan(ctx, 42, "session-a", 9, "550e8400-e29b-41d4-a716-446655440043", []uint32{2}); err == nil || next != nil {
		t.Fatalf("corrupt entitlement claim=%v err=%v", next, err)
	}
}

func TestSavedPlanLicenseClaimPostgresExpiryWhileBindingRollsBack(t *testing.T) {
	db := openSavedPlanLeaseConcurrencyPostgres(t)
	_ = savedPlanTestExecutionGuard(t, db)
	resetSavedPlanLeaseConcurrencyFixture(t, db)
	seedOlderClaimIdentities(t, db)
	guard, private := signedClaimTestGuard(t, db)
	// Calendar-month arithmetic preserves this exact near-future expiry.
	issued := time.Now().UTC().Add(3*time.Second).AddDate(0, -1, 0)
	key := claimTestKey(t, private, issued)
	if _, err := guard.ActivateAt(context.Background(), key, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE FUNCTION delay_scan_binding() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(444213); RETURN NEW; END $$",
		"CREATE TRIGGER delay_scan_binding BEFORE UPDATE OF agent_id ON scan FOR EACH ROW EXECUTE FUNCTION delay_scan_binding()",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	holder := db.Begin()
	defer holder.Rollback()
	if err := holder.Exec("SELECT pg_advisory_xact_lock(444213)").Error; err != nil {
		t.Fatal(err)
	}
	type claimResult struct {
		task string
		err  error
	}
	result := make(chan claimResult, 1)
	go func() {
		plan, err := NewScanTaskRepository(db, guard).ClaimNextCompatibleSavedExecutionPlan(context.Background(), 42, "session-a", 9, "550e8400-e29b-41d4-a716-446655440051", []uint32{2})
		if plan == nil {
			result <- claimResult{err: err}
			return
		}
		result <- claimResult{task: plan.GetTask(), err: err}
	}()
	assertPostgresOperationBlocked(t, result)
	// Prove the claim reached the trigger after its first eligibility check.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting int64
		if err := db.Raw("SELECT COUNT(*) FROM pg_locks WHERE locktype = 'advisory' AND objid = 444213 AND NOT granted").Scan(&waiting).Error; err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("claim did not reach binding lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	claims, err := agentlicense.NewClaims(10, 1, issued)
	if err != nil {
		t.Fatal(err)
	}
	for time.Now().Unix() <= claims.ExpiresAt {
		time.Sleep(20 * time.Millisecond)
	}
	if err := holder.Commit().Error; err != nil {
		t.Fatal(err)
	}
	got := <-result
	if got.err != nil || got.task != "" {
		t.Fatalf("expiry-crossing claim=%+v", got)
	}
	var row struct {
		AgentID *int
		Status  string
	}
	if err := db.Table("scan").Select("agent_id, status").Where("id = 7").Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.AgentID != nil || row.Status != scanStatusPending {
		t.Fatalf("rejected claim retained binding: %+v", row)
	}
	if err := db.Table("scan_task").Select("status").Where("id = 1").Scan(&row.Status).Error; err != nil || row.Status != taskStatusPending {
		t.Fatalf("rejected claim changed pending task: %s %v", row.Status, err)
	}
}

func TestSavedPlanLicenseClaimPostgresSharedBoundaryBlocksIdentityChanges(t *testing.T) {
	db := openSavedPlanLeaseConcurrencyPostgres(t)
	_ = savedPlanTestExecutionGuard(t, db)
	resetSavedPlanLeaseConcurrencyFixture(t, db)
	guard, private := signedClaimTestGuard(t, db)
	entered, release := make(chan struct{}), make(chan struct{})
	holderResult := make(chan error, 1)
	go func() {
		holderResult <- guard.WithinClaim(context.Background(), func(context.Context) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	defer func() {
		close(release)
		if err := <-holderResult; err != nil {
			t.Error(err)
		}
	}()
	// A shared holder must allow an unrelated claim, while exclusive identity
	// and Key writers wait at the same boundary before taking any row locks.
	claimCtx, cancelClaim := context.WithTimeout(context.Background(), time.Second)
	defer cancelClaim()
	if plan, err := NewScanTaskRepository(db, guard).ClaimNextCompatibleSavedExecutionPlan(claimCtx, 42, "session-a", 9, "550e8400-e29b-41d4-a716-446655440061", []uint32{2}); err != nil || plan == nil {
		t.Fatalf("shared claim was serialized globally: %v %v", plan, err)
	}
	key := claimTestKey(t, private, time.Now().UTC().Add(-time.Minute))
	for name, operation := range map[string]func(context.Context) error{
		"activation":      func(ctx context.Context) error { _, err := guard.ActivateAt(ctx, key, time.Now()); return err },
		"identity delete": func(ctx context.Context) error { return agentrepo.NewAgentRepository(db).Delete(ctx, 42) },
		"pinned lifecycle delete": func(ctx context.Context) error {
			return NewScanRepository(db).WithAgentExecutionGuard(guard).DeleteAgent(ctx, 42)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := operation(ctx); err == nil || ctx.Err() == nil {
				t.Fatalf("exclusive writer crossed shared boundary: %v", err)
			}
		})
	}
	var count int64
	if err := db.Table("agent").Where("id = 42").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("cancelled deletion changed identity: %d %v", count, err)
	}
}
