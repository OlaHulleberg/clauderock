# Configuration Guide

`clauderock` stores configuration in profiles at `~/.clauderock/profiles/`.

## AWS Setup

Before using clauderock, you must have AWS CLI installed and authenticated.

### Installing AWS CLI

Follow the official [AWS CLI installation guide](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) for your operating system:

- **macOS:** `brew install awscli` or download the installer
- **Linux:** `curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"`
- **Windows:** Download the MSI installer from AWS

### Authentication Methods

You must authenticate with AWS using one of these methods:

#### AWS SSO (Recommended for Organizations)

If your organization uses AWS SSO:

```bash
# Configure SSO profile
aws configure sso --profile your-profile

# Login (required before each session or when token expires)
aws sso login --profile your-profile
```

You'll need to re-run `aws sso login` periodically when your session expires (typically every 8-12 hours).

#### Static Credentials

For individual accounts or automation:

```bash
aws configure --profile your-profile
```

You'll be prompted for:
- **AWS Access Key ID:** Your access key
- **AWS Secret Access Key:** Your secret key
- **Default region:** e.g., `us-east-1`
- **Default output format:** `json` (recommended)

Your credentials are stored in `~/.aws/credentials`:
```ini
[your-profile]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
```

#### Temporary Credentials

For assumed roles or temporary sessions, you can also use temporary credentials. These are automatically handled by the AWS SDK.

### Verifying Authentication

Test your AWS credentials are working:

```bash
aws sts get-caller-identity --profile your-profile
```

Expected output:
```json
{
    "UserId": "AIDACKCEVSQ6C2EXAMPLE",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/your-user"
}
```

### Required IAM Permissions

Your AWS credentials must have these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock:ListInferenceProfiles",
        "bedrock:InvokeModel"
      ],
      "Resource": "*"
    }
  ]
}
```

If you get access denied errors, contact your AWS administrator to request these permissions.

### AWS Bedrock Access

Ensure AWS Bedrock is:
1. Available in your selected region
2. Enabled for your AWS account
3. Has inference profiles enabled

Check the [AWS Bedrock console](https://console.aws.amazon.com/bedrock/) to verify access.

## Profile Management

Profiles allow you to save and switch between multiple configurations for different use cases (work, personal, different projects).

### List Profiles

```bash
clauderock manage profiles
```

Shows all saved profiles and indicates which one is currently active.

### Create/Save Profile

```bash
# Save current configuration as a new profile
clauderock manage config save my-profile
```

### Switch Profile

```bash
# Switch to a different profile
clauderock manage config switch my-profile
```

The switched profile becomes the active profile for all future runs.

### Delete Profile

```bash
# Delete a profile
clauderock manage config delete my-profile
```

### Rename Profile

```bash
# Rename a profile
clauderock manage config rename old-name new-name
```

### Copy Profile

```bash
# Copy a profile to create a template
clauderock manage config copy template new-project
```

### Migration from Old Config

If you have an old `~/.clauderock/config.json`, it will automatically be migrated to `~/.clauderock/profiles/default.json` on first run. The old file is backed up as `config.json.bak`.

## Interactive Configuration (Recommended)

Run the interactive configuration wizard to set up all your settings:

```bash
clauderock manage config
```

The wizard will guide you through:
1. **AWS Profile Selection** - Choose from your available AWS profiles
2. **Region Selection** - Select the AWS region with real-time filtering
3. **Cross-Region Selection** - Choose between US, EU, or Global routing
4. **Model Selection** - Browse and filter available models from all providers
5. **Fast Model Selection** - Choose a fast model for quick operations
6. **Heavy Model Selection** - Choose a heavy model for complex tasks

Features:
- Real-time search filtering for easy navigation
- Automatically fetches available models from AWS Bedrock
- Supports multiple AI providers (Anthropic, Meta, Amazon, AI21, Cohere, Mistral, etc.)
- Shows friendly model names with provider information

## Configuration File

Each profile is stored as a separate JSON file in `~/.clauderock/profiles/`.

On first run, a default profile is created at `~/.clauderock/profiles/default.json`:

```json
{
  "profile": "default",
  "region": "us-east-1",
  "cross-region": "global",
  "model": "anthropic.claude-sonnet-4-5",
  "fast-model": "anthropic.claude-haiku-4-5",
  "heavy-model": "anthropic.claude-opus-4-1"
}
```

The current active profile is tracked in `~/.clauderock/current-profile.txt`.

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
Main model identifier in the format `provider.model-name`.

**Examples:**
- `"anthropic.claude-sonnet-4-5"` - Anthropic's Claude Sonnet
- `"anthropic.claude-opus-4"` - Anthropic's Claude Opus
- `"meta.llama-3-2-90b"` - Meta's Llama model
- `"amazon.titan-text-premier"` - Amazon's Titan model

**Note:** The interactive config automatically formats models correctly. This is matched against profile IDs like `global.anthropic.claude-sonnet-4-5-20250929-v1:0`

### `fast-model`
Fast model identifier for quick operations (used by Claude Code for certain tasks). Uses the same `provider.model-name` format.

**Examples:**
- `"anthropic.claude-haiku-4-5"` - Fast Anthropic model
- `"anthropic.claude-haiku-3-5"` - Previous fast model

### `heavy-model`
Heavy model identifier for complex tasks requiring maximum capability (used by Claude Code for demanding operations). Uses the same `provider.model-name` format.

**Examples:**
- `"anthropic.claude-opus-4-1"` - Latest Opus model (recommended)
- `"anthropic.claude-opus-4"` - Previous Opus model

## Managing Configuration

All configuration commands operate on the **current active profile**.

### Set a value

```bash
clauderock manage config set <key> <value>
```

Examples:
```bash
clauderock manage config set profile my-aws-profile
clauderock manage config set region us-east-1
clauderock manage config set cross-region global
clauderock manage config set model anthropic.claude-sonnet-4-5
clauderock manage config set fast-model anthropic.claude-haiku-4-5
clauderock manage config set heavy-model anthropic.claude-opus-4-1
```

### Get a value

```bash
clauderock manage config get <key>
```

Example:
```bash
clauderock manage config get profile
# Output: my-aws-profile
```

### List all settings

```bash
clauderock manage config list
```

Output:
```
Current Profile: work-dev

