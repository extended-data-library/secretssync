# SecretSync - Complete Requirements

## Introduction

SecretSync is a production-ready Go application for synchronizing secrets from HashiCorp Vault to AWS Secrets Manager and other external secret stores. This document defines complete functional and non-functional requirements for the entire system.

**Target Users:**
- DevOps Engineers managing multi-account AWS environments
- Platform Engineers building secret management infrastructure
- Security Teams enforcing secret rotation policies
- Organizations migrating from Vault to AWS Secrets Manager

## Functional Requirements

### FR-1: Vault Integration

#### FR-1.1: Vault Authentication

**Requirement:** SecretSync SHALL authenticate to HashiCorp Vault using AppRole method.

**Acceptance Criteria:**
1. WHEN `VAULT_ROLE_ID` and `VAULT_SECRET_ID` environment variables are set THEN authentication SHALL succeed
2. WHEN Vault address is configured via `VAULT_ADDR` THEN client SHALL connect to that address
3. WHEN authentication fails THEN clear error message SHALL explain the cause
4. WHEN token expires THEN client SHALL automatically renew token
5. WHEN token renewal fails THEN client SHALL re-authenticate with AppRole

**Configuration:**
```yaml
vault:
  address: https://vault.example.com:8200
  # role_id and secret_id from env vars
```

#### FR-1.2: Vault KV2 Secret Listing

**Requirement:** SecretSync SHALL recursively list all secrets from Vault KV2 mount paths using BFS traversal.

**Acceptance Criteria:**
1. WHEN listing a Vault path THEN all nested secrets SHALL be discovered
2. WHEN a directory is encountered (ends with `/`) THEN it SHALL be traversed
3. WHEN a secret is found THEN its full path SHALL be returned without leading slash
4. WHEN cycles are detected THEN traversal SHALL prevent infinite loops
5. WHEN `max_secrets` limit is reached THEN traversal SHALL stop
6. WHEN path is invalid THEN error SHALL explain the validation failure
7. WHEN permissions are insufficient THEN error SHALL indicate permission issue

**Implementation:** `pkg/client/vault/vault.go` - `ListSecretsRecursive()`

#### FR-1.3: Vault Secret Reading

**Requirement:** SecretSync SHALL read secret values from Vault KV2 mounts.

**Acceptance Criteria:**
1. WHEN reading a secret THEN both metadata and data SHALL be retrieved
2. WHEN secret does not exist THEN clear error SHALL be returned
3. WHEN secret is deleted THEN appropriate error SHALL be returned
4. WHEN secret has multiple versions THEN latest version SHALL be used
5. WHEN reading fails due to network error THEN retry SHALL be attempted

#### FR-1.4: Path Security

**Requirement:** SecretSync SHALL validate and sanitize all Vault paths to prevent security issues.

**Acceptance Criteria:**
1. WHEN path contains `..` THEN it SHALL be rejected
2. WHEN path contains null bytes (`\x00`) THEN it SHALL be rejected
3. WHEN path contains `//` THEN it SHALL be normalized to single `/`
4. WHEN path is absolute (starts with `/`) THEN it SHALL be handled correctly
5. WHEN path is relative THEN it SHALL be resolved against mount path

### FR-2: AWS Integration

#### FR-2.1: AWS Authentication

**Requirement:** SecretSync SHALL authenticate to AWS using multiple methods.

**Acceptance Criteria:**
1. WHEN running in Kubernetes THEN IRSA SHALL be used for authentication
2. WHEN running in GitHub Actions THEN OIDC SHALL be used for authentication
3. WHEN `AWS_ROLE_ARN` is configured THEN role assumption SHALL be performed
4. WHEN running locally THEN AWS credentials from environment SHALL be used
5. WHEN authentication fails THEN error SHALL explain which method was attempted

#### FR-2.2: AWS Secrets Manager Operations

**Requirement:** SecretSync SHALL perform CRUD operations on AWS Secrets Manager.

**Acceptance Criteria:**
1. WHEN listing secrets THEN pagination SHALL handle > 100 secrets
2. WHEN creating a secret THEN it SHALL be created with appropriate metadata
3. WHEN updating a secret THEN only changed values SHALL be updated
4. WHEN deleting a secret THEN deletion SHALL be confirmed
5. WHEN secret already exists THEN update SHALL be performed instead of create
6. WHEN `NoEmptySecrets` is true THEN empty secrets SHALL be skipped

