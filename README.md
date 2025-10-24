# clauderock

**Launch Claude Code with AWS Bedrock in one command.**

A lightweight CLI that configures Claude Code to use AWS Bedrock's cross-region inference profiles automatically.

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/OlaHulleberg/clauderock/main/install.sh | bash
```

## Quick Start

```bash
# Interactive configuration (recommended)
clauderock config

# Or configure manually
clauderock config set profile my-aws-profile
clauderock config set region us-east-1
clauderock config set cross-region global

# Launch Claude Code
clauderock
```

## Configuration

Five simple settings stored in `~/.clauderock/config.json`:

| Key | Description | Example |
|-----|-------------|---------|
| `profile` | AWS profile name | `production` |
| `region` | AWS region | `us-east-1` |
| `cross-region` | Geography for routing | `us`, `eu`, `global` |
| `model` | Main model | `anthropic.claude-sonnet-4-5` |
| `fast-model` | Fast model | `anthropic.claude-haiku-4-5` |

```bash
clauderock config set <key> <value>  # Set a value
clauderock config list               # View all settings
```

## Usage

```bash
clauderock                    # Launch with configured settings
clauderock config             # Interactive configuration wizard
clauderock --profile staging  # Override AWS profile
clauderock update             # Update to latest version
clauderock version            # Show version
```

## What It Does

1. Fetches available cross-region inference profiles from AWS Bedrock
2. Matches your configured models to available profiles
3. Launches `claude` with the correct environment variables set

## Documentation

- **[Configuration Guide](CONFIGURATION.md)** - Detailed config options and how cross-region inference works
- **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues and solutions
- **[Contributing](CONTRIBUTING.md)** - Development guide and release process

## Requirements

- [Claude Code](https://claude.com/claude-code) installed
- AWS credentials configured
- AWS Bedrock access with inference profiles enabled

## License

MIT
