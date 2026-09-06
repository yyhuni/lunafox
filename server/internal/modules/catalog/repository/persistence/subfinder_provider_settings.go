package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// SubfinderProviderSettings stores API keys for subfinder data sources.
type SubfinderProviderSettings struct {
	ID        int                      `gorm:"primaryKey" json:"id"`
	Providers SubfinderProviderConfigs `gorm:"column:providers;type:jsonb" json:"providers"`
	CreatedAt time.Time                `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time                `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (SubfinderProviderSettings) TableName() string {
	return "subfinder_provider_settings"
}

// SubfinderProviderConfigs maps provider name to its configuration.
type SubfinderProviderConfigs map[string]SubfinderProviderConfig

// SubfinderProviderConfig holds credentials for a single provider.
type SubfinderProviderConfig struct {
	Enabled             bool              `json:"enabled"`
	Status              string            `json:"status,omitempty"`
	Values              map[string]string `json:"values,omitempty"`
	Email               string            `json:"email,omitempty"`
	APIKey              string            `json:"api_key,omitempty"`
	AuthenticationToken string            `json:"authentication_token,omitempty"`
	APIID               string            `json:"api_id,omitempty"`
	APISecret           string            `json:"api_secret,omitempty"`
}

func (p *SubfinderProviderConfigs) Scan(value any) error {
	if value == nil {
		*p = make(SubfinderProviderConfigs)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("SubfinderProviderConfigs: expected []byte from database")
	}
	return json.Unmarshal(bytes, p)
}

func (p SubfinderProviderConfigs) Value() (driver.Value, error) {
	return json.Marshal(p)
}
