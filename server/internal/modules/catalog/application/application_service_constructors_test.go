package application

import "testing"

func TestCatalogApplicationConstructorsExist(t *testing.T) {
	if NewSubfinderAPIKeySettingsService(nil) == nil {
		t.Fatalf("expected subfinder api key settings service")
	}
}
