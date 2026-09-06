package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	identitydomain "github.com/yyhuni/lunafox/server/internal/modules/identity/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/identity/repository/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMCPKeyRepositoryReplaceKeepsOneActiveDigestPerUser(t *testing.T) {
	db := newMCPKeyRepositoryTestDB(t)
	repo := NewMCPKeyRepository(db)

	first, err := repo.Replace(context.Background(), 7, strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("first replace: %v", err)
	}
	if first.ID == 0 || first.UserID != 7 || first.KeyDigest != strings.Repeat("a", 64) {
		t.Fatalf("unexpected first key: %+v", first)
	}
	firstCreatedAt := first.CreatedAt

	resolved, err := repo.GetByDigest(context.Background(), first.KeyDigest)
	if err != nil || resolved.ID != first.ID {
		t.Fatalf("lookup first digest = %+v, %v", resolved, err)
	}

	second, err := repo.Replace(context.Background(), 7, strings.Repeat("b", 64))
	if err != nil {
		t.Fatalf("rotation replace: %v", err)
	}
	if second.ID != first.ID || second.KeyDigest == first.KeyDigest || !second.CreatedAt.Equal(firstCreatedAt) {
		t.Fatalf("rotation did not update one row atomically: first=%+v second=%+v", first, second)
	}
	if _, err := repo.GetByDigest(context.Background(), first.KeyDigest); !errors.Is(err, identitydomain.ErrMCPKeyNotFound) {
		t.Fatalf("old digest lookup error = %v", err)
	}
	if got, err := repo.GetByUserID(context.Background(), 7); err != nil || got.KeyDigest != second.KeyDigest {
		t.Fatalf("current user digest = %+v, %v", got, err)
	}

	var count int64
	if err := db.Model(&model.MCPKey{}).Where("user_id = ?", 7).Count(&count).Error; err != nil {
		t.Fatalf("count active keys: %v", err)
	}
	if count != 1 {
		t.Fatalf("active key count = %d, want 1", count)
	}
}

func TestMCPKeyRepositoryEnforcesDigestUniquenessAndContext(t *testing.T) {
	db := newMCPKeyRepositoryTestDB(t)
	repo := NewMCPKeyRepository(db)
	digest := strings.Repeat("c", 64)
	if _, err := repo.Replace(context.Background(), 1, digest); err != nil {
		t.Fatalf("first digest insert: %v", err)
	}
	if _, err := repo.Replace(context.Background(), 2, digest); err == nil {
		t.Fatal("duplicate digest unexpectedly accepted for another user")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repo.GetByUserID(ctx, 1); err == nil {
		t.Fatal("cancelled lookup unexpectedly succeeded")
	}
	if _, err := repo.GetByDigest(context.Background(), strings.Repeat("d", 64)); !errors.Is(err, identitydomain.ErrMCPKeyNotFound) {
		t.Fatalf("missing digest error = %v", err)
	}
}

func TestMCPKeyPersistenceHasNoPlaintextField(t *testing.T) {
	typ := fmt.Sprintf("%T", model.MCPKey{})
	if strings.Contains(strings.ToLower(typ), "secret") {
		t.Fatalf("persistence type has secret-like name: %s", typ)
	}
	if _, ok := reflect.TypeOf(model.MCPKey{}).FieldByName("Secret"); ok {
		t.Fatal("MCP key persistence model contains plaintext Secret field")
	}
	if _, ok := reflect.TypeOf(model.MCPKey{}).FieldByName("Key"); ok {
		t.Fatal("MCP key persistence model contains plaintext Key field")
	}
}

func newMCPKeyRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:identity-mcp-key-%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sqlite db failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec(`CREATE TABLE mcp_key (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL UNIQUE,
		key_digest CHAR(64) NOT NULL UNIQUE,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create mcp_key table failed: %v", err)
	}
	return db
}
