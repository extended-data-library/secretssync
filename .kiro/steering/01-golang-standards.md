# Go Coding Standards

## Language Version

- **Minimum:** Go 1.25.3 (as specified in `go.mod`)
- **Target:** Latest stable Go release
- Use modern Go features (generics, workspace mode, etc.)

## Code Organization

### Package Structure
```
secretsync/
├── cmd/secretsync/          # Application entrypoints
│   └── cmd/                 # Cobra command implementations
├── pkg/                     # Public library code
│   ├── client/              # External service clients (Vault, AWS)
│   ├── pipeline/            # Core pipeline logic
│   ├── diff/                # Secret diff computation
│   ├── discovery/           # AWS resource discovery
│   ├── driver/              # Legacy compatibility (to be refactored)
│   └── utils/               # Shared utilities
├── api/                     # Kubernetes API types (if needed)
└── tests/                   # Integration tests
    └── integration/
```

### Naming Conventions
```go
// Packages: lowercase, single word preferred
package vault
package pipeline

// Interfaces: noun or adjective
type Reader interface { }
type Runnable interface { }

// Implementations: descriptive noun
type VaultClient struct { }
type S3MergeStore struct { }

// Functions: verb or verb phrase
func ListSecrets() { }
func ComputeDiff() { }

// Constants: CamelCase or ALL_CAPS for exported
const DefaultTimeout = 30 * time.Second
const MaxRetries = 3
```

## Error Handling

### Always Handle Errors
```go
// ❌ BAD - Silent failure
result, _ := doSomething()

// ✅ GOOD - Explicit handling
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Use Error Wrapping
```go
// ✅ Wrap errors with context
if err := client.Connect(); err != nil {
    return fmt.Errorf("connecting to vault at %s: %w", addr, err)
}

// ✅ Check wrapped errors
if errors.Is(err, vault.ErrPermissionDenied) {
    // Handle specific error
}
```

### Custom Error Types (When Needed)
```go
// For errors that need structured data
type ConfigError struct {
    Path    string
    Field   string
    Message string
}

func (e *ConfigError) Error() string {
    return fmt.Sprintf("config error in %s at field %s: %s", 
        e.Path, e.Field, e.Message)
}
```

## Context Usage

### Always Accept Context
```go
// ✅ All I/O functions take context
func ListSecrets(ctx context.Context, path string) ([]string, error) {
    // Use ctx for cancellation and timeouts
}
```

### Propagate Context
```go
// ✅ Pass context through the call chain
func (p *Pipeline) Execute(ctx context.Context) error {
    if err := p.merge(ctx); err != nil {
        return err
    }
    return p.sync(ctx)
}
```

## Testing

### Test File Naming
```go
// Source: vault.go
// Tests:  vault_test.go

