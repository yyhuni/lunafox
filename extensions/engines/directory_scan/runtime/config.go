package directoryscanruntime

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

var (
	asciiIntegerPattern = regexp.MustCompile(`^[0-9]+$`)
	asciiDecimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)
	maximumDelaySeconds = big.NewRat(120, 1)
)

// ValidateConfig defensively validates the complete generated FFUF config
// before Directory reads candidate inputs or starts a scanner process.
func ValidateConfig(config enginecontract.FfufConfig) error {
	if !config.Enabled {
		return fmt.Errorf("ffuf section must be enabled")
	}
	if strings.TrimSpace(config.Wordlist) == "" {
		return invalidFFUFConfig("wordlist")
	}
	if config.RecursionDepth < 1 || config.RecursionDepth > 5 {
		return invalidFFUFConfig("recursion-depth")
	}
	if config.RecursionStrategy != "default" && config.RecursionStrategy != "greedy" {
		return invalidFFUFConfig("recursion-strategy")
	}
	if config.AutoCalibrationMode != "ac" && config.AutoCalibrationMode != "ach" {
		return invalidFFUFConfig("auto-calibration-mode")
	}
	if err := validateMatchCodes(config.MatchCodes); err != nil {
		return err
	}
	if config.Concurrency < 1 || config.Concurrency > 20 {
		return invalidFFUFConfig("concurrency")
	}
	if config.Threads < 1 || config.Threads > 100 {
		return invalidFFUFConfig("threads")
	}
	if config.Rate < 0 || config.Rate > 1000 {
		return invalidFFUFConfig("rate")
	}
	if err := validateDelay(config.Delay); err != nil {
		return err
	}
	if config.RequestTimeout < 1 || config.RequestTimeout > 120 {
		return invalidFFUFConfig("request-timeout")
	}
	if config.Timeout < 60 || config.Timeout > 604800 {
		return invalidFFUFConfig("timeout")
	}
	return nil
}

func validateMatchCodes(raw string) error {
	if raw == "" {
		return invalidFFUFConfig("match-codes")
	}
	for _, token := range strings.Split(raw, ",") {
		if token == "" {
			return invalidFFUFConfig("match-codes")
		}
		if token == "all" {
			continue
		}
		if strings.Count(token, "-") == 0 {
			if _, err := parseMatchCode(token); err != nil {
				return invalidFFUFConfig("match-codes")
			}
			continue
		}
		if strings.Count(token, "-") != 1 {
			return invalidFFUFConfig("match-codes")
		}
		startRaw, endRaw, _ := strings.Cut(token, "-")
		start, startErr := parseMatchCode(startRaw)
		end, endErr := parseMatchCode(endRaw)
		if startErr != nil || endErr != nil || start >= end {
			return invalidFFUFConfig("match-codes")
		}
	}
	return nil
}

func parseMatchCode(raw string) (uint64, error) {
	if !asciiIntegerPattern.MatchString(raw) {
		return 0, fmt.Errorf("match code must contain ASCII digits")
	}
	value, err := strconv.ParseUint(raw, 10, 16)
	if err != nil || value < 100 || value > 999 {
		return 0, fmt.Errorf("match code must be between 100 and 999")
	}
	return value, nil
}

func validateDelay(raw string) error {
	parts := strings.Split(raw, "-")
	if len(parts) < 1 || len(parts) > 2 {
		return invalidFFUFConfig("delay")
	}
	values := make([]*big.Rat, len(parts))
	for index, part := range parts {
		if !asciiDecimalPattern.MatchString(part) {
			return invalidFFUFConfig("delay")
		}
		value, ok := new(big.Rat).SetString(part)
		if !ok || value.Sign() < 0 || value.Cmp(maximumDelaySeconds) > 0 {
			return invalidFFUFConfig("delay")
		}
		values[index] = value
	}
	if len(values) == 2 && values[0].Cmp(values[1]) > 0 {
		return invalidFFUFConfig("delay")
	}
	return nil
}

func invalidFFUFConfig(field string) error {
	return fmt.Errorf("ffuf.%s is invalid", field)
}
