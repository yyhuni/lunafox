package domain

import "time"

// TargetRef is a local projection used by asset context.
type TargetRef struct {
	ID            int
	Name          string
	Type          string
	CreatedAt     time.Time
	LastScannedAt *time.Time
	DeletedAt     *time.Time
}

type Website struct {
	ID              int
	TargetID        int
	URL             string
	Host            string
	Location        string
	CreatedAt       time.Time
	Title           string
	Webserver       string
	ResponseBody    string
	ContentType     string
	Tech            []string
	StatusCode      *int
	ContentLength   *int
	Vhost           *bool
	ResponseHeaders string
}

type Endpoint struct {
	ID                int
	TargetID          int
	URL               string
	Host              string
	Location          string
	CreatedAt         time.Time
	Title             string
	Webserver         string
	ResponseBody      string
	ContentType       string
	Tech              []string
	StatusCode        *int
	ContentLength     *int
	Vhost             *bool
	ResponseHeaders   string
}

type Directory struct {
	ID            int
	TargetID      int
	URL           string
	Status        *int
	ContentLength *int
	ContentType   string
	Duration      *int
	CreatedAt     time.Time
}

type Subdomain struct {
	ID        int
	TargetID  int
	Name      string
	CreatedAt time.Time
}

type HostPort struct {
	ID        int
	TargetID  int
	Host      string
	IP        string
	Port      int
	CreatedAt time.Time
}

type Screenshot struct {
	ID         int
	TargetID   int
	URL        string
	StatusCode *int16
	Image      []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type IPAggregationRow struct {
	IP        string
	CreatedAt time.Time
}