**Operations:**
- `ListSecrets()` - List all secrets with pagination
- `GetSecret()` - Read secret value
- `CreateSecret()` - Create new secret
- `UpdateSecret()` - Update existing secret
- `DeleteSecret()` - Delete secret

#### FR-2.3: Cross-Account Access

**Requirement:** SecretSync SHALL support syncing to AWS accounts other than the execution account.

**Acceptance Criteria:**
1. WHEN `role_arn` is configured for a target THEN that role SHALL be assumed
2. WHEN role assumption fails THEN error SHALL indicate the role ARN and reason
3. WHEN external ID is required THEN it SHALL be configurable
4. WHEN role session expires THEN new session SHALL be created automatically
5. WHEN assuming role in multiple accounts THEN sessions SHALL be managed independently

#### FR-2.4: S3 Merge Store

**Requirement:** SecretSync SHALL store merged secret configurations in S3 for inheritance.

**Acceptance Criteria:**
1. WHEN merge phase completes THEN merged secrets SHALL be written to S3
2. WHEN sync phase starts THEN secrets SHALL be read from S3
3. WHEN S3 bucket is in different account THEN role assumption SHALL work
4. WHEN listing S3 objects THEN pagination SHALL handle > 1000 objects
5. WHEN S3 object does not exist THEN clear error SHALL be returned
6. WHEN S3 access is denied THEN error SHALL include bucket and prefix

**Storage Format:**
- Path: `s3://bucket/prefix/target-name/secret-path.json`
- Content: JSON object with secret data

### FR-3: Pipeline Architecture

#### FR-3.1: Merge Phase

**Requirement:** SecretSync SHALL merge secrets from multiple Vault sources using deep merge strategy.

**Acceptance Criteria:**
1. WHEN multiple sources provide the same secret path THEN values SHALL be deep merged
2. WHEN merging lists THEN items SHALL be appended (not replaced)
3. WHEN merging maps THEN keys SHALL be recursively merged
4. WHEN merging scalars THEN later source SHALL override earlier source
5. WHEN type conflict occurs (list vs map) THEN later source SHALL win
6. WHEN merge completes THEN result SHALL be available for sync phase

**Merge Strategy:**
- Lists: Append
- Maps: Recursive merge
- Sets: Union
- Scalars: Override
- Type conflicts: Override

#### FR-3.2: Sync Phase

**Requirement:** SecretSync SHALL sync merged secrets to configured targets.

**Acceptance Criteria:**
1. WHEN target has no dependencies THEN it SHALL be synced immediately
2. WHEN target has dependencies THEN dependencies SHALL be synced first
3. WHEN circular dependency is detected THEN clear error SHALL be raised
4. WHEN target imports from another target THEN import SHALL be resolved from S3
5. WHEN sync to one target fails THEN other targets SHALL still be attempted
6. WHEN `--dry-run` is specified THEN no actual changes SHALL be made

#### FR-3.3: Target Inheritance

**Requirement:** SecretSync SHALL support target-to-target inheritance via merge store.

**Acceptance Criteria:**
1. WHEN target imports from another target THEN merged output SHALL be read from S3
2. WHEN resolving imports THEN topological sort SHALL determine order
3. WHEN multi-level inheritance exists (A→B→C) THEN all levels SHALL resolve correctly
4. WHEN imported target does not exist in S3 THEN error SHALL indicate the target name
5. WHEN target overrides imported values THEN overrides SHALL take precedence

**Configuration Example:**
```yaml
targets:
  - name: base
    type: aws_secretsmanager
    # Base configuration
    
  - name: production
    imports:
      - base  # Inherits from base target
    overrides:
      environment: production
```

### FR-4: Configuration Management

#### FR-4.1: YAML Configuration

**Requirement:** SecretSync SHALL load configuration from YAML files.

**Acceptance Criteria:**
1. WHEN `--config` flag is provided THEN that file SHALL be loaded
2. WHEN YAML syntax is invalid THEN clear parse error SHALL be shown
3. WHEN required fields are missing THEN validation SHALL fail with specific field names
4. WHEN unknown fields are present THEN warning SHALL be logged
5. WHEN file does not exist THEN error SHALL indicate the path

**Configuration Structure:**
```yaml
vault_sources:
  - mount: secret/
    max_secrets: 10000

merge_store:
  enabled: true
  type: s3
  bucket: my-merge-store

targets:
  - name: production
    type: aws_secretsmanager
    region: us-east-1
```

#### FR-4.2: Environment Variable Substitution

