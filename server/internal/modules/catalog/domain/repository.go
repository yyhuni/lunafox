package domain

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
	BulkSoftDelete(ids []int) (int64, error)
	BulkCreateIgnoreConflicts(targets []Target) (int, error)
	FindByNames(names []string) ([]Target, error)
}

// TargetQueryRepository defines query-side persistence port for target use cases.
type TargetQueryRepository interface {
	GetActiveByID(id int) (*Target, error)
	FindAll(page, pageSize int, targetType, filter string) ([]Target, int64, error)
	GetAssetCounts(targetID int) (*TargetAssetCounts, error)
	GetVulnerabilityCounts(targetID int) (*VulnerabilityCounts, error)
}

// WordlistCommandRepository defines command-side persistence for wordlists.
type WordlistCommandRepository interface {
	GetByID(id int) (*Wordlist, error)
	ExistsByName(name string) (bool, error)
	Create(wordlist *Wordlist) error
	Update(wordlist *Wordlist) error
	Delete(id int) error
}

// WordlistQueryRepository defines query-side persistence for wordlists.
type WordlistQueryRepository interface {
	FindAll(page, pageSize int) ([]Wordlist, int64, error)
	List() ([]Wordlist, error)
	GetByID(id int) (*Wordlist, error)
	FindByName(name string) (*Wordlist, error)
	Update(wordlist *Wordlist) error
}
