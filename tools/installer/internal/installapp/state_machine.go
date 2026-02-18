package installapp

import "fmt"

func transition(from InstallState, to InstallState) error {
	switch from {
	case StateIdle:
		if to == StateRunning {
			return nil
		}
	case StateRunning:
		if to == StateSucceeded || to == StateFailed || to == StateCancelled {
			return nil
		}
	case StateSucceeded, StateFailed, StateCancelled:
		if to == StateRunning {
			return nil
		}
	}
	return fmt.Errorf("invalid state transition: %s -> %s", from, to)
}
