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

## Profile Issues

### "profile not found"

The specified profile doesn't exist.

**Solution:**
```bash
# List all profiles
clauderock profiles

# Create a new profile
clauderock config save my-profile

# Or switch to an existing profile
cloudrock config switch default
```

### Migration didn't work

Old `config.json` wasn't migrated to profiles.

**Solution:**
```bash
# Manually save your config as a profile
clauderock config save default

# Verify it was created
ls ~/.cloudrock/profiles/
```

### Can't delete current profile

You cannot delete the currently active profile.

**Solution:**
```bash
# Switch to a different profile first
clauderock config switch another-profile

# Then delete
clauderock config delete old-profile
```

## Stats & Database Issues

### "database is locked"

The SQLite database is locked by another process.

**Solution:**
1. Close any other clauderock processes
2. If stuck, restart your terminal
3. Last resort: `rm ~/.clauderock/usage.db` (will lose stats)

### Cache hit rate shows >100%

This was a bug in earlier versions that has been fixed.

**Solution:**
```bash
# Reset stats to clear incorrect data
clauderock stats reset

# New sessions will be tracked correctly
```

### Stats not showing recent sessions

Session JSONL file might not have been found or parsed.

**Solution:**
```bash
# Check if JSONL files exist
ls ~/.claude/projects/*/*.jsonl

# Verify working directory encoding is correct
# The directory path gets encoded with dashes replacing slashes
```

### Reset all stats

Delete all usage statistics from the database.

**Solution:**
```bash
# With confirmation dialog
clauderock stats reset

# Skip confirmation (dangerous!)
clauderock stats reset --force
```

## Session Tracking Issues

### "no JSONL files found"

Claude Code didn't create a session file, or it's in an unexpected location.

**Causes:**
- Session was too short
- Claude Code didn't start properly
- Working directory path encoding mismatch

**Solution:**
- Sessions are only tracked if Claude Code actually runs
- Very short sessions (< 1 second) may not generate JSONL files

### TPM/RPM metrics are zero

No API requests were made during the session.

**Causes:**
- Session ended before any requests
- Claude Code didn't connect to Bedrock

**Solution:**
- This is normal for very short sessions
- Only sessions with actual API calls will have TPM/RPM data

## Still Having Issues?

1. Check your configuration: `clauderock config list`
2. Check your current profile: `clauderock profiles`
3. Verify AWS access: `aws bedrock list-inference-profiles --profile YOUR_PROFILE`
4. Check stats database: `ls -lh ~/.clauderock/usage.db`
5. Open an issue: https://github.com/OlaHulleberg/clauderock/issues
