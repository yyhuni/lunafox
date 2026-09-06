package repository

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	systemdomain "github.com/yyhuni/lunafox/server/internal/modules/system/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/system/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestServerLocationRepositoryAtomicallyReplacesSingletonSuccess(t *testing.T) {
	db := openServerLocationTestDatabase(t)
	repository := &serverLocationRepository{db: db}
	ctx := context.Background()
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	if location, err := repository.Get(ctx); err != nil || location != nil {
		t.Fatalf("initial Get() = (%#v, %v), want unknown", location, err)
	}
	firstRadius := 12.5
	first := systemdomain.ServerLocationSnapshot{
		ObservedEgressIP: "8.8.8.8",
		Latitude:         37.4219999,
		Longitude:        -122.0840575,
		AccuracyRadiusKM: &firstRadius,
		ProviderKey:      "freeipapi",
		ResolvedAt:       base,
		UpdatedAt:        base,
	}
	if err := repository.Replace(ctx, first); err != nil {
		t.Fatalf("Replace(first): %v", err)
	}
	if err := repository.MarkExpired(ctx); err != nil {
		t.Fatalf("MarkExpired: %v", err)
	}
	second := systemdomain.ServerLocationSnapshot{
		ObservedEgressIP: "1.1.1.1",
		Latitude:         -33.86,
		Longitude:        151.21,
		ProviderKey:      "freeipapi",
		ResolvedAt:       base.Add(time.Hour),
		UpdatedAt:        base.Add(time.Hour),
	}
	if err := repository.Replace(ctx, second); err != nil {
		t.Fatalf("Replace(second): %v", err)
	}

	stored, err := repository.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored == nil || stored.ObservedEgressIP != "1.1.1.1" || stored.Latitude != second.Latitude || stored.Longitude != second.Longitude || stored.AccuracyRadiusKM != nil || stored.ProviderKey != "freeipapi" || stored.ForcedExpired || !stored.ResolvedAt.Equal(second.ResolvedAt) {
		t.Fatalf("replaced singleton = %#v", stored)
	}
	var count int64
	if err := db.Model(&persistence.ServerLocationSnapshot{}).Count(&count).Error; err != nil {
		t.Fatalf("count singleton rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("singleton row count = %d, want 1", count)
	}
}

func TestServerLocationRepositoryFastFailsIncompleteSuccess(t *testing.T) {
	repository := &serverLocationRepository{db: openServerLocationTestDatabase(t)}
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	valid := systemdomain.ServerLocationSnapshot{
		ObservedEgressIP: "8.8.8.8",
		Latitude:         1,
		Longitude:        2,
		ProviderKey:      "freeipapi",
		ResolvedAt:       base,
		UpdatedAt:        base,
	}
	tests := []struct {
		name   string
		mutate func(*systemdomain.ServerLocationSnapshot)
	}{
		{name: "private egress", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) { snapshot.ObservedEgressIP = "10.0.0.1" }},
		{name: "non-normalized egress", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) { snapshot.ObservedEgressIP = " 8.8.8.8 " }},
		{name: "invalid latitude", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) { snapshot.Latitude = 91 }},
		{name: "invalid longitude", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) { snapshot.Longitude = math.Inf(1) }},
		{name: "invalid radius", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) {
			radius := -1.0
			snapshot.AccuracyRadiusKM = &radius
		}},
		{name: "missing provider", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) { snapshot.ProviderKey = "" }},
		{name: "missing resolution", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) { snapshot.ResolvedAt = time.Time{} }},
		{name: "forced expired success", mutate: func(snapshot *systemdomain.ServerLocationSnapshot) { snapshot.ForcedExpired = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := valid
			test.mutate(&snapshot)
			if err := repository.Replace(context.Background(), snapshot); err == nil {
				t.Fatal("invalid Server success did not fast-fail")
			}
		})
	}
}

func openServerLocationTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "server-location.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	schema := "CREATE TABLE server_location_snapshot (" +
		"singleton_id INTEGER PRIMARY KEY," +
		"observed_egress_ip TEXT NOT NULL," +
		"latitude REAL NOT NULL," +
		"longitude REAL NOT NULL," +
		"accuracy_radius_km REAL," +
		"provider_key TEXT NOT NULL," +
		"resolved_at DATETIME NOT NULL," +
		"forced_expired BOOLEAN NOT NULL DEFAULT FALSE," +
		"updated_at DATETIME NOT NULL" +
		")"
	if err := db.Exec(schema).Error; err != nil {
		t.Fatalf("create Server location table: %v", err)
	}
	return db
}
