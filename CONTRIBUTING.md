# Contributing

Contributions are welcome! This guide covers development, building, and releasing.

## Development Setup

### Prerequisites

- Go 1.23 or later
- AWS credentials with Bedrock access (for testing)
- Claude Code installed (for integration testing)

### Clone and Build

```bash
git clone https://github.com/OlaHulleberg/clauderock.git
cd clauderock
go build -o clauderock
```

### Project Structure

```
clauderock/
├── cmd/                 # CLI commands (Cobra)
│   ├── root.go          # Main command with override flags
│   ├── config.go        # Configuration commands
│   ├── profiles.go      # Profile management
│   ├── models.go        # Model listing
│   ├── stats.go         # Usage statistics
│   └── stats_reset.go   # Reset stats
├── internal/
│   ├── aws/             # AWS Bedrock integration
│   ├── config/          # Configuration structs
│   ├── profiles/        # Profile manager
│   ├── interactive/     # Bubbletea UI components
│   ├── launcher/        # Claude Code launcher
│   ├── monitoring/      # JSONL parser for session tracking
│   ├── usage/           # SQLite usage tracking
│   ├── pricing/         # Cost calculation
│   └── updater/         # Auto-update system
├── main.go              # Entry point
├── install.sh           # Installation script
└── .goreleaser.yml      # Release configuration
```

## Running Tests

```bash
go test ./...
```

### Manual Testing

```bash
# Build
go build -o clauderock

# Test configuration
./clauderock config set profile test
./clauderock config list

# Test with verbose output
./clauderock version
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions focused and small
- Add comments for exported functions

## Making Changes

1. **Fork the repository**
2. **Create a feature branch:**
   ```bash
   git checkout -b feature/my-new-feature
   ```
3. **Make your changes**
4. **Test thoroughly**
5. **Commit with clear messages:**
   ```bash
   git commit -m "Add support for X"
   ```
6. **Push and create a PR:**
   ```bash
   git push origin feature/my-new-feature
   ```

## Release Process

Releases are automated using GoReleaser and GitHub Actions.

### Creating a Release

1. **Update version references** (if needed)
2. **Create and push a tag:**
   ```bash
   git tag v0.2.0
   git push origin v0.2.0
   ```
3. **GitHub Actions automatically:**
   - Builds binaries for all platforms
   - Creates archives (tar.gz, zip)
   - Generates checksums
   - Publishes GitHub release
   - Uploads all artifacts

### Supported Platforms

- **Linux:** amd64, arm64
- **macOS:** amd64, arm64
- **Windows:** amd64, arm64

### Version Format

Follow semantic versioning: `vMAJOR.MINOR.PATCH`

- **MAJOR:** Breaking changes
- **MINOR:** New features, backward compatible
- **PATCH:** Bug fixes

## Auto-Update System

The auto-update system:

1. **On launch:** Checks GitHub API for latest release (background)
2. **On update command:** Downloads appropriate archive
3. **Extracts binary** from tar.gz or zip
4. **Replaces current executable**

### Testing Updates

```bash
# Build with version set
go build -ldflags "-X github.com/OlaHulleberg/clauderock/cmd.Version=v0.1.0" -o clauderock

# This will detect v0.2.0 as newer
./clauderock update
```

## Adding Dependencies

```bash
go get <package>
go mod tidy
```

Update `go.mod` and `go.sum` are committed.

## Documentation

When adding features:

1. Update **README.md** if it affects getting started
2. Update **CONFIGURATION.md** for config changes
3. Update **TROUBLESHOOTING.md** for new error cases
4. Add comments to exported functions

## Questions?

- Open an issue for discussion
- Check existing issues and PRs
- Reach out to maintainers

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
