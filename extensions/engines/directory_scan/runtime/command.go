package directoryscanruntime

import (
	"errors"
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
	if candidate == "" {
		return nil, errors.New("FFUF candidate is required")
	}
	args := []string{
		"-u", candidate + "FUZZ",
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
	return args, nil
}
