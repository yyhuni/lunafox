package directoryscanruntime

import (
	"strconv"

	enginecontract "github.com/yyhuni/lunafox/engines/directory_scan/contract"
)

const FFUFBinary = "ffuf"

// BuildFFUFArgs maps only the closed manifest surface. The caller passes this
// slice directly to exec.CommandContext; no shell or inherited flag string is
// involved.
func BuildFFUFArgs(candidate string, config enginecontract.FfufConfig) ([]string, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, err
	}
	target, err := normalizeDirectoryTarget(candidate)
	if err != nil {
		return nil, err
	}
	args := []string{
		"-u", target + "FUZZ",
		"-w", config.Wordlist,
		"-json",
		"-s",
		"-X", "GET",
		"-raw",
	}
	if config.Recursion {
		args = append(args,
			"-recursion",
			"-recursion-depth", strconv.FormatInt(config.RecursionDepth, 10),
			"-recursion-strategy", config.RecursionStrategy,
		)
	}
	if config.AutoCalibration {
		if config.AutoCalibrationMode == "ach" {
			args = append(args, "-ach")
		} else {
			args = append(args, "-ac")
		}
		// Built-in FFUF calibration strategies are used on purpose: they are
		// randomized per invocation and split across independent strategy
		// groups, so a target-specific WAF rule cannot disable every group like
		// it could for one fixed Engine-wide calibration path.
	}
	args = append(args,
		"-mc", config.MatchCodes,
		"-t", strconv.FormatInt(config.Threads, 10),
		"-rate", strconv.FormatInt(config.Rate, 10),
		"-p", config.Delay,
		"-timeout", strconv.FormatInt(config.RequestTimeout, 10),
	)
	if config.FollowRedirects {
		args = append(args, "-r")
	}
	if config.Http2 {
		args = append(args, "-http2")
	}
	for _, header := range config.Headers {
		args = append(args, "-H", header)
	}
	return args, nil
}
