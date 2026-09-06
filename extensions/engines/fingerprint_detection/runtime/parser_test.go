package fingerprintdetectionruntime

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseObserverWardResultUsesObservedTargetAndNames(t *testing.T) {
	result, err := parseObserverWardResult([]byte(`{"input_target":"https://example.com/app#fragment","target":"https://example.com","success":true,"matched":[{"base_url":"https://example.com/login","result":{"status":200,"name":["nginx","nginx","Vue"]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.InputTarget != "https://example.com/app#fragment" || result.Target != "https://example.com" || !result.Success {
		t.Fatalf("result = %#v", result)
	}
	if !reflect.DeepEqual(result.Matched[0].Result.Names, []string{"nginx", "nginx", "Vue"}) {
		t.Fatalf("names = %#v", result.Matched[0].Result.Names)
	}
}

func TestParseObserverWardResultAllowsOptionalInputTarget(t *testing.T) {
	for _, payload := range []string{
		`{"target":"https://example.com","success":true,"matched":[]}`,
		`{"input_target":null,"target":"https://example.com","success":true,"matched":[]}`,
		`{"input_target":3,"target":"https://example.com","success":true,"matched":[]}`,
		`{"input_target":"","target":"https://example.com","success":true,"matched":[]}`,
		`{"input_target":true,"target":"https://example.com","success":true,"matched":[]}`,
	} {
		if _, err := parseObserverWardResult([]byte(payload)); err != nil {
			t.Fatalf("payload %s rejected despite valid target: %v", payload, err)
		}
	}
}

func TestParseObserverWardResultRequiresObservedTarget(t *testing.T) {
	for _, payload := range []string{
		`{"input_target":"https://example.com","success":true,"matched":[]}`,
		`{"input_target":"https://example.com","target":null,"success":true,"matched":[]}`,
		`{"input_target":"https://example.com","target":3,"success":true,"matched":[]}`,
		`{"input_target":"https://example.com","target":"","success":true,"matched":[]}`,
	} {
		_, err := parseObserverWardResult([]byte(payload))
		if !errors.Is(err, errObserverWardMissingTarget) {
			t.Fatalf("payload %s error = %v, want missing target", payload, err)
		}
	}
}

func TestTrustedObserverWardTechnologyRequiresConcreteStatus(t *testing.T) {
	withoutStatus, err := parseObserverWardResult([]byte(`{"target":"https://example.com","success":true,"matched":[{"base_url":"https://example.com","result":{"name":["nginx"]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	trusted, _ := trustedObserverWardTechnology(withoutStatus)
	if trusted {
		t.Fatal("missing status was treated as trusted")
	}
	withStatus, err := parseObserverWardResult([]byte(`{"target":"https://example.com","success":false,"matched":[{"base_url":"https://example.com","result":{"status":503,"name":["nginx"]}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	trusted, names := trustedObserverWardTechnology(withStatus)
	if !trusted || !reflect.DeepEqual(names, []string{"nginx"}) {
		t.Fatalf("trusted = %v names=%#v", trusted, names)
	}
}

func TestParseObserverWardResultRejectsMalformedProtocol(t *testing.T) {
	for _, payload := range []string{
		`{"target":"https://example.com","success":true,"matched":null}`,
		`{"target":"https://example.com","success":"true","matched":[]}`,
		`{"target":"https://example.com","success":true,"matched":[{"base_url":"https://example.com","result":{"status":"200"}}]}`,
		`{"target":"https://example.com","target":"https://other.example","success":true,"matched":[]}`,
		`{"target":"https://example.com","success":true,"matched":[],"unknown":true}`,
	} {
		if _, err := parseObserverWardResult([]byte(payload)); err == nil {
			t.Fatalf("malformed payload accepted: %s", payload)
		}
	}
}

func TestParseObserverWardResultRejectsInvalidText(t *testing.T) {
	payload := append([]byte(`{"target":"https://example.com/`), 0xff)
	payload = append(payload, []byte(`","success":true,"matched":[]}`)...)
	if _, err := parseObserverWardResult(payload); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}
