# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

clauderock is a CLI tool that launches Claude Code with AWS Bedrock configuration. It manages AWS cross-region inference profiles, tracks usage statistics, and provides profile management capabilities.

## Build and Development Commands

```bash
# Build the project
go build -o clauderock

# Build with version
go build -ldflags "-X github.com/OlaHulleberg/clauderock/cmd.Version=v0.4.0" -o clauderock

# Run tests
go test ./...

# Run the binary
./clauderock

# Test configuration
./clauderock manage config list

# Add dependencies
go get <package>
go mod tidy
```

## Architecture Overview

### Core Flow

1. **Entry Point** (`main.go`): Delegates to `cmd.Execute()`
2. **Root Command** (`cmd/root.go`):
   - Parses clauderock flags and passthrough Claude CLI flags
   - Loads configuration from profiles
   - Applies flag overrides
   - Launches Claude Code via `launcher.Launch()`
3. **Launcher** (`internal/launcher/launcher.go`):
   - Sets environment variables for AWS Bedrock
   - Executes Claude CLI with passthrough args
   - Validates model profile IDs in background
   - Tracks session metrics after completion

### Configuration & Profiles System

**Profile Manager** (`internal/profiles/manager.go`):
- Stores profiles at `~/.clauderock/profiles/{name}.json`
- Tracks current profile in `~/.clauderock/current-profile.txt`
- Handles migration from legacy `config.json` to profiles
- Migrates model names to full profile IDs (v0.4.0+)

**Config Structure** (`internal/config/config.go`):
```json
{
  "version": "0.4.0",
  "profile": "default",
  "region": "us-east-1",
  "cross-region": "global",
  "model": "global.anthropic.claude-sonnet-4-5-20250929-v1:0",
  "fast-model": "global.anthropic.claude-haiku-4-5-20250929-v1:0"
}
```

**Key Concepts**:
- **Model Format Evolution**:
  - Old: `"claude-sonnet-4-5"` (friendly name)
  - v0.2.0: `"anthropic.claude-sonnet-4-5"` (with provider)
  - v0.4.0: `"global.anthropic.claude-sonnet-4-5-20250929-v1:0"` (full profile ID)
- **Full profile IDs** are cached for faster startup (no AWS API query needed)
- **Cross-region** determines routing: `us`, `eu`, or `global`

### AWS Integration

**Bedrock Client** (`internal/aws/bedrock.go`):
- Lists available inference profiles from AWS Bedrock
- Matches friendly model names to full profile IDs
- Validates profile IDs exist in AWS
- Extracts friendly names from full profile IDs

**Key Functions**:
- `FindInferenceProfiles()`: Queries AWS for model IDs (legacy, rarely used after v0.4.0)
- `ResolveModelToProfileID()`: Converts friendly name → full profile ID
- `ValidateProfileIDs()`: Validates profile IDs exist (runs in background during launch)
- `IsFullProfileID()`: Checks if string is full profile ID (starts with us./eu./global.)
- `ExtractFriendlyModelName()`: Converts full profile ID → friendly name for display

### Usage Tracking

**Session Tracking** (`internal/usage/`):
- Stores sessions in SQLite database at `~/.clauderock/usage.db`
- Parses Claude Code JSONL files for metrics (TPM, RPM, token usage, cache stats)
- Tracks per-session and aggregated statistics
- All data stored locally, never sent anywhere

**Metrics Collected**:
- Session timing, duration, exit code
- Token usage (input, output, cache read/creation)
- TPM/RPM (average, peak, P95)
- Cache hit rate
- Model and profile breakdown

**JSONL Parser** (`internal/monitoring/jsonl_parser.go`):
- Finds session JSONL in `.claude/sessions/` directory
- Parses individual request records
- Calculates rolling TPM/RPM metrics
- Computes percentiles (P95)

## Command Structure

All commands are in `cmd/` directory using Cobra:

- `root.go`: Main command with override flags
- `manage.go`: Parent command for all subcommands
- `config.go`: Configuration management
- `profiles.go`: Profile operations
- `models.go`: List available models
- `stats.go`: Usage statistics
- `stats_reset.go`: Clear statistics
- `update.go`: Auto-update system
- `version.go`: Version display

## Override Flags

All `--clauderock-*` flags in root command override config for single run:
- `--clauderock-profile`: Use specific profile
- `--clauderock-model`: Override main model (must be full profile ID)
- `--clauderock-fast-model`: Override fast model (must be full profile ID)
- `--clauderock-aws-profile`: Override AWS profile
- `--clauderock-region`: Override AWS region
- `--clauderock-cross-region`: Override cross-region setting

## Claude CLI Passthrough

The `collectPassthroughArgs()` function separates clauderock flags from Claude CLI flags. All non-clauderock flags are passed directly to Claude CLI (e.g., `--resume`, `--debug`, `--print`).

## Environment Variables Set for Bedrock

When launching Claude Code:
```bash
CLAUDE_CODE_USE_BEDROCK=1
ANTHROPIC_MODEL={full-profile-id}
ANTHROPIC_SMALL_FAST_MODEL={full-profile-id}
AWS_PROFILE={aws-profile}
AWS_REGION={region}
```

## Interactive UI

Uses Bubble Tea framework (`internal/interactive/`):
- `selector.go`: List selection interface
- `confirm.go`: Confirmation prompts
- `region.go`: Region selection
- `config.go`: Interactive configuration wizard

## Release Process

- Version set via `-ldflags` during build
- GoReleaser handles multi-platform releases
- Auto-update system (`internal/updater/`) checks GitHub releases
- Migrations run automatically based on config version field

## Important Implementation Details

1. **Model Resolution**: After v0.4.0, configs store full profile IDs. Interactive config wizard still shows friendly names for UX but saves full IDs.

2. **Background Validation**: Model profile IDs are validated in background during launch. If validation fails, Claude Code is killed and error shown.

3. **Migration Strategy**:
   - Config migrations run automatically on load
   - Profile migrations happen via `MigrateModelsToV040()`
   - Legacy config.json migrated to profiles/default.json

4. **Session Tracking**: Happens after Claude Code exits. Parser looks for JSONL file matching session start time in working directory's `.claude/sessions/` folder.

5. **Pricing Calculation** (`internal/pricing/calculator.go`): Cost estimates based on actual token usage from JSONL metrics.
