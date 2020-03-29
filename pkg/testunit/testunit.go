package testunit

import (
	"context"

	"github.com/google/uuid"
)

// TestUnit type
type TestUnit interface {
	// ID returns identifier
	ID() string
	// Enabled?
	Enabled() (bool, string)
	// Prepare runs prior to Run(context.Context, []interface{}) error
	Prepare(context.Context) ([]interface{}, error)
	// Test is the main run func
	Test(context.Context, []interface{}) ([]interface{}, error)
	// Cleanup runs after Run(context.Context, []interface{}) error
	Cleanup(context.Context, []interface{}) error
}

// Config type
type Config struct {
	Enabled func() (bool, string)
	ID      string
	Prepare func(context.Context) ([]interface{}, error)
	Test    func(context.Context, []interface{}) ([]interface{}, error)
	Cleanup func(context.Context, []interface{}) error
}

// Option type
type Option func(*Config)

// Enabled adds enabled func
func Enabled(f func() (bool, string)) Option {
	return func(cfg *Config) {
		cfg.Enabled = f
	}
}

// ID adds an id
func ID(id string) Option {
	return func(cfg *Config) {
		cfg.ID = id
	}
}

// Prepare adds a prepare func
func Prepare(f func(context.Context) ([]interface{}, error)) Option {
	return func(cfg *Config) {
		cfg.Prepare = f
	}
}

// Test adds a test func
func Test(f func(context.Context, []interface{}) ([]interface{}, error)) Option {
	return func(cfg *Config) {
		cfg.Test = f
	}
}

// Cleanup adds a cleanup func
func Cleanup(f func(context.Context, []interface{}) error) Option {
	return func(cfg *Config) {
		cfg.Cleanup = f
	}
}

// TestAssembly type
type TestAssembly struct {
	cfg *Config
}

// New constructs a new test
func New(options ...Option) *TestAssembly {
	cfg := &Config{
		ID:      uuid.New().String(),
		Enabled: func() (bool, string) { return true, "" },
		Prepare: func(context.Context) ([]interface{}, error) { return nil, nil },
		Test:    func(context.Context, []interface{}) ([]interface{}, error) { return nil, nil },
		Cleanup: func(context.Context, []interface{}) error { return nil },
	}

	for _, opt := range options {
		opt(cfg)
	}
	return &TestAssembly{cfg: cfg}
}

// ID returns identifier
func (test *TestAssembly) ID() string {
	return test.cfg.ID
}

// Enabled return if test is enabled?
func (test *TestAssembly) Enabled() (bool, string) {
	return test.cfg.Enabled()
}

// Prepare runs prior to Test(context.Context, []interface{}) error
func (test *TestAssembly) Prepare(ctx context.Context) ([]interface{}, error) {
	return test.cfg.Prepare(ctx)
}

// Test is the main test func
func (test *TestAssembly) Test(ctx context.Context, args []interface{}) ([]interface{}, error) {
	return test.cfg.Test(ctx, args)
}

// Cleanup runs after Test(context.Context, []interface{}) error
func (test *TestAssembly) Cleanup(ctx context.Context, args []interface{}) error {
	return test.cfg.Cleanup(ctx, args)
}
