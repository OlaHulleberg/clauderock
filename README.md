# clauderock

**Launch Claude Code with AWS Bedrock in one command.**

A lightweight CLI that configures Claude Code to use AWS Bedrock's cross-region inference profiles automatically.

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/OlaHulleberg/clauderock/main/install.sh | bash
```

## Prerequisites & Setup

Before using clauderock, ensure you have AWS CLI installed and authenticated:

### 1. Install AWS CLI

Follow the [AWS CLI installation guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) for your platform.

### 2. Authenticate with AWS

You must be logged in to AWS using one of these methods:

**Option A: AWS SSO (recommended for teams)**
```bash
aws sso login --profile your-profile
```

**Option B: Static Credentials**
```bash
aws configure --profile your-profile
# Enter your Access Key ID, Secret Access Key, and default region
```

### 3. Verify Authentication

```bash
aws sts get-caller-identity --profile your-profile
```

If this command succeeds, you're ready to use clauderock.

For detailed AWS setup and IAM permissions, see [CONFIGURATION.md](CONFIGURATION.md) and [TROUBLESHOOTING.md](TROUBLESHOOTING.md).

## Quick Start

```bash
# Interactive configuration (recommended)
clauderock manage config

# Or configure manually
clauderock manage config set profile my-aws-profile
clauderock manage config set region us-east-1
clauderock manage config set cross-region global

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
clauderock manage config                    # Interactive configuration
clauderock manage config set <key> <value>  # Set a value
clauderock manage config list               # View all settings
```

### Multiple Profiles

Manage multiple named profiles for different use cases:

```bash
clauderock manage profiles                  # List all profiles
clauderock manage config save work-dev      # Save current config as profile
clauderock manage config switch personal    # Switch to different profile
clauderock --clauderock-profile work-prod   # Use specific profile for one run
```

## Usage

```bash
# Launch Claude Code
clauderock                                        # Use current profile
clauderock --clauderock-profile work-dev          # Use specific profile
clauderock --clauderock-model anthropic.claude-opus-4  # Override model

# Pass Claude CLI flags
clauderock --resume                               # Resume last session
clauderock --continue                             # Continue last session
clauderock --debug                                # Debug mode
clauderock --print "analyze this code"            # Non-interactive mode

# Combined (clauderock config + Claude CLI passthrough)
clauderock --clauderock-profile work --resume --debug

# Configuration
clauderock manage config                          # Interactive wizard
clauderock manage config list                     # View settings
clauderock manage config set model <value>        # Update setting

# Profiles
clauderock manage profiles                        # List profiles
clauderock manage config save my-profile          # Save as new profile
clauderock manage config switch my-profile        # Switch profile

# Models
clauderock manage models list                     # List available models
clauderock manage models list --provider anthropic  # Filter by provider

# Stats & Cost Tracking
clauderock manage stats                           # View usage statistics
clauderock manage stats --today                   # Today's stats
clauderock manage stats --month 2025-10           # Monthly stats
clauderock manage stats reset                     # Clear all stats

# Updates
clauderock manage update                          # Update to latest version
clauderock manage version                         # Show version
```

### Override Flags

Override any setting for a single run without changing your saved profile:

```bash
clauderock --clauderock-model anthropic.claude-opus-4
clauderock --clauderock-fast-model anthropic.claude-haiku-4-5
clauderock --clauderock-aws-profile production
clauderock --clauderock-region us-west-2
clauderock --clauderock-cross-region us
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

### 🔄 Claude CLI Passthrough
Pass any Claude CLI flags and commands directly through clauderock (e.g., `--resume`, `--debug`, `--print`).

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
- [AWS CLI](https://aws.amazon.com/cli/) installed and configured
- AWS credentials authenticated (via `aws sso login` or static credentials)
- AWS Bedrock access with inference profiles enabled

## License

MIT
