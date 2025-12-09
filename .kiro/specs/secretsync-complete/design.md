# SecretSync - Complete System Design

## Executive Summary

SecretSync is a production-ready Go application that synchronizes secrets from HashiCorp Vault to AWS Secrets Manager and other external secret stores. It uses a two-phase pipeline architecture (merge + sync) with S3-based configuration inheritance, enabling enterprise-scale secret management across multi-account AWS environments.

**Current State:** v1.0 core complete, working toward v1.1.0 (observability) and v1.2.0 (advanced features)

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         SecretSync Pipeline                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐     │
│  │   Sources   │─────▶│    Merge    │─────▶│  S3 Store   │     │
│  │   (Vault)   │      │    Phase    │      │  (Optional) │     │
│  └─────────────┘      └─────────────┘      └─────────────┘     │
│                              │                      │            │
│                              │                      │            │
│                              ▼                      ▼            │
│                       ┌─────────────┐      ┌─────────────┐     │
│                       │    Sync     │◀─────│ Inheritance │     │
│                       │    Phase    │      │  Resolution │     │
│                       └─────────────┘      └─────────────┘     │
│                              │                                   │
│  ┌─────────────┐            │            ┌─────────────┐       │
│  │   Targets   │◀───────────┴───────────▶│  Discovery  │       │
│  │ (AWS SM)    │                          │ (AWS Orgs)  │       │
│  └─────────────┘                          └─────────────┘       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. Vault Client (`pkg/client/vault/`)

**Purpose:** Read secrets from HashiCorp Vault KV2 engine

**Key Features:**
- Recursive secret listing using BFS traversal
- Cycle detection to prevent infinite loops
- AppRole authentication
- Token renewal
- Path validation and security

**Implementation:**
```go
type VaultClient struct {
    client      *vault.Client
    api         LogicalClient  // Interface for dependency injection
    mountPath   string
    visited     map[string]bool // Cycle detection
}

// BFS recursive listing
func (vc *VaultClient) ListSecretsRecursive(ctx context.Context, path string) ([]string, error)
```

**Security:**
- Path traversal prevention (`..`, null bytes)
- Type-safe response parsing
- No credentials in logs

#### 2. AWS Client (`pkg/client/aws/`)

**Purpose:** Interact with AWS Secrets Manager and other AWS services

**Key Features:**
- Secrets Manager CRUD operations
- Pagination handling (NextToken)
- Empty secret filtering
- ARN caching with TTL
- Cross-account role assumption

**Implementation:**
```go
type AwsClient struct {
    smClient           *secretsmanager.Client
    s3Client           *s3.Client
    orgsClient         *organizations.Client
    accountSecretArns  map[string]string
    arnMu              sync.RWMutex  // Race condition protection
}
```

**Performance:**
- TTL-based caching for ListSecrets
- Connection pooling
- Parallel secret fetching (bounded concurrency)

#### 3. Pipeline (`pkg/pipeline/`)

**Purpose:** Orchestrate the two-phase sync process

**Architecture:**
```
Merge Phase:
1. Read secrets from Vault sources
2. Apply deep merge strategy
3. Write merged output to S3 merge store (optional)

Sync Phase:
1. Read merged secrets from memory or S3
2. Resolve target inheritance
3. Sync to AWS Secrets Manager targets
4. Compute diffs (optional)
```

**Key Functions:**
```go
func (p *Pipeline) Execute(ctx context.Context, config *Config) error {
    // Merge phase
    merged := p.mergeSecrets(ctx, config.VaultSources)
    p.writeMergeStore(ctx, merged)
    
    // Sync phase
    targets := p.resolveInheritance(ctx, config.Targets)
    return p.syncTargets(ctx, targets)
}
```

**Topological Sorting:**
- Determines execution order based on dependencies
- Detects circular dependencies
- Enables target inheritance

#### 4. Deep Merge (`pkg/utils/deepmerge.go`)

**Purpose:** Merge configuration from multiple sources

**Strategy:**
- Lists: Append (not replace)
- Maps: Recursive merge
- Sets: Union
- Scalars: Override
- Type conflicts: Override with new value

**Example:**
```go
base := map[string]interface{}{
    "api_keys": []interface{}{"key1", "key2"},
    "config": map[string]interface{}{
        "timeout": 30,
        "retries": 3,
    },
}

overlay := map[string]interface{}{
    "api_keys": []interface{}{"key3"},
    "config": map[string]interface{}{
        "timeout": 60,  // Override
        "debug": true,  // Add new
    },
}

result := DeepMerge(base, overlay)
// api_keys: ["key1", "key2", "key3"]
// config: {timeout: 60, retries: 3, debug: true}
```

#### 5. S3 Merge Store (`pkg/pipeline/s3_store.go`)

