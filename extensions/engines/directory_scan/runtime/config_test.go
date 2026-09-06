package directoryscanruntime

import (
	"testing"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

func TestValidateConfigAcceptsCanonicalDefaults(t *testing.T) {
	config := defaultFFUFConfig()
	if err := ValidateConfig(config); err != nil {
		t.Fatalf("ValidateConfig(defaults) error = %v", err)
	}
	if config.Wordlist != "/run/lunafox/resources/config/ffuf/wordlist/dir_default.txt" ||
		config.Recursion || config.RecursionDepth != 1 || config.RecursionStrategy != "default" ||
		!config.AutoCalibration || config.AutoCalibrationMode != "ac" ||
		config.MatchCodes != "200-299,301,302,307,401,403,405,500" ||
		config.Concurrency != 5 || config.Threads != 10 || config.Rate != 0 || config.Delay != "0.1-2.0" ||
		config.RequestTimeout != 10 || config.Timeout != 86400 || config.FollowRedirects || config.Http2 {
		t.Fatalf("default fixture drifted: %#v", config)
	}
}

func TestValidateConfigAcceptsIndependentBoundariesAndBooleanModes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*enginecontract.FfufConfig)
	}{
		{name: "minimums", mutate: func(config *enginecontract.FfufConfig) {
			config.RecursionDepth = 1
			config.Concurrency = 1
			config.Threads = 1
			config.Rate = 0
			config.RequestTimeout = 1
			config.Timeout = 60
		}},
		{name: "maximums", mutate: func(config *enginecontract.FfufConfig) {
			config.RecursionDepth = 5
			config.Concurrency = 20
			config.Threads = 100
			config.Rate = 1000
			config.RequestTimeout = 120
			config.Timeout = 604800
		}},
		{name: "alternate enums", mutate: func(config *enginecontract.FfufConfig) {
			config.RecursionStrategy = "greedy"
			config.AutoCalibrationMode = "ach"
		}},
		{name: "boolean modes", mutate: func(config *enginecontract.FfufConfig) {
			config.Recursion = true
			config.AutoCalibration = false
			config.FollowRedirects = true
			config.Http2 = true
		}},
		{name: "maximum concurrency and threads remain independent", mutate: func(config *enginecontract.FfufConfig) {
			config.Concurrency = 20
			config.Threads = 100
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := defaultFFUFConfig()
			test.mutate(&config)
			if err := ValidateConfig(config); err != nil {
				t.Fatalf("ValidateConfig() error = %v", err)
			}
		})
	}
}

func TestValidateConfigRejectsDisabledMissingAndOutOfRangeValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*enginecontract.FfufConfig)
	}{
		{name: "disabled section", mutate: func(config *enginecontract.FfufConfig) { config.Enabled = false }},
		{name: "empty wordlist", mutate: func(config *enginecontract.FfufConfig) { config.Wordlist = "" }},
		{name: "blank wordlist", mutate: func(config *enginecontract.FfufConfig) { config.Wordlist = " \t" }},
		{name: "recursion depth below", mutate: func(config *enginecontract.FfufConfig) { config.RecursionDepth = 0 }},
		{name: "recursion depth above", mutate: func(config *enginecontract.FfufConfig) { config.RecursionDepth = 6 }},
		{name: "recursion strategy", mutate: func(config *enginecontract.FfufConfig) { config.RecursionStrategy = "wide" }},
		{name: "calibration mode", mutate: func(config *enginecontract.FfufConfig) { config.AutoCalibrationMode = "global" }},
		{name: "concurrency below", mutate: func(config *enginecontract.FfufConfig) { config.Concurrency = 0 }},
		{name: "concurrency above", mutate: func(config *enginecontract.FfufConfig) { config.Concurrency = 21 }},
		{name: "threads below", mutate: func(config *enginecontract.FfufConfig) { config.Threads = 0 }},
		{name: "threads above", mutate: func(config *enginecontract.FfufConfig) { config.Threads = 101 }},
		{name: "rate below", mutate: func(config *enginecontract.FfufConfig) { config.Rate = -1 }},
		{name: "rate above", mutate: func(config *enginecontract.FfufConfig) { config.Rate = 1001 }},
		{name: "request timeout below", mutate: func(config *enginecontract.FfufConfig) { config.RequestTimeout = 0 }},
		{name: "request timeout above", mutate: func(config *enginecontract.FfufConfig) { config.RequestTimeout = 121 }},
		{name: "website timeout below", mutate: func(config *enginecontract.FfufConfig) { config.Timeout = 59 }},
		{name: "website timeout above", mutate: func(config *enginecontract.FfufConfig) { config.Timeout = 604801 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := defaultFFUFConfig()
			test.mutate(&config)
			if err := ValidateConfig(config); err == nil {
				t.Fatal("ValidateConfig() accepted invalid config")
			}
		})
	}
}