// Integration tests
// File: pipeline_integration_test.go
```

### Table-Driven Tests
```go
func TestVaultClient_ListSecrets(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        want    []string
        wantErr bool
    }{
        {
            name: "single level",
            path: "secret/data/app",
            want: []string{"api-key", "db-password"},
        },
        {
            name: "nested directories",
            path: "secret/data/",
            want: []string{"app/api-key", "app/db-password"},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := client.ListSecrets(context.Background(), tt.path)
            if (err != nil) != tt.wantErr {
                t.Errorf("ListSecrets() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ListSecrets() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Use Testify for Assertions
```go
import "github.com/stretchr/testify/assert"

func TestSomething(t *testing.T) {
    result, err := doSomething()
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
    assert.NotNil(t, result.Field)
}
```

### Mock External Dependencies
```go
// Use interfaces for testability
type VaultAPI interface {
    List(ctx context.Context, path string) (*vault.Secret, error)
}

// Implement mock for tests
type mockVaultAPI struct {
    listFunc func(context.Context, string) (*vault.Secret, error)
}

func (m *mockVaultAPI) List(ctx context.Context, path string) (*vault.Secret, error) {
    return m.listFunc(ctx, path)
}
```

## Concurrency

### Use Goroutines Responsibly
```go
// ✅ GOOD - Bounded concurrency
func processSecrets(secrets []Secret) error {
    sem := make(chan struct{}, 10) // Max 10 concurrent
    errs := make(chan error, len(secrets))
    
    for _, s := range secrets {
        sem <- struct{}{}
        go func(secret Secret) {
            defer func() { <-sem }()
            errs <- processSecret(secret)
        }(s)
    }
    
    // Collect errors...
}
```

### Protect Shared State
```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]CacheItem
}

func (c *Cache) Get(key string) (CacheItem, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    item, ok := c.items[key]
    return item, ok
}

func (c *Cache) Set(key string, item CacheItem) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = item
}
```

### Use Channels for Communication
```go
// ✅ Producer-consumer pattern
func producer(ctx context.Context, items <-chan Item) <-chan Result {
    results := make(chan Result)
    go func() {
        defer close(results)
        for item := range items {
            select {
            case <-ctx.Done():
                return
            case results <- process(item):
            }
        }
    }()
    return results
}
```

## Logging

### Use Structured Logging
```go
import "github.com/sirupsen/logrus"

// ✅ Structured fields
log.WithFields(logrus.Fields{
    "secret_path": path,
    "target":      targetName,
    "duration_ms": elapsed.Milliseconds(),
}).Info("secret synced successfully")
```

### Log Levels
```go
// ERROR - Actionable errors
log.WithError(err).Error("failed to sync secret")

// WARN - Degraded state but continuing
log.Warn("using cached data, API unavailable")

// INFO - Normal operations
log.Info("pipeline execution started")

// DEBUG - Detailed troubleshooting
log.Debug("received vault response", "keys", keys)
```

## Configuration

### Use Struct Tags
```go
type Config struct {
    VaultAddr    string        `yaml:"vault_addr" validate:"required"`
    Timeout      time.Duration `yaml:"timeout" default:"30s"`
    MaxRetries   int           `yaml:"max_retries" default:"3"`
}
```

### Validate Configuration
```go
func (c *Config) Validate() error {
    if c.VaultAddr == "" {
        return fmt.Errorf("vault_addr is required")
    }
    if c.Timeout < 0 {
        return fmt.Errorf("timeout must be positive")
    }
    return nil
}
```

## Performance

### Use Pointers for Large Structs
```go
// ✅ Pass pointers to avoid copying
func ProcessConfig(cfg *Config) error { }
```

### Preallocate Slices
```go
// ✅ When size is known
results := make([]Result, 0, len(inputs))
```

### Use sync.Pool for Frequent Allocations
```go
var bufPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process() {
    buf := bufPool.Get().(*bytes.Buffer)
    defer bufPool.Put(buf)
    buf.Reset()
    // Use buffer...
}
```

## Code Review Checklist

- [ ] All errors are handled
- [ ] Context is propagated through I/O operations
- [ ] Tests cover new functionality
- [ ] Logging uses structured fields
- [ ] No race conditions (verified with `-race`)
- [ ] Documentation comments on exported symbols
- [ ] Conventional commit message
- [ ] `golangci-lint` passes

## Documentation

### Package Documentation
```go
// Package vault provides a client for HashiCorp Vault KV2 secrets engine.
//
// The client supports recursive secret listing, authentication via AppRole,
// and automatic token renewal.
package vault
```

### Function Documentation
```go
// ListSecrets recursively lists all secrets under the given path.
// It returns the full paths of all secrets found, excluding directories.
//
// The path should be in the format "secret/data/path" for KV2 engines.
// Directories are identified by trailing slashes and are traversed recursively.
//
// Returns an error if the path is invalid or if Vault returns an error.
func ListSecrets(ctx context.Context, path string) ([]string, error) { }
```

## Common Patterns

### Dependency Injection
```go
// ✅ Accept interfaces, return structs
type Client struct {
    vault VaultAPI
    aws   AWSAPI
}

func NewClient(vault VaultAPI, aws AWSAPI) *Client {
    return &Client{vault: vault, aws: aws}
}
```

### Options Pattern
```go
type ClientOption func(*Client)

func WithTimeout(d time.Duration) ClientOption {
    return func(c *Client) {
        c.timeout = d
    }
}

func NewClient(opts ...ClientOption) *Client {
    c := &Client{timeout: defaultTimeout}
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

### Resource Cleanup
```go
func (c *Client) Close() error {
    // Close resources in reverse order of creation
    if err := c.conn.Close(); err != nil {
        return err
    }
    return nil
}

// Usage with defer
client, err := NewClient()
if err != nil {
    return err
}
defer client.Close()
```

## References

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

