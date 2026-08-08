package cmd

import "io"

func writeAndClose(destination io.WriteCloser, write func(io.Writer) error) error {
	err := write(destination)
	if closed := destination.Close(); err == nil {
		err = closed
	}

	return err
}
