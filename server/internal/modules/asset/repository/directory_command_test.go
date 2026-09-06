package repository

import (
	"context"
	"math"
	"testing"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
)

func TestDirectoryBatchUpsertReplacesCompleteObservationAndReplayConverges(t *testing.T) {
	db := newAssetRepositoryDB(t)
	repo := NewDirectoryRepository(db)
	url := "https://Example.com/%00?x=1#frag"
	firstStatus, secondStatus := 999, 0
	firstLength, secondLength := int64(math.MaxInt64), int64(0)
	firstDuration, secondDuration := int64(math.MaxInt64), int64(0)

	first := assetdomain.Directory{
		TargetID: 1, URL: url, Status: &firstStatus, ContentLength: &firstLength,
		ContentType: "application/octet-stream", Duration: &firstDuration,
	}
	if affected, err := repo.BatchUpsert([]assetdomain.Directory{first}); err != nil || affected != 1 {
		t.Fatalf("first Directory upsert = affected %d, err %v", affected, err)
	}

	var created model.Directory
	if err := db.Where("target_id = ? AND url = ?", 1, url).First(&created).Error; err != nil {
		t.Fatalf("read first Directory observation: %v", err)
	}
	if created.ContentLength == nil || *created.ContentLength != math.MaxInt64 || created.Duration == nil || *created.Duration != math.MaxInt64 {
		t.Fatalf("MaxInt64 observation was not persisted exactly: %+v", created)
	}

	replacement := assetdomain.Directory{
		TargetID: 1, URL: url, Status: &secondStatus, ContentLength: &secondLength,
		ContentType: "", Duration: &secondDuration,
	}
	for attempt := 0; attempt < 2; attempt++ {
		if affected, err := repo.BatchUpsert([]assetdomain.Directory{replacement}); err != nil || affected != 1 {
			t.Fatalf("replacement attempt %d = affected %d, err %v", attempt+1, affected, err)
		}
	}

	var stored model.Directory
	if err := db.Where("target_id = ? AND url = ?", 1, url).First(&stored).Error; err != nil {
		t.Fatalf("read replaced Directory observation: %v", err)
	}
	if stored.ID != created.ID || !stored.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("upsert changed stable identity fields: before=%+v after=%+v", created, stored)
	}
	if stored.Status == nil || *stored.Status != 0 || stored.ContentLength == nil || *stored.ContentLength != 0 || stored.ContentType != "" || stored.Duration == nil || *stored.Duration != 0 {
		t.Fatalf("replacement did not use one complete zero-valued observation: %+v", stored)
	}
	var count int64
	if err := db.Model(&model.Directory{}).Where("target_id = ? AND url = ?", 1, url).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("replay did not converge to one row: count=%d err=%v", count, err)
	}
}

func TestDirectoryRepositoryKeepsExactURLIdentitiesAndFailedScanDoesNotHideAssets(t *testing.T) {
	db := newAssetRepositoryDB(t)
	if err := db.Exec(`CREATE TABLE scan (id INTEGER PRIMARY KEY, target_id INTEGER NOT NULL, status TEXT NOT NULL)`).Error; err != nil {
		t.Fatalf("create terminal Scan fixture: %v", err)
	}
	if err := db.Exec(`INSERT INTO scan (id, target_id, status) VALUES (7, 1, 'failed')`).Error; err != nil {
		t.Fatalf("seed failed Scan fixture: %v", err)
	}

	repo := NewDirectoryRepository(db)
	status := 200
	length, duration := int64(1), int64(2)
	items := []assetdomain.Directory{
		{TargetID: 1, URL: "https://Example.com/%00", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration},
		{TargetID: 1, URL: "https://example.com/%00", Status: &status, ContentLength: &length, ContentType: "text/html", Duration: &duration},
	}
	if affected, err := repo.BatchUpsert(items); err != nil || affected != 2 {
		t.Fatalf("exact-identity upsert = affected %d, err %v", affected, err)
	}

	listed, total, err := repo.ListByTargetID(1, 1, 20, "", "createdAt asc")
	if err != nil || total != 2 || len(listed) != 2 {
		t.Fatalf("failed Scan hid Target Directory assets: total=%d listed=%d err=%v", total, len(listed), err)
	}
	seen := map[string]bool{}
	if err := repo.ForEachByTargetID(context.Background(), 1, func(item assetdomain.Directory) error {
		seen[item.URL] = true
		return nil
	}); err != nil {
		t.Fatalf("stream Target Directory assets: %v", err)
	}
	if !seen[items[0].URL] || !seen[items[1].URL] {
		t.Fatalf("exact URL identities were collapsed or hidden: %#v", seen)
	}
	if count, err := repo.CountByTargetID(1); err != nil || count != 2 {
		t.Fatalf("Target Directory count after failed Scan = %d, err %v", count, err)
	}
}
