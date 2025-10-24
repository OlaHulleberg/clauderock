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

Settings are stored in profiles at `~/.clauderock/profiles/`:

| Key | Description | Example |
|-----|-------------|---------|
| `profile` | AWS profile name | `production` |
| `region` | AWS region | `us-east-1` |
| `cross-region` | Geography for routing | `us`, `eu`, `global` |
| `model` | Main model | `anthropic.claude-sonnet-4-5` |
| `fast-model` | Fast model | `anthropic.claude-haiku-4-5` |

```bash
clauderock config                    # Interactive configuration
clauderock config set <key> <value>  # Set a value
clauderock config list               # View all settings
```

### Multiple Profiles

Manage multiple named profiles for different use cases:

```bash
clauderock profiles                  # List all profiles
clauderock config save work-dev      # Save current config as profile
clauderock config switch personal    # Switch to different profile
clauderock --profile work-prod       # Use specific profile for one run
```

## Usage

```bash
# Launch Claude Code
clauderock                             # Use current profile
clauderock --profile work-dev          # Use specific profile
clauderock --model anthropic.claude-opus-4  # Override model

# Configuration
clauderock config                      # Interactive wizard
clauderock config list                 # View settings
clauderock config set model <value>    # Update setting

# Profiles
clauderock profiles                    # List profiles
clauderock config save my-profile      # Save as new profile
clauderock config switch my-profile    # Switch profile

# Models
clauderock models list                 # List available models
clauderock models list --provider anthropic  # Filter by provider

# Stats & Cost Tracking
clauderock stats                       # View usage statistics
clauderock stats --today               # Today's stats
clauderock stats --month 2025-10       # Monthly stats
clauderock stats reset                 # Clear all stats

# Updates
clauderock update                      # Update to latest version
clauderock version                     # Show version
```

### Override Flags

Override any setting for a single run without changing your saved profile:

```bash
clauderock --model anthropic.claude-opus-4
clauderock --fast-model anthropic.claude-haiku-4-5
clauderock --aws-profile production
clauderock --region us-west-2
clauderock --cross-region us
```

## What It Does

1. Fetches available cross-region inference profiles from AWS Bedrock
2. Matches your configured models to available profiles
3. Launches `claude` with the correct environment variables set
4. Tracks sessions with detailed TPM/RPM metrics and cost estimates

## Features

### 📋 Multiple Profiles
Save and switch between different configurations (work, personal, different projects).

### 🔍 Model Discovery
List all available models from AWS Bedrock with provider filtering.

### ⚡ Quick Overrides
Override any setting for a single run using command-line flags.

### 📊 Usage Tracking
Track coding sessions with:
- Token usage (input/output/cache metrics)
- TPM/RPM (Tokens/Requests Per Minute)
- Cache efficiency
- Cost estimates based on actual usage

### 🔒 Privacy-First
All usage data stored locally in `~/.clauderock/usage.db`. Never sent anywhere.

## Documentation

- **[Configuration Guide](CONFIGURATION.md)** - Detailed config options, profiles, and cross-region inference
- **[Pricing & Costs](PRICING.md)** - Cost tracking methodology and pricing estimates
- **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues and solutions
- **[Contributing](CONTRIBUTING.md)** - Development guide and release process

## Requirements

- [Claude Code](https://claude.com/claude-code) installed
- AWS credentials configured
- AWS Bedrock access with inference profiles enabled

## License

MIT