**Requirement:** SecretSync SHALL support environment variable substitution in configuration.

**Acceptance Criteria:**
1. WHEN configuration contains `${VAR}` THEN it SHALL be replaced with env var value
2. WHEN env var is not set THEN error SHALL indicate the variable name
3. WHEN default is specified `${VAR:-default}` THEN default SHALL be used if var not set
4. WHEN substitution is escaped `$${VAR}` THEN literal string SHALL be preserved

#### FR-4.3: Configuration Validation

**Requirement:** SecretSync SHALL validate configuration before execution.

**Acceptance Criteria:**
1. WHEN validating THEN all required fields SHALL be checked
2. WHEN role ARNs are invalid format THEN error SHALL explain ARN format
3. WHEN S3 bucket name is invalid THEN error SHALL explain bucket naming rules
4. WHEN region is invalid THEN error SHALL list valid regions
5. WHEN validation passes THEN confirmation message SHALL be logged

### FR-5: Discovery

#### FR-5.1: AWS Organizations Discovery

**Requirement:** SecretSync SHALL discover AWS accounts from AWS Organizations.

**Acceptance Criteria:**
1. WHEN discovery is enabled THEN all accounts in organization SHALL be found
2. WHEN tag filters are specified THEN only matching accounts SHALL be discovered
3. WHEN OU filter is specified THEN only accounts in that OU SHALL be discovered
4. WHEN account is suspended THEN it SHALL be excluded
5. WHEN discovery completes THEN account list SHALL be available for target generation

**Configuration:**
```yaml
discovery:
  enabled: true
  type: aws_organizations
  filters:
    - tag: Environment
      values: [production, staging]
    - ou: ou-prod-xxxx
```

#### FR-5.2: Dynamic Target Generation (v1.2.0)

**Requirement:** SecretSync SHALL generate targets dynamically from discovered accounts.

**Acceptance Criteria:**
1. WHEN target template is defined THEN it SHALL be applied to each discovered account
2. WHEN template uses account ID THEN it SHALL be substituted
3. WHEN template uses account tags THEN they SHALL be substituted
4. WHEN generated targets have dependencies THEN order SHALL be determined automatically
5. WHEN account list changes THEN targets SHALL be regenerated

### FR-6: Diff and Dry-Run

#### FR-6.1: Diff Computation

**Requirement:** SecretSync SHALL compute differences between current and desired state.

**Acceptance Criteria:**
1. WHEN `--diff` flag is provided THEN differences SHALL be computed
2. WHEN secret is new THEN it SHALL be marked as "added"
3. WHEN secret value changes THEN it SHALL be marked as "modified"
4. WHEN secret is removed THEN it SHALL be marked as "deleted"
5. WHEN secret metadata changes THEN it SHALL be marked as "modified"
6. WHEN no changes exist THEN "no differences" message SHALL be shown

**Output Format:**
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

#### FR-6.2: Dry-Run Mode

**Requirement:** SecretSync SHALL support dry-run mode for safe validation.

**Acceptance Criteria:**
1. WHEN `--dry-run` is specified THEN no actual changes SHALL be made
2. WHEN in dry-run mode THEN diff SHALL still be computed
3. WHEN in dry-run mode THEN all validation SHALL still occur
4. WHEN in dry-run mode THEN output SHALL clearly indicate "DRY RUN" mode
5. WHEN errors occur in dry-run THEN they SHALL still be reported

### FR-7: Observability (v1.1.0)

#### FR-7.1: Prometheus Metrics

**Requirement:** SecretSync SHALL expose Prometheus-compatible metrics.

**Acceptance Criteria:**
1. WHEN `--metrics-port` is specified THEN metrics endpoint SHALL be available
2. WHEN Vault API is called THEN request duration SHALL be recorded
3. WHEN AWS API is called THEN request duration SHALL be recorded
4. WHEN pipeline executes THEN execution duration SHALL be recorded
5. WHEN errors occur THEN error counters SHALL be incremented
6. WHEN metrics are scraped THEN standard Go runtime metrics SHALL be included

**Metrics:**
- `secretsync_vault_request_duration_seconds{operation}`
- `secretsync_aws_request_duration_seconds{service, operation}`
- `secretsync_pipeline_duration_seconds{phase}`
- `secretsync_secrets_synced_total{target}`
- `secretsync_errors_total{component, error_type}`

#### FR-7.2: Structured Logging

**Requirement:** SecretSync SHALL log using structured format with contextual information.

