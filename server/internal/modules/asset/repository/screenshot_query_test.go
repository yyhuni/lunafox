package repository

import (
	"reflect"
	"testing"
	"time"

	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
)

func TestScreenshotRepositoryListByTargetIDUsesScopedStableOrderingAndNullsLast(t *testing.T) {
	db := newAssetRepositoryDB(t)
	older := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	status200 := int16(200)
	status404 := int16(404)
	if err := db.Create([]model.Screenshot{
		{ID: 1, TargetID: 1, URL: "https://example.com/old", StatusCode: &status404, CreatedAt: older, UpdatedAt: older},
		{ID: 2, TargetID: 1, URL: "https://example.com/null-status", StatusCode: nil, CreatedAt: newer, UpdatedAt: newer},
		{ID: 3, TargetID: 1, URL: "https://example.com/new", StatusCode: &status200, CreatedAt: newer, UpdatedAt: newer},
		{ID: 4, TargetID: 2, URL: "https://other.example.com/new", StatusCode: &status200, CreatedAt: newer, UpdatedAt: newer},
	}).Error; err != nil {
		t.Fatalf("seed screenshots failed: %v", err)
	}

	items, total, err := NewScreenshotRepository(db).ListByTargetID(1, 1, 10, "", "createdAt desc")
	if err != nil {
		t.Fatalf("ListByTargetID returned error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected target-scoped total 3, got %d", total)
	}
	gotIDs := []int{items[0].ID, items[1].ID, items[2].ID}
	wantIDs := []int{3, 2, 1}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected createdAt desc stable order: got %v want %v", gotIDs, wantIDs)
	}

	items, _, err = NewScreenshotRepository(db).ListByTargetID(1, 1, 10, "", "statusCode asc")
	if err != nil {
		t.Fatalf("ListByTargetID returned error: %v", err)
	}
	gotIDs = []int{items[0].ID, items[1].ID, items[2].ID}
	wantIDs = []int{3, 1, 2}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("unexpected statusCode asc nulls-last order: got %v want %v", gotIDs, wantIDs)
	}
}

func TestScreenshotRepositoryListSummariesByTargetAndURLsExcludesImage(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	status := int16(200)
	url := "https://api.acme.com/v1"
	if err := db.Create([]model.Screenshot{
		{ID: 1, TargetID: 1, URL: url, StatusCode: &status, Image: []byte("must-not-be-selected"), CreatedAt: now, UpdatedAt: now},
		{ID: 2, TargetID: 2, URL: url, StatusCode: &status, Image: []byte("other-target"), CreatedAt: now, UpdatedAt: now},
		{ID: 3, TargetID: 1, URL: "https://api.acme.com/other", StatusCode: &status, Image: []byte("other-url"), CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatalf("seed screenshots: %v", err)
	}

	items, err := NewScreenshotRepository(db).ListSummariesByTargetAndURLs(1, []string{url})
	if err != nil {
		t.Fatalf("ListSummariesByTargetAndURLs returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 || items[0].URL != url {
		t.Fatalf("expected one exact summary, got %+v", items)
	}
	if len(items[0].Image) != 0 {
		t.Fatalf("screenshot summary must exclude image bytes, got %q", items[0].Image)
	}
}
