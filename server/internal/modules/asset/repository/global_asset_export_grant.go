package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	assetapp "github.com/yyhuni/lunafox/server/internal/modules/asset/application"
)

const globalAssetExportGrantKeyPrefix = "lunafox:asset-export:grant:"

type GlobalAssetExportGrantStore struct {
	client *redis.Client
	now    func() time.Time
}

func NewGlobalAssetExportGrantStore(client *redis.Client) *GlobalAssetExportGrantStore {
	return &GlobalAssetExportGrantStore{client: client, now: time.Now}
}

func (store *GlobalAssetExportGrantStore) Issue(ctx context.Context, grant assetapp.GlobalAssetExportGrant, ttl time.Duration) (string, error) {
	if store == nil || store.client == nil || ttl <= 0 {
		return "", errors.New("global asset export grant store unavailable")
	}
	token, err := assetapp.NewOpaqueExportToken()
	if err != nil {
		return "", err
	}
	grantBytes, err := assetapp.MarshalExportGrant(grant)
	if err != nil {
		return "", err
	}
	return token, store.client.Set(ctx, globalAssetExportKey(token), grantBytes, ttl).Err()
}

func (store *GlobalAssetExportGrantStore) Consume(ctx context.Context, token string) (*assetapp.GlobalAssetExportGrant, error) {
	if store == nil || store.client == nil || token == "" {
		return nil, assetapp.ErrGlobalAssetExportGrant
	}
	value, err := consumeGlobalAssetExportScript.Run(ctx, store.client, []string{globalAssetExportKey(token)}).Result()
	if err != nil || value == nil {
		return nil, assetapp.ErrGlobalAssetExportGrant
	}
	var raw string
	switch item := value.(type) {
	case string:
		raw = item
	case []byte:
		raw = string(item)
	default:
		return nil, assetapp.ErrGlobalAssetExportGrant
	}
	var grant assetapp.GlobalAssetExportGrant
	if err := json.Unmarshal([]byte(raw), &grant); err != nil {
		return nil, assetapp.ErrGlobalAssetExportGrant
	}
	if !grant.ExpiresAt.After(store.now().UTC()) {
		return nil, assetapp.ErrGlobalAssetExportGrant
	}
	return &grant, nil
}

var consumeGlobalAssetExportScript = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if not value then return false end
redis.call('DEL', KEYS[1])
return value
`)

func globalAssetExportKey(token string) string {
	digest := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%s%s", globalAssetExportGrantKeyPrefix, hex.EncodeToString(digest[:]))
}