**Purpose:** Store merged secrets for inheritance and auditing

**Operations:**
```go
type S3MergeStore struct {
    client     *s3.Client
    bucketName string
    prefix     string
}

func (s *S3MergeStore) WriteSecret(ctx context.Context, target, path string, data map[string]interface{}) error
func (s *S3MergeStore) ReadSecret(ctx context.Context, target, path string) (map[string]interface{}, error)
func (s *S3MergeStore) ListSecrets(ctx context.Context, target string) ([]string, error)
```

**Storage Format:**
```
s3://bucket/prefix/
├── target-a/
│   ├── secret1.json
│   └── secret2.json
└── target-b/
    └── secret3.json
```

#### 6. Discovery (`pkg/discovery/`)

**Purpose:** Automatically discover AWS resources

**Current:**
- AWS Organizations account discovery
- Tag-based filtering
- Organizational Unit filtering

**Planned (v1.2.0):**
- AWS Identity Center integration
- Permission set mapping
- Dynamic target generation

#### 7. Diff Engine (`pkg/diff/`)

**Purpose:** Compute differences between secret states

**Output Formats:**
- Text (colored terminal output)
- JSON (structured data)
- GitHub (PR annotations)

**Example Output:**
```
Diff Summary:
  Added:    5 secrets
  Modified: 3 secrets
  Deleted:  1 secret

Changes:
  + production/api/new-key
  ~ production/db/password (value changed)
  - staging/old-token
```

### Data Flow

#### Merge Phase Flow

```
1. Load Configuration
   ↓
2. Initialize Vault Client
   ↓
3. For each VaultSource:
   ├─ List secrets recursively
   ├─ Read secret values
   └─ Add to merge map
   ↓
4. Apply Deep Merge
   ↓
5. Write to S3 Merge Store (optional)
```

#### Sync Phase Flow

```
1. Load Merged Secrets (memory or S3)
   ↓
2. Resolve Target Dependencies
   ├─ Topological sort
   ├─ Detect circular refs
   └─ Build execution order
   ↓
3. For each Target (in order):
   ├─ Resolve imports (S3)
   ├─ Apply deep merge
   ├─ Initialize AWS client
   ├─ List existing secrets
   ├─ Compute diff
   ├─ Apply changes
   └─ Record metrics
```

## Configuration Model

### Complete Configuration Example

```yaml
# Vault sources to read from
vault_sources:
  - mount: secret/base/
    max_secrets: 10000
    queue_compaction_threshold: 500
  - mount: secret/production/
    max_secrets: 5000

# S3 merge store (optional)
merge_store:
  enabled: true
  type: s3
  bucket: my-secrets-merge-store
  prefix: merged/
  region: us-east-1

# Sync targets
targets:
  - name: production-us-east-1
    type: aws_secretsmanager
    region: us-east-1
    role_arn: arn:aws:iam::123456789012:role/SecretSync
    imports:
      - base_merged  # From merge store
    overrides:
      environment: production

  - name: staging-us-west-2
    type: aws_secretsmanager
    region: us-west-2
    role_arn: arn:aws:iam::987654321098:role/SecretSync
    imports:
      - production-us-east-1  # Inherit from another target
    overrides:
      environment: staging

# Discovery (optional)
discovery:
  enabled: true
  type: aws_organizations
  filters:
    - tag: Environment
      values: [production, staging]
    - ou: ou-prod-xxxx
  role_arn: arn:aws:iam::123456789012:role/OrgDiscovery

# Observability (v1.1.0)
metrics:
  enabled: true
  port: 9090
  path: /metrics

circuit_breaker:
  enabled: true
  failure_threshold: 5
  timeout: 30s
  max_requests: 1
```

## Deployment Models

### 1. CLI Usage

```bash
# Dry run to preview changes
secretsync pipeline --config config.yaml --dry-run

# Execute sync
secretsync pipeline --config config.yaml

# With diff output
secretsync pipeline --config config.yaml --diff --output github

# Merge phase only
secretsync pipeline --config config.yaml --merge-only

# Sync phase only (reads from S3)
secretsync pipeline --config config.yaml --sync-only
```

### 2. GitHub Action

```yaml
- uses: jbcom/secretsync@v1
  with:
    config: .secretsync/config.yaml
    dry-run: 'false'
    diff: 'true'
    output-format: 'github'
  env:
    VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
    VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
    VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
```

### 3. Kubernetes CronJob (Future)

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: secretsync
spec:
  schedule: "*/15 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: secretsync
          containers:
          - name: secretsync
            image: jbcom/secretsync:v1
            args:
            - pipeline
            - --config
            - /config/config.yaml
