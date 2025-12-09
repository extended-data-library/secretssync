# SecretSync v1.2.0 Requirements

## Overview

Version 1.2.0 delivers advanced features for complex enterprise use cases: enhanced secret management, advanced discovery patterns, and sophisticated configuration inheritance.

## Milestone: v1.2.0 - Advanced Features

**Goal:** Enable enterprise-scale secret synchronization with advanced patterns

**Target Date:** Q2 2025

## Requirements

### Requirement 1: Vault Recursive Secret Listing (#23)

**Status:** ✅ COMPLETED (Merged in PR #29)

**User Story:** As a user, I want to sync all secrets in a Vault hierarchy without manual enumeration.

**Acceptance Criteria:**

1. ✅ WHEN listing secrets from a Vault path THEN all nested secrets SHALL be discovered recursively
2. ✅ WHEN a directory is encountered (trailing `/`) THEN it SHALL be traversed using BFS algorithm
3. ✅ WHEN a secret is found THEN its full path SHALL be returned without leading slash
4. ✅ WHEN cycles are detected THEN traversal SHALL prevent infinite loops
5. ✅ WHEN errors occur THEN they SHALL be logged and traversal SHALL continue
6. ✅ WHEN maximum depth is exceeded THEN traversal SHALL stop with clear error

**Implementation:** `pkg/client/vault/vault.go` lines 479-545

---

### Requirement 2: Deep Merge Compatibility (#21)

**Status:** ✅ COMPLETED (Merged in PR #29)

**User Story:** As a user, I want configuration merging to work predictably with complex nested structures.

**Acceptance Criteria:**

1. ✅ WHEN merging lists THEN they SHALL be appended (not replaced)
2. ✅ WHEN merging maps THEN keys SHALL be recursively merged
3. ✅ WHEN merging sets THEN union operation SHALL be performed
4. ✅ WHEN merging scalars THEN overlay value SHALL override base value
5. ✅ WHEN type conflicts occur THEN overlay value SHALL replace base value
6. ✅ WHEN merging 3+ levels deep THEN recursion SHALL work correctly

**Implementation:** `pkg/utils/deepmerge.go`

**Test Coverage:** 13 test functions covering all merge strategies

---

### Requirement 3: Target Inheritance Model (#22)

**Status:** ✅ COMPLETED (Merged in PR #29)

**User Story:** As a user, I want to inherit configuration from other targets to reduce duplication.

**Acceptance Criteria:**

1. ✅ WHEN a target imports another target THEN it SHALL read from merge store
2. ✅ WHEN resolving inheritance THEN topological sort SHALL determine execution order
3. ✅ WHEN multi-level inheritance exists THEN all levels SHALL be resolved correctly
4. ✅ WHEN circular dependencies exist THEN clear error SHALL be raised
5. ✅ WHEN merge store is S3 THEN inheritance SHALL read from correct S3 paths

**Implementation:** `pkg/pipeline/config.go` lines 480-510

**Functions:**
- `IsInheritedTarget()` - Detects target-to-target imports
- `GetSourcePath()` - Resolves merge store paths

---

### Requirement 4: S3 Merge Store (#4)

**Status:** ✅ COMPLETED (Merged in PR #29)

**User Story:** As a user, I want to store merged secrets in S3 for inheritance and auditing.

**Acceptance Criteria:**

1. ✅ WHEN merge phase completes THEN secrets SHALL be written to S3 bucket
2. ✅ WHEN sync phase starts THEN secrets SHALL be read from S3 bucket
3. ✅ WHEN S3 objects exceed 1000 THEN pagination SHALL handle all objects
4. ✅ WHEN S3 bucket is in different account THEN role assumption SHALL work
5. ✅ WHEN reading non-existent object THEN clear error SHALL be returned
6. ✅ WHEN S3 access is denied THEN error SHALL include bucket and key

**Implementation:** `pkg/pipeline/s3_store.go` lines 107-180

**Functions:**
- `ReadSecret()` - Read and parse JSON from S3
- `ListSecrets()` - List with pagination
- `WriteSecret()` - Write JSON to S3

---

### Requirement 5: AWS Secrets Manager Pagination (#24)

**Status:** ✅ COMPLETED (Already implemented)

**User Story:** As a user with > 100 secrets, I want pagination to work automatically.

**Acceptance Criteria:**

1. ✅ WHEN listing secrets THEN NextToken SHALL be used for pagination
2. ✅ WHEN empty secrets exist AND NoEmptySecrets=true THEN they SHALL be filtered
3. ✅ WHEN cache TTL expires THEN fresh list SHALL be fetched
4. ✅ WHEN pagination fails THEN error SHALL include page information

**Implementation:** `pkg/client/aws/aws.go`

**Features:**
- NextToken handling
- NoEmptySecrets filtering
- TTL-based caching (added in PR #29)

---

### Requirement 6: Path Handling and Security (#25)

**Status:** ✅ COMPLETED (Enhanced in PR #29)

**User Story:** As a security-conscious user, I want path traversal attacks prevented.

**Acceptance Criteria:**

1. ✅ WHEN paths contain `..` THEN they SHALL be rejected
2. ✅ WHEN paths contain null bytes THEN they SHALL be rejected
3. ✅ WHEN paths contain `//` THEN they SHALL be normalized
4. ✅ WHEN leading slash format varies THEN `getAlternatePath()` SHALL handle it
5. ✅ WHEN documenting behavior THEN path format expectations SHALL be clear

**Implementation:** `pkg/client/aws/aws.go`

**Security Enhancements (PR #29):**
- Path traversal prevention
- Null byte rejection
- Double slash normalization
- Leading slash validation

---

### Requirement 7: AWS Organizations Discovery (#NEW)

**User Story:** As an enterprise user, I want automatic discovery of AWS accounts in my organization.

**Acceptance Criteria:**

1. WHEN discovery is enabled THEN all accounts in organization SHALL be found
2. WHEN account tags exist THEN they SHALL be used for filtering
3. WHEN delegated administrator is configured THEN discovery SHALL use that role
4. WHEN organizational units are specified THEN only accounts in those OUs SHALL be discovered
5. WHEN discovery completes THEN account IDs and names SHALL be available for target generation
6. WHEN discovery fails THEN clear error SHALL explain permission requirements

**Configuration:**
```yaml
discovery:
  enabled: true
  type: aws_organizations
  filters:
    - tag: Environment
      values: [production, staging]
    - ou: ou-prod-xxxx
  role_arn: arn:aws:iam::123456789012:role/OrgDiscoveryRole
```

**Implementation Notes:**
- Package: `pkg/discovery/organizations`
- Use AWS Organizations API
- Support tag-based filtering
- Cache discovered accounts (TTL: 1 hour)

---

### Requirement 8: AWS Identity Center Integration (#NEW)

**User Story:** As an Identity Center user, I want to sync permission sets and account assignments.

**Acceptance Criteria:**

1. WHEN Identity Center is configured THEN permission sets SHALL be discovered
2. WHEN account assignments exist THEN they SHALL be mapped to permission sets
3. WHEN syncing THEN permission set names SHALL be usable as secret paths
4. WHEN assignment changes THEN sync SHALL reflect updates
5. WHEN Identity Center instance is in different region THEN cross-region calls SHALL work

**Configuration:**
```yaml
discovery:
  enabled: true
  type: aws_identity_center
  instance_arn: arn:aws:sso:::instance/ssoins-xxxx
  store_arn: arn:aws:identitystore:::us-east-1:xxxx:identitystore/d-xxxx
```

**Implementation Notes:**
- Package: `pkg/discovery/identitycenter`
- Use SSO Admin and Identity Store APIs
- Map permission set ARNs to names
- Cache assignments (TTL: 30 minutes)

---

### Requirement 9: Secret Versioning Support (#NEW)

**User Story:** As a user, I want to track secret versions and roll back if needed.

**Acceptance Criteria:**

1. WHEN secrets are synced THEN version metadata SHALL be preserved
2. WHEN AWS Secrets Manager versions exist THEN latest version SHALL be used by default
3. WHEN specific version is requested THEN that version SHALL be synced
4. WHEN version history is enabled THEN previous versions SHALL be accessible
5. WHEN displaying diffs THEN version numbers SHALL be shown

**Configuration:**
```yaml
targets:
  - name: production
    type: aws_secretsmanager
    versioning:
      enabled: true
      retain_versions: 10
```

**Implementation Notes:**
- Add version tracking to diff system
- Store version metadata in S3 merge store
- Support version rollback via CLI flag
- Display version information in output

---

### Requirement 10: Enhanced Diff Output (#NEW)

**User Story:** As a user reviewing changes, I want detailed diff output with side-by-side comparison.

**Acceptance Criteria:**

1. WHEN running with `--diff` THEN changes SHALL be clearly highlighted
2. WHEN secret value changes THEN old and new values SHALL be shown (masked)
3. WHEN new secrets are added THEN they SHALL be marked with `+` prefix
4. WHEN secrets are deleted THEN they SHALL be marked with `-` prefix
5. WHEN output format is `github` THEN annotations SHALL be created for PR reviews
6. WHEN output format is `json` THEN structured diff SHALL be provided
7. WHEN large diffs occur THEN summary statistics SHALL be shown

**Output Format:**
```
Diff Summary:
  Added:    5 secrets
  Modified: 3 secrets
  Deleted:  1 secret

Changes:
  + secret/app/new-api-key
  ~ secret/app/db-password (value changed)
  - secret/app/old-token
```

**Implementation Notes:**
- Enhance `pkg/diff` package
- Support multiple output formats
- Add color coding for terminals
- Mask sensitive values by default
- Add `--show-values` flag for debugging

---

## Advanced Configuration Patterns

### Multi-Environment Inheritance

**Use Case:** Share common secrets across environments, override per-environment

**Example:**
```yaml
# Base secrets in Vault
vault_sources:
  - mount: secret/base/
  - mount: secret/production/

# Target with inheritance
targets:
  - name: production-us-east-1
    imports:
      - base_merged  # From merge store
      - production   # Env-specific
    overrides:
      region: us-east-1
```

### Dynamic Target Generation

**Use Case:** Auto-generate targets from discovered AWS accounts

**Example:**
```yaml
discovery:
  enabled: true
  type: aws_organizations
  
target_template:
  type: aws_secretsmanager
  region: "{{ account.region }}"
  role_arn: "arn:aws:iam::{{ account.id }}:role/SecretSync"
  path_prefix: "{{ account.environment }}/"
```

### Conditional Secret Sync

**Use Case:** Only sync secrets matching certain criteria

**Example:**
```yaml
targets:
  - name: production
    filters:
      - path_regex: "^production/.*"
      - tag: environment=production
      - exclude_path: ".*test.*"
```

## Non-Functional Requirements

### Scalability
- Support 10,000+ secrets per sync
- Handle 100+ AWS accounts
- Complete discovery in < 2 minutes

### Performance
- Discovery results cached appropriately
- Parallel secret fetching where possible
- Incremental updates for large configurations

### Usability
- Clear error messages for configuration issues
- Dry-run mode validates before execution
- Progress indicators for long operations

## Release Checklist

- [ ] All completed features verified
- [ ] New features implemented and tested
- [ ] Integration tests cover new patterns
- [ ] Documentation updated with examples
- [ ] CHANGELOG.md updated
- [ ] Migration guide for v1.1.0 users
- [ ] Performance tested at scale
- [ ] Security review completed
- [ ] Git tag `v1.2.0` created
- [ ] GitHub release published

## Success Metrics

- Supports organizations with 100+ AWS accounts
- Reduces configuration duplication by 80%+
- Discovery completes in < 2 minutes for large orgs
- User satisfaction score ≥ 4.5/5
- Zero security vulnerabilities in new features

