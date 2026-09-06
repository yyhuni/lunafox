package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yyhuni/lunafox/server/internal/modules/notification/domain"
)

func TestMaterializerNotifiesOnlyAfterTransactionCommit(t *testing.T) {
	now := time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC)
	transaction := &materializationTransactionFake{}
	factStore := materializationFactStoreFake{fact: domain.Fact{ID: 10, EventID: "scan:1:succeeded", Kind: domain.KindScanSucceeded, Category: domain.CategoryScan, Priority: domain.PriorityNormal, OccurredAt: now}}
	audience := materializationAudienceStoreFake{projection: AudienceProjection{InboxItems: []domain.InboxItem{{UserID: 7}, {UserID: 7}, {UserID: 9}}}}
	destinations := &materializationDestinationStoreFake{transaction: transaction}
	notifier := &notifierFake{transaction: transaction}
	materializer := NewMaterializer(transaction, factStore, audience, destinations, NewTemplates(), notifier)

	if err := materializer.Materialize(context.Background(), domain.OutboxEvent{ID: 1, Occurrence: testScanSucceededOccurrence(t, now)}); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	if destinations.calledWhileTransactionInactive {
		t.Fatal("delivery fanout ran outside materialization transaction")
	}
	if notifier.calledWhileTransactionActive {
		t.Fatal("realtime notification ran inside materialization transaction")
	}
	if len(notifier.userIDs) != 2 || notifier.userIDs[0] != 7 || notifier.userIDs[1] != 9 {
		t.Fatalf("notifier users = %#v, want unique frozen recipients", notifier.userIDs)
	}
}

func TestMaterializerRoutesDeterministicPersistenceErrorToUnsupported(t *testing.T) {
	now := time.Date(2026, time.August, 7, 0, 0, 0, 0, time.UTC)
	transaction := &materializationTransactionFake{}
	materializer := NewMaterializer(
		transaction,
		materializationFactStoreFake{err: domain.NewDeterministicError("invalid_recipient_locale", errors.New("bad locale"))},
		materializationAudienceStoreFake{},
		&materializationDestinationStoreFake{transaction: transaction},
		NewTemplates(),
		nil,
	)
	err := materializer.Materialize(context.Background(), domain.OutboxEvent{ID: 1, Occurrence: testScanSucceededOccurrence(t, now)})
	var unsupported *UnsupportedEventError
	if !errors.As(err, &unsupported) || unsupported.Code != "invalid_recipient_locale" {
		t.Fatalf("Materialize error = %v, want unsupported invalid_recipient_locale", err)
	}
}

func TestTemplatesRenderNucleiSnapshotsWithOnlyApprovedDisplayFields(t *testing.T) {
	templates := NewTemplates()
	taskID := uuid.MustParse("6dd1f0fd-5b34-4cbb-9c4f-3b9c6c0f6a12")
	commitSHA := "0123456789abcdef0123456789abcdef01234567"
	succeeded, err := domain.NewNucleiPOCSyncSucceededOccurrence(taskID, "git", commitSHA, 184, time.Now().UTC())
	if err != nil {
		t.Fatalf("NewNucleiPOCSyncSucceededOccurrence: %v", err)
	}
	validated, err := domain.DecodeOccurrence(succeeded)
	if err != nil {
		t.Fatalf("DecodeOccurrence success: %v", err)
	}
	snapshot, err := templates.Render(validated, domain.LocaleEnglish)
	if err != nil || !strings.Contains(snapshot.Message, "184") || !strings.Contains(snapshot.Message, commitSHA[:12]) || strings.Contains(snapshot.Message, commitSHA[12:]) {
		t.Fatalf("success render = %#v, %v; want count and only SHA prefix", snapshot, err)
	}

	failed, err := domain.NewNucleiPOCSyncFailedOccurrence(taskID, "git", "TEMPLATE_INVALID", "One or more Nuclei templates failed validation.", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewNucleiPOCSyncFailedOccurrence: %v", err)
	}
	validated, err = domain.DecodeOccurrence(failed)
	if err != nil {
		t.Fatalf("DecodeOccurrence failure: %v", err)
	}
	snapshot, err = templates.Render(validated, domain.LocaleChinese)
	if err != nil || !strings.Contains(snapshot.Message, "TEMPLATE_INVALID") || !strings.Contains(snapshot.Message, "Nuclei") {
		t.Fatalf("failure render = %#v, %v; want code and redacted summary", snapshot, err)
	}
}

type materializationTransactionFake struct{ active bool }

func (transaction *materializationTransactionFake) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	transaction.active = true
	err := callback(ctx)
	transaction.active = false
	return err
}

type materializationFactStoreFake struct {
	fact domain.Fact
	err  error
}

func (store materializationFactStoreFake) CreateOrGetFact(context.Context, domain.ValidatedEvent) (domain.Fact, error) {
	return store.fact, store.err
}

type materializationAudienceStoreFake struct {
	projection AudienceProjection
	err        error
}

func (store materializationAudienceStoreFake) CreateRecipientsAndInbox(context.Context, domain.Fact, func(domain.Locale) (domain.RenderSnapshot, error)) (AudienceProjection, error) {
	return store.projection, store.err
}

type materializationDestinationStoreFake struct {
	transaction                    *materializationTransactionFake
	calledWhileTransactionInactive bool
}

func (store *materializationDestinationStoreFake) CreateDeliveries(context.Context, domain.Fact, domain.RenderSnapshot) error {
	if !store.transaction.active {
		store.calledWhileTransactionInactive = true
	}
	return nil
}

type notifierFake struct {
	transaction                  *materializationTransactionFake
	calledWhileTransactionActive bool
	userIDs                      []int
}

func (notifier *notifierFake) NotifyUser(userID int) {
	if notifier.transaction.active {
		notifier.calledWhileTransactionActive = true
	}
	notifier.userIDs = append(notifier.userIDs, userID)
}