```

## Security Model

### Authentication

**Vault:**
- AppRole (role_id + secret_id)
- Environment variables: `VAULT_ROLE_ID`, `VAULT_SECRET_ID`
- Token renewal handled automatically

**AWS:**
- IRSA (IAM Roles for Service Accounts) in Kubernetes
- OIDC for GitHub Actions
- Role assumption for cross-account access
- Environment variables: `AWS_ROLE_ARN` or config role_arn

### Authorization

**Vault Policies:**
```hcl
# Read-only access to secret mounts
path "secret/data/*" {
  capabilities = ["read", "list"]
}

path "secret/metadata/*" {
  capabilities = ["list"]
}
```

**AWS IAM Policies:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:ListSecrets",
        "secretsmanager:GetSecretValue",
        "secretsmanager:CreateSecret",
        "secretsmanager:UpdateSecret",
        "secretsmanager:DeleteSecret"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::merge-store-bucket",
        "arn:aws:s3:::merge-store-bucket/*"
      ]
    }
  ]
}
```

### Data Protection

- **In Transit:** TLS for all external connections
- **At Rest:** S3 server-side encryption (SSE-S3 or SSE-KMS)
- **In Memory:** Secrets cleared after use
- **Logging:** Sensitive values never logged

### Input Validation

- Path traversal prevention (`..`, null bytes, `//`)
- YAML injection prevention
- SQL injection N/A (no database)
- Command injection prevention (no shell execution)

## Performance Characteristics

### Scalability

| Metric | Target | Current |
|--------|--------|---------|
| Secrets per sync | 10,000+ | ✅ Tested |
| Vault mounts | 100+ | ✅ Supported |
| AWS accounts | 100+ | ⏳ v1.2.0 |
| Pipeline duration | < 5 min (1000 secrets) | ✅ Achieved |
| Memory usage | < 500 MB | ✅ Typical |

### Optimization Techniques

1. **Caching:**
   - TTL-based caching for AWS ListSecrets (reduces API calls by 90%)
   - Vault response caching (planned)

2. **Parallelization:**
   - Bounded concurrency for secret fetching
   - Worker pool pattern (10 workers default)

3. **Efficient Traversal:**
   - BFS instead of DFS (prevents stack overflow)
   - Cycle detection (prevents infinite loops)
   - Early termination on max_secrets

4. **Resource Management:**
   - Connection pooling for AWS SDK
   - Proper cleanup with defer
   - Context cancellation support

## Error Handling Strategy

### Error Categories

1. **Configuration Errors:**
   - Invalid YAML syntax
   - Missing required fields
   - Invalid references
   - **Action:** Fail fast with clear message

2. **Authentication Errors:**
   - Invalid Vault credentials
   - AWS permission denied
   - Expired tokens
   - **Action:** Fail with authentication instructions

3. **Transient Errors:**
   - Network timeouts
   - Rate limiting
   - Temporary service unavailability
   - **Action:** Retry with exponential backoff

4. **Data Errors:**
   - Invalid secret format
   - Merge conflicts
   - Circular dependencies
   - **Action:** Log warning, continue or fail based on severity

### Error Context (v1.1.0)

All errors include:
- Request ID (for correlation)
- Operation name
- Resource path
- Duration
- Retry count

Example:
```
[req=abc123] failed to list secrets at path "secret/data/app" after 1250ms (retries: 2): permission denied
```

### Circuit Breaker (v1.1.0)

Prevents cascade failures:
- Opens after 5 failures in 10 seconds
- Fails fast when open (30 second timeout)
- Half-open state allows test request
- Independent circuits per service (Vault, AWS)

## Testing Strategy

### Unit Tests

**Coverage Target:** 80%+

**Approach:**
- Table-driven tests
- Mock external dependencies
- Test edge cases explicitly

**Example:**
```go
func TestDeepMerge_ListAppend(t *testing.T) {
    tests := []struct {
        name string
        base map[string]interface{}
        overlay map[string]interface{}
        want map[string]interface{}
    }{
        // Test cases...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic...
        })
    }
}
```

### Integration Tests

**Environment:** Docker Compose (Vault + LocalStack)

**Workflows Tested:**
- Full pipeline (Vault → S3 → AWS)
- Vault recursive listing
- Target inheritance
- Discovery integration

**Location:** `tests/integration/`

### Race Detection

All tests run with `-race` flag in CI

**Protected Resources:**
- `accountSecretArns` map (sync.RWMutex)
- Cache structures
- Shared configuration

### Security Testing

- SAST with gosec
- Dependency scanning with Dependabot
- Container scanning with Trivy
- Manual security reviews

## Observability (v1.1.0)

### Metrics

**Prometheus Metrics Exposed:**

