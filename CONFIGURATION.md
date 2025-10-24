# Configuration Guide

`clauderock` stores configuration in `~/.clauderock/config.json`.

## Configuration File

On first run, a default configuration is created:

```json
{
  "profile": "default",
  "region": "us-east-1",
  "cross-region": "global",
  "model": "claude-sonnet-4-5",
  "fast-model": "claude-haiku-4-5"
}
```

## Configuration Keys

### `profile`
AWS profile name from `~/.aws/credentials`.

**Example:** `"my-aws-profile"`, `"production"`, `"default"`

### `region`
AWS region where Bedrock is available.

**Example:** `"us-east-1"`, `"us-west-2"`, `"eu-west-1"`

### `cross-region`
Cross-region inference profile geography. Determines which AWS regions your requests can be routed to.

**Valid values:**
- `"us"` - Routes within US regions
- `"eu"` - Routes within EU regions
- `"global"` - Routes across all available regions

### `model`
Main model identifier prefix. Used to match against available inference profiles.

**Example:** `"claude-sonnet-4-5"`, `"claude-opus-4"`

**Note:** This is matched against profile IDs like `global.anthropic.claude-sonnet-4-5-20250929-v1:0`

### `fast-model`
Fast model identifier for quick operations (used by Claude Code for certain tasks).

**Example:** `"claude-haiku-4-5"`, `"claude-haiku-3-5"`

## Managing Configuration

### Set a value

```bash
clauderock config set <key> <value>
```

Examples:
```bash
clauderock config set profile my-aws-profile
clauderock config set region us-east-1
clauderock config set cross-region global
clauderock config set model claude-sonnet-4-5
clauderock config set fast-model claude-haiku-4-5
```

### Get a value

```bash
clauderock config get <key>
```

Example:
```bash
clauderock config get profile
# Output: my-aws-profile
```

### List all settings

```bash
clauderock config list
```

Output:
```
Configuration:
  profile:      my-aws-profile
  region:       us-east-1
  cross-region: global
  model:        claude-sonnet-4-5
  fast-model:   claude-haiku-4-5
```

## Supported Models

The tool dynamically discovers available models from AWS Bedrock. Common model prefixes include:

- `claude-sonnet-4-5` - Latest Sonnet model
- `claude-opus-4` - Opus model
- `claude-haiku-4-5` - Fast Haiku model
- `claude-sonnet-3-5` - Previous Sonnet version

**Note:** Availability depends on your AWS region and Bedrock access.

## How Cross-Region Inference Works

When you run `clauderock`:

1. Connects to AWS Bedrock in your configured `region`
2. Lists all cross-region (SYSTEM_DEFINED) inference profiles
3. Filters for profiles matching: `{cross-region}.anthropic.{model}*`
4. Uses the matched profile ID to launch Claude Code

**Example profile IDs:**
- `global.anthropic.claude-sonnet-4-5-20250929-v1:0`
- `us.anthropic.claude-haiku-4-5-20251001-v1:0`
- `eu.anthropic.claude-opus-4-20250514-v1:0`

## Environment Variables Set

When launching Claude Code, `clauderock` sets these environment variables:

```bash
CLAUDE_CODE_USE_BEDROCK=1
ANTHROPIC_MODEL=<matched-model-profile-id>
ANTHROPIC_SMALL_FAST_MODEL=<matched-fast-model-profile-id>
AWS_PROFILE=<your-profile>
AWS_REGION=<your-region>
```
