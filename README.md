# clauderock

**Launch Claude Code with AWS Bedrock or custom API endpoints.**

Lightweight CLI for Claude Code with:
- AWS Bedrock cross-region inference
- Custom API gateway support with API key authentication

---

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/OlaHulleberg/clauderock/main/install.sh | bash
```

## Prerequisites

**For AWS Bedrock:**
- AWS CLI installed and authenticated (`aws sso login` or `aws configure`)
- See [CONFIGURATION.md](CONFIGURATION.md) for AWS setup

**For API Mode:**
- API endpoint URL and key from your provider

## Quick Start

```bash
# Launch clauderock (interactive setup runs automatically on first use)
clauderock
```

On first run, clauderock will guide you through configuration:
- Choose profile type (AWS Bedrock or API)
- Enter credentials (AWS profile or API key)
- Select models (main/fast/heavy)

## Configuration

Profiles stored at `~/.clauderock/profiles/`. Each profile is either:
- **Bedrock**: AWS cross-region inference
- **API**: Custom endpoint with API key

```bash
clauderock manage config                # Interactive wizard (full setup)
clauderock manage config models         # Change models only
clauderock manage config list           # View current settings
clauderock manage profiles              # List all profiles
clauderock manage config switch <name>  # Switch profile
```

## Usage

```bash
# Launch
clauderock                              # Use current profile (auto-setup on first run)
clauderock --clauderock-profile work    # Use specific profile

# Claude CLI passthrough (all flags pass through)
clauderock --resume                     # Resume last session
clauderock --debug                      # Debug mode

# Configuration
clauderock manage config                # Interactive wizard (full setup)
clauderock manage config models         # Change models only
clauderock manage config list           # View current settings
clauderock manage profiles              # List all profiles

# Management
clauderock manage models list           # List available models (Bedrock only)
clauderock manage stats                 # Usage statistics
clauderock manage update                # Update to latest version
clauderock manage version               # Show version
```

### Override Flags

Temporary overrides without changing saved profile:

```bash
# For Bedrock profiles
--clauderock-aws-profile <profile>
--clauderock-region <region>
--clauderock-cross-region <us|eu|global>

# For API profiles
--clauderock-base-url <url>
--clauderock-api-key <key>

# All profiles
--clauderock-model <model-id>
--clauderock-fast-model <model-id>
--clauderock-heavy-model <model-id>
```

## Features

- **Auto-configuration**: Interactive setup runs automatically on first launch
- **Dual mode**: AWS Bedrock or custom API endpoints
- **Multiple profiles**: Switch between configurations (work/personal/projects)
- **Model selection**: Choose main/fast/heavy models per profile (can update independently)
- **Usage tracking**: Token metrics, TPM/RPM, cost estimates (stored locally)
- **Secure storage**: API keys in OS keychain (macOS/Linux/Windows)
- **Override flags**: Temporary config changes without saving
- **Passthrough**: All Claude CLI flags work (`--resume`, `--debug`, etc.)

## Documentation

- **[Configuration Guide](CONFIGURATION.md)** - Detailed config options, profiles, and cross-region inference
- **[Pricing & Costs](PRICING.md)** - Cost tracking methodology and pricing estimates
- **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues and solutions
- **[Contributing](CONTRIBUTING.md)** - Development guide and release process

## Requirements

- [Claude Code](https://claude.com/claude-code) installed
- **For AWS Bedrock**: AWS CLI installed, configured, and authenticated with Bedrock access
- **For API Mode**: API endpoint URL and key from your provider

## License

MIT
