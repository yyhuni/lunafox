package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/asset/repository/persistence"
	"gorm.io/gorm"
)

func TestRepositoryFindByTargetIDReturnsQueryError(t *testing.T) {
	expectedErr := errors.New("query failed")

	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "directory",
			run: func(t *testing.T) {
				repo := NewDirectoryRepository(newHostPortQueryScriptedDB(t,
					hostPortQueryResult{
						expect:  "COUNT(*)",
						columns: []string{"count"},
						rows:    [][]driver.Value{{int64(1)}},
					},
					hostPortQueryResult{err: expectedErr},
				))

				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "endpoint",
			run: func(t *testing.T) {
				repo := NewEndpointRepository(newHostPortQueryScriptedDB(t,
					hostPortQueryResult{
						expect:  "COUNT(*)",
						columns: []string{"count"},
						rows:    [][]driver.Value{{int64(1)}},
					},
					hostPortQueryResult{err: expectedErr},
				))

				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "subdomain",
			run: func(t *testing.T) {
				repo := NewSubdomainRepository(newHostPortQueryScriptedDB(t,
					hostPortQueryResult{
						expect:  "COUNT(*)",
						columns: []string{"count"},
						rows:    [][]driver.Value{{int64(1)}},
					},
					hostPortQueryResult{err: expectedErr},
				))

				items, total, err := repo.ListByTargetID(1, 1, 10, "", "")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "website",
			run: func(t *testing.T) {
				repo := NewWebsiteRepository(newHostPortQueryScriptedDB(t,
					hostPortQueryResult{
						expect:  "COUNT(*)",
						columns: []string{"count"},
						rows:    [][]driver.Value{{int64(1)}},
					},
					hostPortQueryResult{err: expectedErr},
				))

				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "screenshot",
			run: func(t *testing.T) {
				repo := NewScreenshotRepository(newHostPortQueryScriptedDB(t,
					hostPortQueryResult{
						expect:  "COUNT(*)",
						columns: []string{"count"},
						rows:    [][]driver.Value{{int64(1)}},
					},
					hostPortQueryResult{err: expectedErr},
				))

				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func TestRepositoryFindByTargetIDReturnsCountError(t *testing.T) {
	expectedErr := errors.New("count failed")

	testCases := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "directory",
			run: func(t *testing.T) {
				repo := NewDirectoryRepository(newHostPortQueryScriptedDB(t, hostPortQueryResult{err: expectedErr}))
				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "endpoint",
			run: func(t *testing.T) {
				repo := NewEndpointRepository(newHostPortQueryScriptedDB(t, hostPortQueryResult{err: expectedErr}))
				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "subdomain",
			run: func(t *testing.T) {
				repo := NewSubdomainRepository(newHostPortQueryScriptedDB(t, hostPortQueryResult{err: expectedErr}))
				items, total, err := repo.ListByTargetID(1, 1, 10, "", "")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "website",
			run: func(t *testing.T) {
				repo := NewWebsiteRepository(newHostPortQueryScriptedDB(t, hostPortQueryResult{err: expectedErr}))
				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
		{
			name: "screenshot",
			run: func(t *testing.T) {
				repo := NewScreenshotRepository(newHostPortQueryScriptedDB(t, hostPortQueryResult{err: expectedErr}))
				items, total, err := repo.ListByTargetID(1, 1, 10, "", "createdAt desc")
				if !errors.Is(err, expectedErr) || items != nil || total != 0 {
					t.Fatalf("unexpected result items=%v total=%d err=%v", items, total, err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}

func TestRepositoryGetByIDNotFound(t *testing.T) {
	db := newAssetRepositoryDB(t)

	testCases := []struct {
		name string
		run  func() error
	}{
		{
			name: "endpoint",
			run: func() error {
				_, err := NewEndpointRepository(db).GetByID(999)
				return err
			},
		},
		{
			name: "website",
			run: func() error {
				_, err := NewWebsiteRepository(db).GetByID(999)
				return err
			},
		},
		{
			name: "screenshot",
			run: func() error {
				_, err := NewScreenshotRepository(db).GetByID(999)
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatalf("expected ErrRecordNotFound, got %v", err)
			}
		})
	}
}

func TestRepositoryForEachPropagatesCallbackError(t *testing.T) {
	db := newAssetRepositoryDB(t)
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	callbackErr := errors.New("callback failed")

	testCases := []struct {
		name string
		run  func(*testing.T, *gorm.DB)
	}{
		{
			name: "directory",
			run: func(t *testing.T, db *gorm.DB) {
				if err := db.Create(directoryFixture(1, now)).Error; err != nil {
					t.Fatalf("create directory: %v", err)
				}
				if err := NewDirectoryRepository(db).ForEachByTargetID(context.Background(), 1, func(assetdomain.Directory) error { return callbackErr }); !errors.Is(err, callbackErr) {
					t.Fatalf("expected callback error, got %v", err)
				}
			},
		},
		{
			name: "endpoint",
			run: func(t *testing.T, db *gorm.DB) {
				if err := db.Create(endpointFixture(1, now)).Error; err != nil {
					t.Fatalf("create endpoint: %v", err)
				}
				if err := NewEndpointRepository(db).ForEachByTargetID(context.Background(), 1, func(assetdomain.Endpoint) error { return callbackErr }); !errors.Is(err, callbackErr) {
					t.Fatalf("expected callback error, got %v", err)
				}
			},
		},
		{
			name: "host_port",
			run: func(t *testing.T, db *gorm.DB) {
				if err := db.Create(hostPortFixture(1, now)).Error; err != nil {
					t.Fatalf("create host port: %v", err)
				}
				if err := NewHostPortRepository(db).ForEachByTargetID(context.Background(), 1, func(assetdomain.HostPort) error { return callbackErr }); !errors.Is(err, callbackErr) {
					t.Fatalf("expected callback error, got %v", err)
				}
			},
		},
		{
			name: "subdomain",
			run: func(t *testing.T, db *gorm.DB) {
				if err := db.Create(subdomainFixture(1, now)).Error; err != nil {
					t.Fatalf("create subdomain: %v", err)
				}
				if err := NewSubdomainRepository(db).ForEachByTargetID(context.Background(), 1, func(assetdomain.Subdomain) error { return callbackErr }); !errors.Is(err, callbackErr) {
					t.Fatalf("expected callback error, got %v", err)
				}
			},
		},
		{
			name: "website",
			run: func(t *testing.T, db *gorm.DB) {
				if err := db.Create(websiteFixture(1, now)).Error; err != nil {
					t.Fatalf("create website: %v", err)
				}
				if err := NewWebsiteRepository(db).ForEachByTargetID(context.Background(), 1, func(assetdomain.Website) error { return callbackErr }); !errors.Is(err, callbackErr) {
					t.Fatalf("expected callback error, got %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, db)
		})
	}
}

func directoryFixture(targetID int, createdAt time.Time) *model.Directory {
	status := 200
	return &model.Directory{TargetID: targetID, URL: "https://example.com/dir", Status: &status, CreatedAt: createdAt}
}

func endpointFixture(targetID int, createdAt time.Time) *model.Endpoint {
	return &model.Endpoint{TargetID: targetID, URL: "https://example.com/api", Host: "example.com", CreatedAt: createdAt}
}

func hostPortFixture(targetID int, createdAt time.Time) *model.HostPort {
	return &model.HostPort{TargetID: targetID, Host: "a.example.com", IP: "1.1.1.1", Port: 443, CreatedAt: createdAt}
}

func subdomainFixture(targetID int, createdAt time.Time) *model.Subdomain {
	return &model.Subdomain{TargetID: targetID, DNSName: "api.example.com", CreatedAt: createdAt}
}

func websiteFixture(targetID int, createdAt time.Time) *model.Website {
	return &model.Website{TargetID: targetID, URL: "https://example.com", Host: "example.com", CreatedAt: createdAt}
}
