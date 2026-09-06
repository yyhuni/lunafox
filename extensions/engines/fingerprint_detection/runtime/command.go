package fingerprintdetectionruntime

import (
	"fmt"
	"strconv"
	"time"

	enginecontract "github.com/yyhuni/lunafox/engines/fingerprint_detection/contract"
)

type observerWardCommand struct {
	Name    string
	Args    []string
	Timeout time.Duration
}

func validateObserverWardConfig(config enginecontract.ObserverWardConfig) error {
	if config.Timeout < 60 || config.Timeout > 604800 {
		return fmt.Errorf("timeout must be between 60 and 604800 seconds")
	}
	if config.RequestTimeout < 1 || config.RequestTimeout > 120 {
		return fmt.Errorf("request-timeout must be between 1 and 120 seconds")
	}
	if config.Threads < 1 || config.Threads > 200 {
		return fmt.Errorf("threads must be between 1 and 200")
	}
	return nil
}

func buildObserverWardCommand(candidatePath, probePath string, config enginecontract.ObserverWardConfig) (observerWardCommand, error) {
	if err := validateObserverWardConfig(config); err != nil {
		return observerWardCommand{}, err
	}
	if candidatePath == "" || probePath == "" {
		return observerWardCommand{}, fmt.Errorf("Observer Ward candidate and probe paths are required")
	}
	return observerWardCommand{
		Name: "observer-ward",
		Args: []string{
			"-l", candidatePath,
			"-p", probePath,
			"--mode", "http",
			"--timeout", strconv.FormatInt(config.RequestTimeout, 10),
			"--thread", strconv.FormatInt(config.Threads, 10),
			"--format", "json",
			"--silent",
			"--no-color",
		},
		Timeout: time.Duration(config.Timeout) * time.Second,
	}, nil
}
