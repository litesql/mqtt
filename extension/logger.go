package extension

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/litesql/mqtt/config"
)

func loggerFromConfig(option string) (*slog.Logger, io.Closer, error) {
	var closer io.Closer
	var w io.Writer
	switch option {
	case "stdout":
		w = os.Stdout
	case "stderr":
		w = os.Stderr
	case "":
		w = io.Discard
	default:
		if !strings.HasPrefix(option, "file:") {
			return nil, nil, fmt.Errorf("invalid %q option, use stdout, stderr or file:filepath", config.Logger)
		}
		filename := strings.TrimPrefix(option, "file:")
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("opening/creating file %q: %w", filename, err)
		}
		closer = file
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	})), closer, nil
}
