package domain

import "context"

// TargetAssetCounts holds asset count statistics for a target.
type TargetAssetCounts struct {
	Subdomains  int64
	Websites    int64
	Endpoints   int64
	IPs         int64
	Directories int64
	Screenshots int64
}

// VulnerabilityCounts holds vulnerability count statistics by severity.
type VulnerabilityCounts struct {
	Total    int64
	Critical int64
	High     int64
	Medium   int64
	Low      int64
}

// TargetCommandRepository defines command-side persistence port for target use cases.
type TargetCommandRepository interface {
	GetActiveByID(id int) (*Target, error)
	ExistsByName(name string, excludeID ...int) (bool, error)
	Create(target *Target) error
	Update(target *Target) error
	SoftDelete(id int) error
	BatchSoftDelete(ids []int) (int64, error)
	BatchCreateIgnoreConflicts(targets []Target) (int, error)
	FindByNames(names []string) ([]Target, error)
}

// TargetQueryRepository defines query-side persistence port for target use cases.
type TargetQueryRepository interface {
	GetActiveByID(id int) (*Target, error)
	List(page, pageSize int, filter, orderBy string) ([]Target, int64, error)
	GetAssetCountsSummary(targetID int) (*TargetAssetCounts, error)
	GetVulnerabilityCountsSummary(targetID int) (*VulnerabilityCounts, error)
}

// WordlistCommandRepository defines command-side persistence for wordlists.
type WordlistCommandRepository interface {
	GetByID(id int) (*Wordlist, error)
	ExistsByFileName(fileName string, excludeID ...int) (bool, error)
	Create(wordlist *Wordlist) error
	Update(wordlist *Wordlist) error
	Delete(id int) error
}

// WordlistQueryRepository defines query-side persistence for wordlists.
type WordlistQueryRepository interface {
	List(page, pageSize int, filter, orderBy string) ([]Wordlist, int64, error)
	ListAll() ([]Wordlist, error)
	ListTagSummaries(page, pageSize int, filter string) ([]WordlistTagSummary, int64, error)
	GetByID(id int) (*Wordlist, error)
	Update(wordlist *Wordlist) error
}

// InstalledEngineQueryRepository exposes only complete current engine records.
type InstalledEngineQueryRepository interface {
	ListInstalledEngines() ([]Engine, error)
	GetInstalledEngineByID(engineID string) (*Engine, error)
}

// InstalledEngineCommandRepository atomically replaces an engine's current package.
type InstalledEngineCommandRepository interface {
	UpsertInstalledEngine(ctx context.Context, engine *Engine, allowReplacement bool) error
}

// EngineReplacementConflictError reports an attempted current-package change
// that was fully validated but not explicitly approved by its caller.
type EngineReplacementConflictError struct {
	EngineID              string
	CurrentPackageDigest  string
	ProposedPackageDigest string
}

func (err *EngineReplacementConflictError) Error() string {
	return "installed engine " + err.EngineID + " has a different current package digest"
}
