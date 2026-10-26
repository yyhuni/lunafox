package repository

import (
	"context"
	"testing"
	"time"

	"github.com/lib/pq"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
)

func TestGlobalAssetExportSnapshotReadsActiveRowsInStableOrder(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	deletedAt := now.Add(-time.Hour)
	if err := db.Create(&model.AssetTargetRef{ID: 2, Name: "deleted.example", Type: assetdomain.TargetTypeDomain, DeletedAt: &deletedAt}).Error; err != nil {
		t.Fatalf("seed targets: %v", err)
	}
	if err := db.Create([]model.Website{
		{ID: 10, TargetID: 1, URL: "https://example.test/new", Host: "example.test", CreatedAt: now, Tech: pq.StringArray{}},
		{ID: 11, TargetID: 1, URL: "https://example.test/old", Host: "example.test", CreatedAt: now.Add(-time.Second), Tech: pq.StringArray{}},
		{ID: 12, TargetID: 2, URL: "https://example.test/deleted", Host: "example.test", CreatedAt: now.Add(time.Second), Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery("example")
	if err != nil {
		t.Fatalf("parse export query: %v", err)
	}

	var got []assetdomain.Website
	err = NewGlobalAssetExportSnapshotStore(db).WithSnapshot(context.Background(), ast, assetapp.GlobalAssetSearchAssetTypeWebsite, func(snapshot assetapp.GlobalAssetExportSnapshot) error {
		return snapshot.ForEachWebsite(context.Background(), func(item assetdomain.Website) error {
			got = append(got, item)
			return nil
		})
	})
	if err != nil {
		t.Fatalf("read export snapshot: %v", err)
	}
	if len(got) != 2 || got[0].ID != 10 || got[1].ID != 11 {
		t.Fatalf("snapshot rows = %+v, want active rows ordered by created_at/id desc", got)
	}
}

func TestGlobalAssetExportSnapshotAppliesCurrentOrganizationPredicate(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)
	if err := db.Create(&model.AssetTargetRef{ID: 2, Name: "other.example", Type: assetdomain.TargetTypeDomain}).Error; err != nil {
		t.Fatalf("seed second target: %v", err)
	}
	if err := db.Exec(`INSERT INTO organization (id, name) VALUES (1, 'Tencent'), (2, 'Other')`).Error; err != nil {
		t.Fatalf("seed organizations: %v", err)
	}
	if err := db.Exec(`INSERT INTO organization_target (organization_id, target_id) VALUES (1, 1)`).Error; err != nil {
		t.Fatalf("seed organization target: %v", err)
	}
	if err := db.Create([]model.Website{
		{ID: 21, TargetID: 1, URL: "https://tencent.example.test", Host: "tencent.example.test", CreatedAt: now, Tech: pq.StringArray{}},
		{ID: 22, TargetID: 2, URL: "https://other.example.test", Host: "other.example.test", CreatedAt: now.Add(-time.Second), Tech: pq.StringArray{}},
	}).Error; err != nil {
		t.Fatalf("seed websites: %v", err)
	}
	ast, err := assetapp.ParseGlobalAssetSearchQuery(`organization=="Tencent"`)
	if err != nil {
		t.Fatalf("parse organization export query: %v", err)
	}

	var got []assetdomain.Website
	err = NewGlobalAssetExportSnapshotStore(db).WithSnapshot(context.Background(), ast, assetapp.GlobalAssetSearchAssetTypeWebsite, func(snapshot assetapp.GlobalAssetExportSnapshot) error {
		return snapshot.ForEachWebsite(context.Background(), func(item assetdomain.Website) error {
			got = append(got, item)
			return nil
		})
	})
	if err != nil {
		t.Fatalf("read organization export snapshot: %v", err)
	}
	if len(got) != 1 || got[0].ID != 21 {
		t.Fatalf("organization export rows = %+v, want only current Tencent relation", got)
	}
}
