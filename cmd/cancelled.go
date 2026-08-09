package cmd

import (
	"context"
	"errors"
	"fmt"
)

func describeCancellation(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("interrompido")
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("tempo esgotado")
	default:
		return err
	}
}
