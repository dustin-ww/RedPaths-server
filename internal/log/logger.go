package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

func InitLogger() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	logPath := filepath.Join(cwd, "app.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, file)
	slog.SetDefault(slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	return nil
}
