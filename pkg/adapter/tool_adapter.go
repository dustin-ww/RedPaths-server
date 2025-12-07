package adapter

import (
	"RedPaths-server/pkg/model"
	"context"
	"time"
)

type ToolAdapter interface {
	GetName() string
	GetVersion() string
	IsAvailable(ctx context.Context) bool
}

type ScanAdapter interface {
	ToolAdapter

	Scan(ctx context.Context, options ...ScanOption) (ScanResult, error)
}

type ScanResult interface {
	GetRawOutput() []byte
	GetHosts() []model.Host
	GetServices() []model.Service
}

type ScanOption func(interface{})

func WithTimeout(duration time.Duration) ScanOption {
	return func(opts interface{}) {
		if scanOpts, ok := opts.(*ScanOptions); ok {
			scanOpts.Timeout = duration
		}
	}
}

func WithOutputFormat(format string) ScanOption {
	return func(opts interface{}) {
		if scanOpts, ok := opts.(*ScanOptions); ok {
			scanOpts.OutputFormat = format
		}
	}
}

type ScanOptions struct {
	Targets      []string
	Timeout      time.Duration
	OutputFormat string
	CustomFlags  []string
}