Configuration:
  profile:      my-aws-profile
  region:       us-east-1
  cross-region: global
  model:        anthropic.claude-sonnet-4-5
  fast-model:   anthropic.claude-haiku-4-5
  heavy-model:  anthropic.claude-opus-4-1
```

## Supported Models

The tool dynamically discovers available models from AWS Bedrock across multiple providers:

**Anthropic Models:**
- `anthropic.claude-sonnet-4-5` - Latest Sonnet model
- `anthropic.claude-opus-4` - Opus model
- `anthropic.claude-haiku-4-5` - Fast Haiku model

**Meta Models:**
- `meta.llama-3-2-90b` - Llama 3.2 90B
- `meta.llama-3-2-11b` - Llama 3.2 11B

**Amazon Models:**
- `amazon.titan-text-premier` - Titan Text Premier

**Other Providers:**
- AI21, Cohere, Mistral models (check AWS Bedrock console for availability)

**Note:** Availability depends on your AWS region, cross-region setting, and Bedrock access. Use the interactive config to see all available models for your setup.

## How Cross-Region Inference Works

When you run `clauderock`:

1. Connects to AWS Bedrock in your configured `region`
2. Lists all cross-region (SYSTEM_DEFINED) inference profiles
3. Filters for profiles matching: `{cross-region}.{provider}.{model}*`
4. Uses the matched profile ID to launch Claude Code

**Example profile IDs:**
- `global.anthropic.claude-sonnet-4-5-20250929-v1:0`
- `us.anthropic.claude-haiku-4-5-20251001-v1:0`
- `eu.meta.llama-3-2-90b-20251001-v1:0`
- `global.amazon.titan-text-premier-20250514-v1:0`

## Environment Variables Set

When launching Claude Code, `clauderock` sets these environment variables:

```bash
CLAUDE_CODE_USE_BEDROCK=1
ANTHROPIC_MODEL=<matched-model-profile-id>
ANTHROPIC_DEFAULT_HAIKU_MODEL=<matched-fast-model-profile-id>
AWS_PROFILE=<your-profile>
AWS_REGION=<your-region>
```

## Advanced Features

### Automatic Credential Suppression

By default, `clauderock` temporarily suppresses stored credentials during startup to prevent authentication conflict warnings. This is done by:

1. Renaming `~/.claude/.credentials.json` to `~/.claude/.credentials.json.disabled` before launching
2. Starting the process
3. Waiting 1 second (1000ms) for initialization
4. Restoring the file to `~/.claude/.credentials.json`

This ensures a clean authentication state when using Bedrock or custom API endpoints, while preserving your normal authentication for regular use.

**Why this happens:** If you've previously authenticated directly with the CLI (via `claude setup-token`), the stored credentials can conflict with the API key or Bedrock configuration that clauderock sets. The temporary suppression prevents this conflict.

**Note:** Your credentials are automatically restored after 1 second, so your normal authentication remains available for other uses.

### Disabling Credential Suppression

If you want to disable this behavior and see authentication warnings:

```bash
clauderock --clauderock-disable-auth-suppress
```

**When you might want this:**
- **Debugging authentication issues** - See which credentials are being used
- **Verifying configuration** - Confirm that the correct authentication method is active
- **Development and testing** - Troubleshoot authentication-related problems

**Note:** This flag only affects the current run and is not saved to your profile. Authentication warnings will be displayed if multiple credentials are detected.