**Acceptance Criteria:**
1. WHEN operations occur THEN logs SHALL include timestamp, level, message
2. WHEN errors occur THEN logs SHALL include error context
3. WHEN request ID exists THEN it SHALL be included in log fields
4. WHEN `--log-format json` is specified THEN logs SHALL be JSON formatted
5. WHEN sensitive data is logged THEN it SHALL be redacted

**Log Fields:**
- `timestamp` - ISO 8601 format
- `level` - ERROR, WARN, INFO, DEBUG
- `message` - Human-readable message
- `request_id` - Unique request identifier
- `operation` - Operation name
- `duration_ms` - Operation duration
- `error` - Error message (if applicable)

#### FR-7.3: Enhanced Error Context (v1.1.0)

**Requirement:** SecretSync SHALL include rich context in all error messages.

**Acceptance Criteria:**
1. WHEN error occurs THEN request ID SHALL be included
2. WHEN API call fails THEN operation name and path SHALL be included
3. WHEN operation is slow THEN duration SHALL be included
4. WHEN retries occur THEN retry count SHALL be included
5. WHEN error wraps another error THEN full chain SHALL be preserved

**Error Format:**
```
[req=abc123] failed to list secrets at path "secret/data/app" after 1250ms (retries: 2): permission denied
```

### FR-8: Reliability (v1.1.0)

#### FR-8.1: Circuit Breaker

**Requirement:** SecretSync SHALL implement circuit breaker pattern for external API calls.

**Acceptance Criteria:**
1. WHEN Vault fails 5 times in 10 seconds THEN circuit SHALL open
2. WHEN circuit is open THEN requests SHALL fail immediately
3. WHEN circuit timeout expires THEN circuit SHALL enter half-open state
4. WHEN half-open request succeeds THEN circuit SHALL close
5. WHEN half-open request fails THEN circuit SHALL re-open
6. WHEN circuit state changes THEN event SHALL be logged

**Configuration:**
```yaml
circuit_breaker:
  enabled: true
  failure_threshold: 5
  timeout: 30s
  max_requests: 1
```

#### FR-8.2: Retry with Backoff

**Requirement:** SecretSync SHALL retry transient failures with exponential backoff.

**Acceptance Criteria:**
1. WHEN network error occurs THEN retry SHALL be attempted
2. WHEN rate limit is encountered THEN backoff SHALL honor retry-after header
3. WHEN retry succeeds THEN operation SHALL complete normally
4. WHEN max retries is reached THEN error SHALL be returned
5. WHEN non-transient error occurs THEN no retry SHALL be attempted

**Backoff Strategy:**
- Initial delay: 100ms
- Max delay: 30s
- Multiplier: 2
- Max attempts: 3

#### FR-8.3: Graceful Degradation

**Requirement:** SecretSync SHALL continue operation when non-critical failures occur.

**Acceptance Criteria:**
1. WHEN one target fails THEN other targets SHALL still sync
2. WHEN one secret fails THEN other secrets SHALL still sync
3. WHEN discovery fails THEN manually configured targets SHALL still work
4. WHEN metrics endpoint fails THEN pipeline SHALL still execute
5. WHEN all failures occur THEN summary SHALL list all errors

## Non-Functional Requirements

### NFR-1: Performance

**Requirements:**
1. Pipeline SHALL complete within 5 minutes for 1,000 secrets
2. Vault listing SHALL process 100 directories/second minimum
3. AWS Secrets Manager sync SHALL process 50 secrets/second minimum
4. Memory usage SHALL not exceed 500MB for typical workloads
5. API response time p95 SHALL be < 500ms

**Targets:**
- Secrets synced: 10,000+
- Vault mounts: 100+
- AWS accounts: 100+
- Concurrent operations: 10 workers

### NFR-2: Reliability

**Requirements:**
1. Pipeline SHALL succeed 99.9% of the time when services are healthy
2. Transient failures SHALL be retried automatically
3. Circuit breaker SHALL prevent cascade failures
4. State SHALL be consistent (all or nothing for targets)
5. Concurrent executions SHALL not interfere with each other

### NFR-3: Security

**Requirements:**
1. Credentials SHALL never be logged
2. All external connections SHALL use TLS
3. Secrets SHALL never be written to disk unencrypted
4. Path traversal attacks SHALL be prevented
5. Input validation SHALL prevent injection attacks
6. Least privilege principle SHALL be followed for IAM policies