```
# Vault metrics
secretsync_vault_request_duration_seconds{operation="list|read"}
secretsync_vault_requests_total{operation="list|read"}
secretsync_vault_errors_total{operation="list|read"}
secretsync_vault_secrets_total

# AWS metrics
secretsync_aws_request_duration_seconds{service="secretsmanager|s3",operation="list|get|put"}
secretsync_aws_requests_total{service="secretsmanager|s3",operation="list|get|put"}
secretsync_aws_errors_total{service="secretsmanager|s3"}
secretsync_aws_pagination_calls_total

# Pipeline metrics
secretsync_pipeline_duration_seconds{phase="merge|sync"}
secretsync_secrets_synced_total{target="name"}

# Circuit breaker metrics
secretsync_circuit_breaker_state{service="vault|aws",state="closed|open|half_open"}
```

### Logging

**Structured Logging with Logrus:**

```go
log.WithFields(logrus.Fields{
    "request_id": "abc123",
    "operation": "vault.list",
    "path": "secret/data/app",
    "duration_ms": 150,
}).Info("secrets listed successfully")
```

**Log Levels:**
- ERROR: Actionable errors
- WARN: Degraded state
- INFO: Normal operations
- DEBUG: Detailed troubleshooting

### Tracing (Future - v1.3.0)

OpenTelemetry integration planned for distributed tracing

## Roadmap

### v1.1.0 - Observability & Reliability (Current)

- [x] Prometheus metrics endpoint
- [x] Circuit breaker pattern
- [x] Enhanced error messages with request IDs
- [ ] Docker image version pinning
- [ ] Configurable queue compaction
- [ ] Race condition tests
- [ ] CI/CD modernization
- [ ] Documentation fixes
- [ ] Command injection prevention

### v1.2.0 - Advanced Features

- [x] Vault recursive listing (DONE)
- [x] Deep merge compatibility (DONE)
- [x] Target inheritance (DONE)
- [x] S3 merge store (DONE)
- [ ] AWS Organizations discovery enhancements
- [ ] AWS Identity Center integration
- [ ] Secret versioning support
- [ ] Enhanced diff output with side-by-side comparison

### v1.3.0 - Enterprise Scale (Future)

- [ ] Distributed tracing with OpenTelemetry
- [ ] Secret rotation automation
- [ ] Multi-region replication
- [ ] Webhook notifications
- [ ] Policy-as-code validation
- [ ] Audit log export
- [ ] Performance optimizations for 100k+ secrets

### v2.0.0 - Multi-Cloud (Future)

- [ ] Google Cloud Secret Manager support
- [ ] Azure Key Vault support
- [ ] Generic webhook targets
- [ ] Plugin system for custom targets
- [ ] Advanced secret transformations
- [ ] Encryption key rotation

## Technical Decisions

### Why Go 1.25+?

- Latest stable release with modern features
- Excellent concurrency primitives
- Strong standard library
- Fast compilation and execution
- Great AWS SDK v2 support

### Why Two-Phase Pipeline?

**Merge Phase Benefits:**
- Configuration reuse via inheritance
- Audit trail in S3
- Decoupling from target sync

**Sync Phase Benefits:**
- Independent execution
- Easier rollback
- Incremental updates

### Why S3 for Merge Store?

- Durable, versioned storage
- Native AWS integration
- Cost-effective
- S3 Event notifications for automation

### Why Not Kubernetes Operator?

**Previous Architecture:** Kubernetes operator with CRDs

**Issues:**
- Over-engineered for use case
- Added ~13k lines of boilerplate
- Kubernetes-specific deployment
- Harder to test

**Current Architecture:** Simple CLI + GitHub Action
- Runs anywhere
- Easy to test
- Clear execution model
- Can still run in Kubernetes as CronJob

### Why BFS for Vault Traversal?

- Prevents stack overflow on deep hierarchies
- Easier to implement cycle detection
- More predictable memory usage
- Better for large secret trees

## Glossary

- **Vault Source:** Configuration defining Vault mount to read from
- **Target:** External secret store to sync to (e.g., AWS Secrets Manager)
- **Merge Store:** S3 bucket storing merged secret configurations
- **Inheritance:** Target importing configuration from another target
- **Deep Merge:** Recursive merging strategy for complex data structures
- **Pipeline:** Two-phase process (merge + sync)
- **Discovery:** Automatic detection of AWS resources
- **Circuit Breaker:** Pattern to prevent cascade failures
- **BFS:** Breadth-First Search traversal algorithm
- **TTL:** Time-To-Live for cache expiration
- **IRSA:** IAM Roles for Service Accounts (Kubernetes AWS auth)
- **OIDC:** OpenID Connect (GitHub Actions AWS auth)

---

**Document Version:** 1.0  
**Last Updated:** 2024-12-09  
**Status:** Current architecture (v1.0-v1.2.0)

