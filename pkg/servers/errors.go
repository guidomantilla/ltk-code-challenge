package servers

import (
	"fmt"
)

func ErrServerFailedToStart(name string, err error) error {
	return fmt.Errorf("server %s failed to start: %w", name, err)
}

func ErrServerFailedToStop(name string, err error) error {
	return fmt.Errorf("server %s failed to stop: %w", name, err)
}
