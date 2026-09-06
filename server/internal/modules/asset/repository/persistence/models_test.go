package model

import "testing"

func TestTableNamesAndBeforeCreate(t *testing.T) {
	tests := map[string]string{
		Directory{}.TableName():      "directory",
		Subdomain{}.TableName():      "subdomain",
		Website{}.TableName():        "website",
		Endpoint{}.TableName():       "endpoint",
		Screenshot{}.TableName():     "screenshot",
		AssetTargetRef{}.TableName(): "target",
		HostPort{}.TableName():       "host_port_mapping",
	}

	for got, want := range tests {
		if got != want {
			t.Fatalf("expected table name %q, got %q", want, got)
		}
	}

	website := &Website{}
	if err := website.BeforeCreate(nil); err != nil {
		t.Fatalf("website BeforeCreate failed: %v", err)
	}
	if website.Tech == nil || len(website.Tech) != 0 {
		t.Fatalf("expected website tech to be initialized, got %#v", website.Tech)
	}

	endpoint := &Endpoint{}
	if err := endpoint.BeforeCreate(nil); err != nil {
		t.Fatalf("endpoint BeforeCreate failed: %v", err)
	}
	if endpoint.Tech == nil || len(endpoint.Tech) != 0 {
		t.Fatalf("expected endpoint tech to be initialized, got %#v", endpoint.Tech)
	}
}
