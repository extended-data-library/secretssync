# .kiro Directory

This directory contains agent instructions, specifications, and quality hooks for AI-assisted development of SecretSync.

## Structure

```
.kiro/
├── steering/           # Development philosophy and standards
│   ├── 00-production-release-focus.md
│   ├── 01-golang-standards.md
│   └── 02-testing-requirements.md
├── specs/              # Feature specifications by milestone
│   ├── secretsync-complete/        # Complete system specification
│   │   ├── design.md               # Architecture and design decisions
│   │   ├── requirements.md         # Functional and non-functional requirements
│   │   └── tasks.md                # Implementation tasks by milestone
│   ├── v1.1.0-observability/
│   │   └── requirements.md
│   └── v1.2.0-advanced-features/
│       └── requirements.md
├── hooks/              # Code quality and security hooks
│   ├── go-security-scanner.kiro.hook
│   ├── go-code-quality.kiro.hook
│   └── docs-consistency.kiro.hook
└── settings/           # MCP server configuration
    └── mcp.json
```

## Purpose

### Steering Documents (`steering/`)

High-level development philosophy and coding standards that all contributors (human and AI) should follow:

- **00-production-release-focus.md** - Project mission, release priorities, anti-patterns to avoid
- **01-golang-standards.md** - Go coding conventions, patterns, error handling, testing
- **02-testing-requirements.md** - Testing philosophy, coverage requirements, test patterns

### Specifications (`specs/`)

Detailed requirements for each milestone, organized by version:

- **secretsync-complete/** - Complete system specification
  - `design.md` - System architecture, component breakdown, data flow
  - `requirements.md` - Functional and non-functional requirements
  - `tasks.md` - Implementation tasks organized by milestone
- **v1.1.0-observability/** - Observability, reliability, security hardening
- **v1.2.0-advanced-features/** - Advanced enterprise features

Each milestone spec includes:
- User stories with acceptance criteria
- Implementation notes
- Configuration examples
- Non-functional requirements
- Release checklists

The complete spec provides:
- High-level architecture diagrams
- Component design details
- End-to-end workflows
- Discrete implementation tasks
- Task dependencies and prioritization

### Hooks (`hooks/`)

Code quality automation hooks that trigger on file edits:

- **go-security-scanner.kiro.hook** - Detects security issues (command injection, path traversal, etc.)
- **go-code-quality.kiro.hook** - Enforces Go idioms and best practices
- **docs-consistency.kiro.hook** - Ensures docs match code

### Settings (`settings/`)

Configuration for development tools:

- **mcp.json** - Model Context Protocol server configuration

## Usage

### For AI Agents

Before starting work:
1. Read `steering/00-production-release-focus.md` for context
2. Review relevant spec in `specs/[version]/requirements.md`
3. Check `AGENTS.md` in project root for workflow
4. Read `memory-bank/activeContext.md` for current session state

During work:
- Hooks will automatically review code changes
- Follow standards in `steering/` documents
- Refer to specs for detailed requirements

### For Human Developers

These documents are also valuable for:
- Understanding project philosophy and goals
- Learning coding standards and patterns
- Reviewing feature requirements before implementation
- Onboarding new contributors

## Philosophy

This structure is inspired by the rivermarsh project's `.kiro` setup, adapted for a production Go project. The goal is to:

1. **Prevent common mistakes** - Clear guidelines stop agents from suggesting outdated tools or unnecessary rewrites
2. **Maintain quality** - Automated hooks catch issues early
3. **Enable autonomous work** - Detailed specs allow agents to work independently
4. **Document decisions** - Architecture and requirements are explicitly stated
5. **Ship production software** - Focus on quality over quantity

## Maintaining This Directory

### Adding New Specifications

When planning a new milestone:
1. Create directory: `.kiro/specs/vX.Y.Z-name/`
2. Add `requirements.md` with detailed acceptance criteria
3. Reference GitHub issues for traceability
4. Include examples and configuration

### Updating Standards

When patterns emerge:
1. Update relevant `steering/` document
2. Document the decision and rationale
3. Add examples of good/bad patterns
4. Update `AGENTS.md` if workflow changes

### Adding Hooks

When automating quality checks:
1. Create `.kiro.hook` file in `hooks/`
2. Specify file patterns to watch
3. Define prompt for agent review
4. Test with sample code changes

## Integration

The `.kiro` directory integrates with:
- **Cursor AI** - Reads steering docs and specs for context
- **GitHub Copilot** - Uses patterns from standards docs
- **MCP Servers** - Configured in `settings/mcp.json`
- **Memory Bank** - Cross-references with `memory-bank/activeContext.md`

## References

- Project root `AGENTS.md` - Complete agent workflow guide
- `memory-bank/activeContext.md` - Current session state and handoff protocol
- GitHub issues and PRs - Implementation tracking

---

**Remember:** This is a production open source project. All content should be professional, vendor-neutral, and focused on shipping quality software.

