# Troubleshooting

## "claude binary not found in PATH"

Claude Code is not installed or not in your PATH.

**Solution:**
1. Install Claude Code: https://claude.com/claude-code
2. Verify it's in your PATH:
   ```bash
   which claude
   ```

## "failed to load AWS config"

AWS credentials are not configured or the profile doesn't exist.

**Solutions:**

1. **Configure AWS credentials:**
   ```bash
   aws configure --profile YOUR_PROFILE
   ```

2. **Check if profile exists:**
   ```bash
   cat ~/.aws/credentials
   ```

3. **Verify AWS CLI works:**
   ```bash
   aws sts get-caller-identity --profile YOUR_PROFILE
   ```

## "could not find inference profile"

The tool couldn't find a matching inference profile for your configuration.

**Common causes:**
- Cross-region setting doesn't match available profiles
- Model not available in your region
- No Bedrock access for inference profiles

**Solution:**

The error message shows available profiles. For example:

```
Error: main model: could not find inference profile for model 'claude-sonnet-4-5' with cross-region 'global'
Available profiles:
  - us.anthropic.claude-sonnet-4-5-20250929-v1:0
  - eu.anthropic.claude-sonnet-4-5-20250929-v1:0
  - global.anthropic.claude-haiku-4-5-20251001-v1:0
```

In this case, change your cross-region or model:
```bash
clauderock config set cross-region us
# OR
clauderock config set model claude-haiku-4-5
```

## "cannot update development build"

You're running a development build (`version dev`) which cannot self-update.

**Solution:**
- Install from releases instead of building locally
- Or manually update your build

## Rate Limiting / Throttling

AWS Bedrock has rate limits. If you hit them:

**Solutions:**
- Wait a few seconds and retry
- Use a different region if available
- Check your AWS Service Quotas for Bedrock

## Access Denied Errors

Your AWS credentials don't have permission to access Bedrock.

**Required IAM permissions:**
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

**Solution:**
1. Contact your AWS administrator
2. Request Bedrock access for your IAM user/role

## Installation Issues

### install.sh fails

**Common issues:**

1. **No internet connection:** Verify you can reach GitHub
2. **Unsupported platform:** Check supported platforms in releases
3. **Permission denied:** May need sudo for `/usr/local/bin`

**Debug:**
```bash
# Download and run manually to see errors
curl -fsSL https://raw.githubusercontent.com/OlaHulleberg/clauderock/main/install.sh -o install.sh
bash -x install.sh  # Run with debug output
```

### Binary not in PATH

After installation, `clauderock` command not found.

**Solution:**
```bash
# If installed to ~/.local/bin, add to PATH
export PATH="$PATH:$HOME/.local/bin"

# Add to your shell profile (.bashrc, .zshrc, etc.)
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc
```

## Still Having Issues?

1. Check your configuration: `clauderock config list`
2. Verify AWS access: `aws bedrock list-inference-profiles --profile YOUR_PROFILE`
3. Open an issue: https://github.com/OlaHulleberg/clauderock/issues
