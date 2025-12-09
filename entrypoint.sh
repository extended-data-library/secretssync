#!/bin/sh
# Entrypoint for SecretSync
# All configuration via environment variables - no action inputs needed
# This makes it work identically in GitHub Actions, GitLab CI, or local Docker

set -e

# Build argument list safely (no eval, no command injection)
ARGS="pipeline"

# Config file (required)
CONFIG="${SECRETSYNC_CONFIG:-config.yaml}"
ARGS="$ARGS --config $CONFIG"

# Optional: specific targets
if [ -n "$SECRETSYNC_TARGETS" ]; then
    ARGS="$ARGS --targets $SECRETSYNC_TARGETS"
fi

# Boolean flags
if [ "$SECRETSYNC_DRY_RUN" = "true" ]; then
    ARGS="$ARGS --dry-run"
fi

if [ "$SECRETSYNC_MERGE_ONLY" = "true" ]; then
    ARGS="$ARGS --merge-only"
fi

if [ "$SECRETSYNC_SYNC_ONLY" = "true" ]; then
    ARGS="$ARGS --sync-only"
fi

if [ "$SECRETSYNC_DISCOVER" = "true" ]; then
    ARGS="$ARGS --discover"
fi

if [ "$SECRETSYNC_DIFF" = "true" ]; then
    ARGS="$ARGS --diff"
fi

if [ "$SECRETSYNC_EXIT_CODE" = "true" ]; then
    ARGS="$ARGS --exit-code"
fi

# Output format (default: github for Actions, human otherwise)
OUTPUT="${SECRETSYNC_OUTPUT:-github}"
ARGS="$ARGS --output $OUTPUT"

# Logging
LOG_LEVEL="${SECRETSYNC_LOG_LEVEL:-info}"
ARGS="$ARGS --log-level $LOG_LEVEL"

LOG_FORMAT="${SECRETSYNC_LOG_FORMAT:-text}"
ARGS="$ARGS --log-format $LOG_FORMAT"

# Debug mode - print command
if [ "$LOG_LEVEL" = "debug" ] || [ "$SECRETSYNC_DEBUG" = "true" ]; then
    echo "Executing: secretsync $ARGS"
fi

# Execute directly without eval to prevent command injection
# shellcheck disable=SC2086
exec secretsync $ARGS
