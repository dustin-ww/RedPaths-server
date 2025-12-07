package util

import (
	"context"
	"sync"
)

type ExecutableAdapter interface {
	GetExecutablePath() string
	SetExecutablePath(path string)
	IsPathChecked() bool
	SetPathChecked(checked bool)
}

type ExecutableHelper struct {
	executablePath string
	pathChecked    bool
	mu             sync.Mutex
}

func NewExecutableHelper(defaultPath string) *ExecutableHelper {
	return &ExecutableHelper{
		executablePath: defaultPath,
		pathChecked:    false,
	}
}

func (h *ExecutableHelper) GetExecutablePath() string {
	return h.executablePath
}

func (h *ExecutableHelper) SetExecutablePath(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if path != "" {
		h.executablePath = path
		h.pathChecked = false
	}
}

func (h *ExecutableHelper) IsPathChecked() bool {
	return h.pathChecked
}

func (h *ExecutableHelper) SetPathChecked(checked bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pathChecked = checked
}

func ExecWithFallback(ctx context.Context, adapter ExecutableAdapter, defaultPath string, args ...string) ([]byte, error) {
	/*var execPath string

	if !adapter.IsPathChecked() {
		configuredPath := adapter.GetExecutablePath()

		cmd := exec.CommandContext(ctx, configuredPath, "--version")
		err := cmd.Run()

		if err != nil {
			log.Printf("Configured Path '%s' is not found, trying fallback cmd '%s'", configuredPath, defaultPath)
			fallbackCmd := exec.CommandContext(ctx, defaultPath, "--version")
			if fallbackCmd.Run() == nil {
				log.Printf("Using '%s' as Fallback", defaultPath)
				adapter.SetExecutablePath(defaultPath)
			}
		}

		adapter.SetPathChecked(true)
	}
	*/
	/*execPath = adapter.GetExecutablePath()

	cmd := exec.CommandContext(ctx, execPath, args...)*/
	//return cmd.CombinedOtput()
	return nil, nil
}

func ExecWithFallbackSimple(adapter ExecutableAdapter, defaultPath string, args ...string) ([]byte, error) {
	return ExecWithFallback(context.Background(), adapter, defaultPath, args...)
}