**Security Standards:**
- Follow OWASP Secure Coding Practices
- Pass security scanning (gosec, Trivy)
- No HIGH or CRITICAL CVEs in dependencies
- Regular dependency updates via Dependabot

### NFR-4: Maintainability

**Requirements:**
1. Code coverage SHALL be ≥ 80%
2. All public APIs SHALL have documentation comments
3. Complex logic SHALL have inline comments explaining why
4. Git commits SHALL follow Conventional Commits format
5. Breaking changes SHALL be documented in CHANGELOG.md

**Code Quality:**
- Pass `golangci-lint` with no errors
- Pass `go vet` with no warnings
- Pass race detector (`go test -race`)
- Follow Go standard project layout

### NFR-5: Usability

**Requirements:**
1. Error messages SHALL be clear and actionable
2. `--help` flag SHALL provide complete usage information
3. Common operations SHALL be achievable with single command
4. Configuration SHALL be validated before execution
5. Progress indicators SHALL show long-running operations

**User Experience:**
- Clear success/failure indication
- Dry-run mode for safe testing
- Diff output for change preview
- Examples in documentation

### NFR-6: Portability

**Requirements:**
1. Application SHALL run on Linux, macOS, and Windows
2. Application SHALL run in Kubernetes
3. Application SHALL run in GitHub Actions
4. Application SHALL run as standalone CLI
5. Docker image SHALL support multi-arch (amd64, arm64)

**Deployment Targets:**
- Local development machines
- Kubernetes clusters
- GitHub Actions runners
- AWS Lambda (future)
- Azure DevOps pipelines (future)

### NFR-7: Observability

**Requirements:**
1. Metrics SHALL be Prometheus-compatible
2. Logs SHALL be structured (JSON or text)
3. Request tracing SHALL use request IDs
4. Error context SHALL include operation details
5. Circuit breaker state SHALL be observable

**Monitoring Integration:**
- Prometheus/Grafana
- CloudWatch (via EMF)
- Datadog (via statsd)
- Generic StatsD endpoint

### NFR-8: Scalability

**Requirements:**
1. SHALL handle 10,000+ secrets per execution
2. SHALL support 100+ AWS accounts
3. SHALL support 100+ Vault mounts
4. Memory usage SHALL scale linearly with secret count
5. Execution time SHALL scale sub-linearly with secret count

**Scalability Techniques:**
- Streaming for large secret lists
- Bounded concurrency
- Efficient data structures
- Connection pooling

## Acceptance Testing

### End-to-End Scenarios

#### Scenario 1: Basic Vault to AWS Sync

**Given:** Vault contains 100 secrets in `secret/app/`  
**When:** Pipeline executes with target for AWS Secrets Manager  
**Then:** All 100 secrets are synced to AWS  
**And:** Diff shows 100 additions  
**And:** No errors occur  

#### Scenario 2: Inheritance

**Given:** Base target synced with 50 secrets  
**And:** Production target imports base and adds 25 secrets  
**When:** Pipeline executes  
**Then:** Production target has 75 secrets (50 + 25)  
**And:** Base secrets are not duplicated  

#### Scenario 3: Discovery

**Given:** AWS Organization has 10 accounts  
**And:** 5 accounts tagged "Environment: production"  
**When:** Discovery runs with tag filter  
**Then:** 5 targets are generated  
**And:** Each target has correct account ID  

#### Scenario 4: Circuit Breaker

**Given:** Vault is unavailable  
**When:** 5 requests fail  
**Then:** Circuit opens  
**And:** Subsequent requests fail fast  
**And:** After 30 seconds circuit allows test request  

#### Scenario 5: Dry-Run

**Given:** Configuration with 100 secrets to sync  
**When:** Pipeline runs with `--dry-run`  
**Then:** Diff is computed and displayed  
**And:** No actual changes are made to AWS  
**And:** Exit code indicates changes would be made  

## Success Criteria

SecretSync v1.2.0 SHALL be considered complete when:

1. ✅ All functional requirements are implemented
2. ✅ All non-functional requirements are met
3. ✅ Test coverage ≥ 80%
4. ✅ All integration tests pass
5. ✅ Security scan shows no HIGH/CRITICAL issues
6. ✅ Performance targets are met
7. ✅ Documentation is complete
8. ✅ Example configurations work
9. ✅ GitHub Action is published
10. ✅ Docker image is published

---

**Document Version:** 1.0  
**Last Updated:** 2024-12-09  
**Status:** Complete system requirements (v1.0-v1.2.0)

