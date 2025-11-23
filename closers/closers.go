package closers

import (
	"io"
	"log/slog"
)

func CloseOrLog(log *slog.Logger, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Error("failed to close", "error", err)
	}
}

func CloseOrPanic(c io.Closer) {
	if err := c.Close(); err != nil {
		panic(err)
	}
}