func TestValidateConfigPreservesAcceptedMatchCodeForms(t *testing.T) {
	for _, raw := range []string{
		"all",
		"100",
		"00100",
		"100-999",
		"all,200,200,200-299,250-350",
	} {
		t.Run(raw, func(t *testing.T) {
			config := defaultFFUFConfig()
			config.MatchCodes = raw
			if err := ValidateConfig(config); err != nil {
				t.Fatalf("ValidateConfig(%q) error = %v", raw, err)
			}
			if config.MatchCodes != raw {
				t.Fatalf("match-codes = %q, want unchanged %q", config.MatchCodes, raw)
			}
		})
	}
}

func TestValidateConfigRejectsInvalidMatchCodeForms(t *testing.T) {
	for _, raw := range []string{
		"", ",", "200,", ",200", "200,,201", "ALL", " 200", "200 ",
		"99", "1000", "200-200", "201-200", "100-", "-200", "100--200",
		"+200", "2e2", "２００",
	} {
		t.Run(raw, func(t *testing.T) {
			config := defaultFFUFConfig()
			config.MatchCodes = raw
			if err := ValidateConfig(config); err == nil {
				t.Fatalf("ValidateConfig() accepted match-codes %q", raw)
			}
		})
	}
}

func TestValidateConfigPreservesAcceptedDelayForms(t *testing.T) {
	for _, raw := range []string{
		"0", "120", "0.0", "120.000", "000.100", "0-120", "1.0-1.00", "01.000-02.00",
	} {
		t.Run(raw, func(t *testing.T) {
			config := defaultFFUFConfig()
			config.Delay = raw
			if err := ValidateConfig(config); err != nil {
				t.Fatalf("ValidateConfig(%q) error = %v", raw, err)
			}
			if config.Delay != raw {
				t.Fatalf("delay = %q, want unchanged %q", config.Delay, raw)
			}
		})
	}
}

func TestValidateConfigRejectsInvalidDelayForms(t *testing.T) {
	for _, raw := range []string{
		"", " 1", "1 ", "-1", "+1", "1e1", "0x1", "1_0", "NaN", "Inf",
		".1", "1.", "1-", "-1", "1--2", "121", "0-120.0001", "2-1", "１",
	} {
		t.Run(raw, func(t *testing.T) {
			config := defaultFFUFConfig()
			config.Delay = raw
			if err := ValidateConfig(config); err == nil {
				t.Fatalf("ValidateConfig() accepted delay %q", raw)
			}
		})
	}
}

func defaultFFUFConfig() enginecontract.FfufConfig {
	return enginecontract.FfufConfig{
		Enabled:             true,
		Wordlist:            "/run/lunafox/resources/config/ffuf/wordlist/dir_default.txt",
		Recursion:           false,
		RecursionDepth:      1,
		RecursionStrategy:   "default",
		AutoCalibration:     true,
		AutoCalibrationMode: "ac",
		MatchCodes:          "200-299,301,302,307,401,403,405,500",
		Concurrency:         5,
		Threads:             10,
		Rate:                0,
		Delay:               "0.1-2.0",
		RequestTimeout:      10,
		Timeout:             86400,
		FollowRedirects:     false,
		Http2:               false,
	}
}
