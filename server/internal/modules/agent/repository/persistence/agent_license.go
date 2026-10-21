package model

// AgentLicense keeps the signed entitlement as the only mutable authorization source.
type AgentLicense struct {
	Singleton bool   `gorm:"column:singleton;primaryKey;autoIncrement:false"`
	SignedKey string `gorm:"column:signed_key;not null"`
}

// TableName identifies the deployment-wide singleton entitlement table.
func (AgentLicense) TableName() string { return "agent_license" }
