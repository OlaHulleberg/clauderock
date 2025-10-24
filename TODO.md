# clauderock Feature Implementation TODO

This document contains comprehensive analysis and implementation tasks for four new features:
1. Multiple Named Profiles
2. Show Available Models Command
3. Quick Model Switching (Override Flags)
4. Cost Tracking & Usage Stats

---

## Feature 1: Multiple Named Profiles

### Overview
Allow users to save and switch between multiple named configuration profiles (e.g., `personal`, `work-dev`, `work-prod`) instead of maintaining a single global configuration.

### Architecture Decisions
- **Storage**: Separate file per profile in `~/.clauderock/profiles/` directory
- **File format**: JSON, same structure as current config
- **Current profile tracking**: Store active profile name in `~/.clauderock/current-profile.txt`
- **Backward compatibility**: Auto-migrate existing `config.json` to `profiles/default.json`

### Directory Structure
```
~/.clauderock/
├── profiles/
│   ├── default.json      # Migrated from old config.json
│   ├── personal.json
│   ├── work-dev.json
│   └── work-prod.json
├── current-profile.txt   # Contains: "default" (or active profile name)
└── usage.db             # SQLite database (for Feature 4)
```

### New Components

#### 1. Profile Manager (`internal/profiles/manager.go`)
```go
type Manager struct {
    profilesDir string
}

// Core operations
func (m *Manager) List() ([]string, error)
func (m *Manager) Load(name string) (*config.Config, error)
func (m *Manager) Save(name string, cfg *config.Config) error
func (m *Manager) Delete(name string) error
func (m *Manager) Exists(name string) bool
func (m *Manager) GetCurrent() (string, error)
func (m *Manager) SetCurrent(name string) error
func (m *Manager) GetCurrentConfig(version string) (*config.Config, error)
```

### Migration Strategy

#### Phase 1: Auto-migration on first run with new version
```go
// In internal/profiles/manager.go
func (m *Manager) MigrateFromLegacyConfig(version string) error {
    oldConfigPath := "~/.clauderock/config.json"
    if !fileExists(oldConfigPath) {
        return nil // No migration needed
    }

    // Load old config
    cfg := config.Load(version)

    // Save as "default" profile
    m.Save("default", cfg)

    // Set as current profile
    m.SetCurrent("default")

    // Rename old config to config.json.bak
    os.Rename(oldConfigPath, oldConfigPath + ".bak")

    return nil
}
```

### Command Changes

#### New Subcommands (`cmd/profiles.go`)
```bash
# Save current config as a named profile
clauderock config save --profile work-dev

# Switch to a different profile
clauderock --profile work-prod

# List all profiles
clauderock config profiles

# Delete a profile
clauderock config delete --profile old-project

# Rename a profile
clauderock config rename --from old-name --to new-name

# Copy a profile
clauderock config copy --from template --to new-project
```

#### Updated Commands
```bash
# Load/edit the current active profile
clauderock config

# Set value in current profile
clauderock config set model anthropic.claude-opus-4

# List current profile settings
clauderock config list
```

### File Modifications

#### `internal/config/config.go`
- **No changes needed** - Config struct remains the same
- Remove `configPath()` function (moved to profiles manager)
- Remove `Load()` and `Save()` functions (handled by profile manager)

#### `cmd/root.go`
```go
// Update runRoot to use profile manager
func runRoot(cmd *cobra.Command, args []string) error {
    profileMgr := profiles.NewManager()

    var cfg *config.Config
    var err error

    if profileFlag != "" {
        // Load specific profile
        cfg, err = profileMgr.Load(profileFlag)
    } else {
        // Load current profile
        cfg, err = profileMgr.GetCurrentConfig(Version)
    }

    if err != nil {
        return fmt.Errorf("failed to load profile: %w", err)
    }

    // Rest of launch logic...
}
```

#### `cmd/config.go`
- Update all config commands to use profile manager
- Add `profilesCmd`, `saveProfileCmd`, `deleteProfileCmd`, `renameProfileCmd`, `copyProfileCmd`

### Implementation Tasks

- [ ] **Task 1.1**: Create `internal/profiles/manager.go`
  - Implement `Manager` struct with profilesDir field
  - Implement `List()`, `Load()`, `Save()`, `Delete()`, `Exists()`
  - Implement `GetCurrent()`, `SetCurrent()`
  - Implement `GetCurrentConfig()` wrapper
  - Add helper functions for path resolution
  - **Dependencies**: None
  - **Estimated time**: 2-3 hours

- [ ] **Task 1.2**: Implement migration logic
  - Add `MigrateFromLegacyConfig()` to manager
  - Detect old config.json and migrate to profiles/default.json
  - Create backup of old config
  - Write migration tests
  - **Dependencies**: Task 1.1
  - **Estimated time**: 1-2 hours

- [ ] **Task 1.3**: Create `cmd/profiles.go`
  - Implement `profilesListCmd` (list all profiles)
  - Implement `profileSaveCmd` (save current as named profile)
  - Implement `profileDeleteCmd` (delete a profile)
  - Implement `profileRenameCmd` (rename a profile)
  - Implement `profileCopyCmd` (copy a profile)
  - **Dependencies**: Task 1.1
  - **Estimated time**: 2 hours

- [ ] **Task 1.4**: Update `cmd/root.go`
  - Modify `runRoot()` to use profile manager
  - Handle `--profile` flag to load specific profile
  - If no flag, load current profile
  - **Dependencies**: Task 1.1, 1.2
  - **Estimated time**: 1 hour

- [ ] **Task 1.5**: Update `cmd/config.go`
  - Modify all config commands to use profile manager
  - Update `configCmd` (interactive wizard) to save to current profile
  - Update `configSetCmd`, `configGetCmd`, `configListCmd`
  - **Dependencies**: Task 1.1, 1.4
  - **Estimated time**: 1-2 hours

- [ ] **Task 1.6**: Update `internal/interactive/config.go`
  - Modify `RunInteractiveConfig()` to accept profile manager
  - Save to current profile instead of global config
  - **Dependencies**: Task 1.1, 1.5
  - **Estimated time**: 30 minutes

- [ ] **Task 1.7**: Testing
  - Test migration from old config.json
  - Test profile CRUD operations (create, read, update, delete)
  - Test profile switching
  - Test current profile tracking
  - Test backward compatibility
  - **Dependencies**: All above tasks
  - **Estimated time**: 2 hours

- [ ] **Task 1.8**: Documentation
  - Update README.md with profile examples
  - Update CONFIGURATION.md with profile management section
  - Add migration guide
  - **Dependencies**: Task 1.7
  - **Estimated time**: 1 hour

**Total estimated time for Feature 1**: 10-13 hours

---

## Feature 2: Show Available Models Command

### Overview
Add a `models` command to list available models from AWS Bedrock without going through the interactive wizard.

### Architecture Decisions
- **Live fetching**: Always query AWS Bedrock for up-to-date models (no caching)
- **Requires**: AWS credentials and internet connection
- **Output**: Grouped by provider with friendly formatting
- **Filtering**: Support `--provider`, `--cross-region`, `--profile`, `--region` flags

### Command Design

```bash
# List all models (uses current profile's region and cross-region)
clauderock models list

# List models for specific provider
clauderock models list --provider anthropic
clauderock models list --provider meta

# List models for specific cross-region
clauderock models list --cross-region us
clauderock models list --cross-region eu

# Override AWS profile and region
clauderock models list --profile production --region us-west-2

# Combination
clauderock models list --provider anthropic --cross-region global --region us-east-1
```

### Output Format

```
Available models in us-east-1 (global cross-region):

Anthropic:
  • anthropic.claude-opus-4
  • anthropic.claude-sonnet-4-5 (recommended)
  • anthropic.claude-haiku-4-5 (fast)

Meta:
  • meta.llama-3-2-90b
  • meta.llama-3-2-11b

Amazon:
  • amazon.titan-text-premier

AI21:
  • ai21.jamba-instruct

Found 7 models across 4 providers.
```

### New Components

#### 1. Models Command (`cmd/models.go`)
```go
var modelsCmd = &cobra.Command{
    Use:   "models",
    Short: "Manage and list available models",
}

var modelsListCmd = &cobra.Command{
    Use:   "list",
    Short: "List available models from AWS Bedrock",
    RunE:  runModelsList,
}

var (
    providerFilter     string
    crossRegionFilter  string
    profileFilter      string
    regionFilter       string
)
```

#### 2. Enhanced AWS Functions (`internal/aws/bedrock.go`)
```go
// ModelInfo contains detailed model information
type ModelInfo struct {
    Name     string // e.g., "anthropic.claude-sonnet-4-5"
    Provider string // e.g., "anthropic"
    Model    string // e.g., "claude-sonnet-4-5"
}

// GetAvailableModelsDetailed fetches models with metadata
func GetAvailableModelsDetailed(profile, region, crossRegion string) ([]ModelInfo, error)

// GroupModelsByProvider groups models by their provider
func GroupModelsByProvider(models []ModelInfo) map[string][]ModelInfo
```

### File Modifications

#### New: `cmd/models.go`
```go
func runModelsList(cmd *cobra.Command, args []string) error {
    // Load profile or use flags
    var profile, region, crossRegion string

    if profileFilter != "" {
        // Load from specified profile
        profileMgr := profiles.NewManager()
        cfg, err := profileMgr.Load(profileFilter)
        if err != nil {
            return fmt.Errorf("failed to load profile: %w", err)
        }
        profile = cfg.Profile
        region = cfg.Region
        crossRegion = cfg.CrossRegion
    } else {
        // Use current profile
        profileMgr := profiles.NewManager()
        cfg, err := profileMgr.GetCurrentConfig(Version)
        if err != nil {
            return fmt.Errorf("failed to load config: %w", err)
        }
        profile = cfg.Profile
        region = cfg.Region
        crossRegion = cfg.CrossRegion
    }

    // Override with flags if provided
    if regionFilter != "" {
        region = regionFilter
    }
    if crossRegionFilter != "" {
        crossRegion = crossRegionFilter
    }

    // Fetch models
    models, err := aws.GetAvailableModelsDetailed(profile, region, crossRegion)
    if err != nil {
        return fmt.Errorf("failed to fetch models: %w", err)
    }

    // Filter by provider if specified
    if providerFilter != "" {
        models = filterByProvider(models, providerFilter)
    }

    // Group and display
    grouped := aws.GroupModelsByProvider(models)
    displayModels(grouped, region, crossRegion)

    return nil
}
```

#### Enhanced: `internal/aws/bedrock.go`
- Add `ModelInfo` struct
- Add `GetAvailableModelsDetailed()` function
- Add `GroupModelsByProvider()` helper
- Keep existing `GetAvailableModels()` for backward compatibility

### Implementation Tasks

- [ ] **Task 2.1**: Create `cmd/models.go`
  - Implement `modelsCmd` parent command
  - Implement `modelsListCmd` with flags
  - Add `--provider`, `--cross-region`, `--profile`, `--region` flags
  - Implement `runModelsList()` function
  - **Dependencies**: Feature 1 (profile manager)
  - **Estimated time**: 2 hours

- [ ] **Task 2.2**: Enhance `internal/aws/bedrock.go`
  - Create `ModelInfo` struct
  - Implement `GetAvailableModelsDetailed()`
  - Implement `GroupModelsByProvider()`
  - Add provider name extraction logic
  - **Dependencies**: None
  - **Estimated time**: 1-2 hours

- [ ] **Task 2.3**: Implement display formatting
  - Create `displayModels()` function in cmd/models.go
  - Group models by provider
  - Add visual indicators (•, recommended, fast)
  - Add summary line (count and providers)
  - **Dependencies**: Task 2.1, 2.2
  - **Estimated time**: 1 hour

- [ ] **Task 2.4**: Implement filtering logic
  - Add `filterByProvider()` helper
  - Test filtering with multiple providers
  - Validate cross-region values
  - **Dependencies**: Task 2.2
  - **Estimated time**: 1 hour

- [ ] **Task 2.5**: Register command
  - Add `modelsCmd` to root command in `cmd/root.go`
  - Update init() function
  - **Dependencies**: Task 2.1
  - **Estimated time**: 15 minutes

- [ ] **Task 2.6**: Testing
  - Test model listing across different regions
  - Test provider filtering
  - Test cross-region filtering
  - Test with multiple AWS profiles
  - Test error handling (no credentials, no access)
  - **Dependencies**: All above tasks
  - **Estimated time**: 1-2 hours

- [ ] **Task 2.7**: Documentation
  - Update README.md with `models list` examples
  - Add MODELS.md guide (optional)
  - Document filtering options
  - **Dependencies**: Task 2.6
  - **Estimated time**: 30 minutes

**Total estimated time for Feature 2**: 6-8 hours

---

## Feature 3: Quick Model Switching (Override Flags)

### Overview
Allow users to override configuration settings for a single run using command-line flags without modifying saved configuration.

### Architecture Decisions
- **Scope**: Override flags apply only to the current execution
- **No persistence**: Flags do not modify saved profile
- **Validation**: Verify overridden models exist in AWS Bedrock before launching
- **Priority**: Flags > Profile > Defaults

### Flag Design

```bash
# Override main model
clauderock --model anthropic.claude-opus-4

# Override fast model
clauderock --fast-model anthropic.claude-haiku-4-5

# Override AWS settings
clauderock --aws-profile production --region us-west-2

# Override cross-region
clauderock --cross-region us

# Combine multiple overrides
clauderock --model anthropic.claude-opus-4 \
           --fast-model anthropic.claude-haiku-4-5 \
           --cross-region global \
           --region us-east-1

# Still supports profile switching (from Feature 1)
clauderock --profile work-dev --model anthropic.claude-opus-4
```

### Flag Priority
1. Command-line flags (highest priority)
2. Specified profile (`--profile` flag)
3. Current active profile
4. Defaults (lowest priority)

### File Modifications

#### `cmd/root.go`
```go
var (
    // Existing flags
    profileFlag string

    // New override flags
    modelFlag       string
    fastModelFlag   string
    awsProfileFlag  string
    regionFlag      string
    crossRegionFlag string
)

func init() {
    // Existing
    rootCmd.Flags().StringVar(&profileFlag, "profile", "", "Override profile for this run")

    // New flags
    rootCmd.Flags().StringVar(&modelFlag, "model", "", "Override main model for this run")
    rootCmd.Flags().StringVar(&fastModelFlag, "fast-model", "", "Override fast model for this run")
    rootCmd.Flags().StringVar(&awsProfileFlag, "aws-profile", "", "Override AWS profile for this run")
    rootCmd.Flags().StringVar(&regionFlag, "region", "", "Override AWS region for this run")
    rootCmd.Flags().StringVar(&crossRegionFlag, "cross-region", "", "Override cross-region setting for this run")
}

func runRoot(cmd *cobra.Command, args []string) error {
    // Check for updates in background
    go updater.CheckForUpdates(Version)

    // Load configuration from profile
    profileMgr := profiles.NewManager()
    var cfg *config.Config
    var err error

    if profileFlag != "" {
        cfg, err = profileMgr.Load(profileFlag)
    } else {
        cfg, err = profileMgr.GetCurrentConfig(Version)
    }
    if err != nil {
        return fmt.Errorf("failed to load profile: %w", err)
    }

    // Apply overrides from flags
    if awsProfileFlag != "" {
        cfg.Profile = awsProfileFlag
    }
    if regionFlag != "" {
        cfg.Region = regionFlag
    }
    if crossRegionFlag != "" {
        cfg.CrossRegion = crossRegionFlag
    }
    if modelFlag != "" {
        cfg.Model = modelFlag
    }
    if fastModelFlag != "" {
        cfg.FastModel = fastModelFlag
    }

    // Validate configuration (including overrides)
    if err := cfg.Validate(); err != nil {
        return fmt.Errorf("invalid configuration: %w", err)
    }

    // Find inference profile IDs (this will validate model availability)
    mainModelID, fastModelID, err := aws.FindInferenceProfiles(cfg)
    if err != nil {
        return fmt.Errorf("failed to find inference profiles: %w", err)
    }

    // Show what we're using (helpful when overrides are in effect)
    if modelFlag != "" || fastModelFlag != "" || awsProfileFlag != "" ||
       regionFlag != "" || crossRegionFlag != "" {
        fmt.Println("Using overrides:")
        if awsProfileFlag != "" {
            fmt.Printf("  AWS Profile: %s\n", cfg.Profile)
        }
        if regionFlag != "" {
            fmt.Printf("  Region: %s\n", cfg.Region)
        }
        if crossRegionFlag != "" {
            fmt.Printf("  Cross Region: %s\n", cfg.CrossRegion)
        }
        if modelFlag != "" {
            fmt.Printf("  Model: %s\n", cfg.Model)
        }
        if fastModelFlag != "" {
            fmt.Printf("  Fast Model: %s\n", cfg.FastModel)
        }
        fmt.Println()
    }

    fmt.Printf("Using model: %s\n", mainModelID)
    fmt.Printf("Using fast model: %s\n", fastModelID)

    // Launch Claude Code
    return launcher.Launch(cfg, mainModelID, fastModelID)
}
```

### Implementation Tasks

- [ ] **Task 3.1**: Add override flags to root command
  - Add flag definitions in `cmd/root.go`
  - Add `--model`, `--fast-model`, `--aws-profile`, `--region`, `--cross-region`
  - **Dependencies**: None
  - **Estimated time**: 30 minutes

- [ ] **Task 3.2**: Implement flag override logic
  - Modify `runRoot()` to apply flag overrides after loading profile
  - Implement priority: flags > profile > defaults
  - **Dependencies**: Task 3.1, Feature 1
  - **Estimated time**: 1 hour

- [ ] **Task 3.3**: Add override display feedback
  - Show which settings are being overridden
  - Display final effective configuration
  - Make output clear and concise
  - **Dependencies**: Task 3.2
  - **Estimated time**: 30 minutes

- [ ] **Task 3.4**: Enhance validation
  - Ensure `cfg.Validate()` catches invalid overrides
  - Validate cross-region values
  - Let AWS Bedrock API validate model availability
  - **Dependencies**: Task 3.2
  - **Estimated time**: 30 minutes

- [ ] **Task 3.5**: Testing
  - Test each flag individually
  - Test flag combinations
  - Test invalid values (should error before launch)
  - Test model availability validation
  - Test flag + profile combination
  - **Dependencies**: All above tasks
  - **Estimated time**: 1-2 hours

- [ ] **Task 3.6**: Documentation
  - Update README.md with override examples
  - Add use cases (testing different models, one-off switches)
  - Update help text for clarity
  - **Dependencies**: Task 3.5
  - **Estimated time**: 30 minutes

**Total estimated time for Feature 3**: 4-5 hours

---

## Feature 4: Cost Tracking & Usage Stats

### Overview
Track clauderock usage (requests, models used, timestamps) in a local SQLite database and provide a `stats` command to view usage statistics with estimated costs.

### Architecture Decisions
- **Storage**: SQLite database at `~/.clauderock/usage.db`
- **Tracking**: Log each launch with timestamp, profile, model, fast-model
- **Cost calculation**: Maintain pricing table, calculate at display time (not at tracking time)
- **Privacy**: All data stored locally, never sent anywhere
- **Token tracking**: Capture from environment or Claude Code output if available

### Database Schema

```sql
-- usage.db
CREATE TABLE launches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    profile_name TEXT NOT NULL,
    aws_profile TEXT NOT NULL,
    region TEXT NOT NULL,
    cross_region TEXT NOT NULL,
    model TEXT NOT NULL,
    model_profile_id TEXT NOT NULL,
    fast_model TEXT NOT NULL,
    fast_model_profile_id TEXT NOT NULL
);

CREATE INDEX idx_timestamp ON launches(timestamp);
CREATE INDEX idx_profile_name ON launches(profile_name);
CREATE INDEX idx_model ON launches(model);

-- Optional: token tracking (if we can capture it)
CREATE TABLE token_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    launch_id INTEGER NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    input_tokens INTEGER,
    output_tokens INTEGER,
    FOREIGN KEY (launch_id) REFERENCES launches(id)
);
```

### Pricing Table

```go
// internal/pricing/calculator.go
type ModelPrice struct {
    Provider    string
    Model       string
    InputCost   float64 // Cost per 1M input tokens
    OutputCost  float64 // Cost per 1M output tokens
}

var PricingTable = map[string]ModelPrice{
    "anthropic.claude-opus-4": {
        Provider:   "anthropic",
        Model:      "claude-opus-4",
        InputCost:  15.00,  // $15 per 1M tokens
        OutputCost: 75.00,  // $75 per 1M tokens
    },
    "anthropic.claude-sonnet-4-5": {
        Provider:   "anthropic",
        Model:      "claude-sonnet-4-5",
        InputCost:  3.00,   // $3 per 1M tokens
        OutputCost: 15.00,  // $15 per 1M tokens
    },
    "anthropic.claude-haiku-4-5": {
        Provider:   "anthropic",
        Model:      "claude-haiku-4-5",
        InputCost:  0.80,   // $0.80 per 1M tokens
        OutputCost: 4.00,   // $4 per 1M tokens
    },
    // Add more models...
}

func CalculateCost(model string, inputTokens, outputTokens int64) float64
```

### New Components

#### 1. Usage Tracker (`internal/usage/tracker.go`)
```go
type Tracker struct {
    db *Database
}

func NewTracker() (*Tracker, error)

func (t *Tracker) TrackLaunch(cfg *config.Config, mainModelID, fastModelID string) error

func (t *Tracker) GetStats(filter StatsFilter) (*Stats, error)

type StatsFilter struct {
    ProfileName  string
    StartDate    time.Time
    EndDate      time.Time
    Model        string
}

type Stats struct {
    TotalLaunches int
    ModelBreakdown map[string]int
    ProfileBreakdown map[string]int
    EstimatedCost float64
}
```

#### 2. Database Layer (`internal/usage/database.go`)
```go
type Database struct {
    db *sql.DB
}

func NewDatabase(path string) (*Database, error)
func (d *Database) Init() error
func (d *Database) InsertLaunch(launch Launch) error
func (d *Database) QueryLaunches(filter StatsFilter) ([]Launch, error)
func (d *Database) Close() error
```

#### 3. Pricing Calculator (`internal/pricing/calculator.go`)
```go
var PricingTable map[string]ModelPrice

func CalculateCost(model string, inputTokens, outputTokens int64) float64
func GetModelPrice(model string) (ModelPrice, bool)
```

### Stats Command

```bash
# View overall stats
clauderock stats

# Stats for specific profile
clauderock stats --profile work-dev

# Stats for specific model
clauderock stats --model anthropic.claude-opus-4

# Stats for time period
clauderock stats --since 2025-10-01
clauderock stats --since 2025-10-01 --until 2025-10-31
clauderock stats --month 2025-10
clauderock stats --today
clauderock stats --week

# Detailed output
clauderock stats --detailed

# Export to CSV
clauderock stats --export usage-report.csv
```

### Stats Output Format

```
Usage Statistics (All Time):

Total Launches: 156

By Profile:
  work-dev:  89 launches (57%)
  personal:  45 launches (29%)
  work-prod: 22 launches (14%)

By Model:
  anthropic.claude-sonnet-4-5: 112 launches (72%)
  anthropic.claude-opus-4:      34 launches (22%)
  anthropic.claude-haiku-4-5:   10 launches (6%)

Estimated Costs (based on average usage):
  Note: Actual costs may vary based on token usage

  anthropic.claude-sonnet-4-5: ~$45.00
  anthropic.claude-opus-4:     ~$89.00
  anthropic.claude-haiku-4-5:  ~$2.50

  Total Estimated Cost: ~$136.50

Most Active Days:
  2025-10-24: 23 launches
  2025-10-23: 18 launches
  2025-10-22: 15 launches
```

### File Modifications

#### `internal/launcher/launcher.go`
```go
func Launch(cfg *config.Config, mainModelID, fastModelID string) error {
    // Track usage BEFORE launching
    tracker, err := usage.NewTracker()
    if err == nil {
        // Don't fail launch if tracking fails
        _ = tracker.TrackLaunch(cfg, mainModelID, fastModelID)
    }

    // Find claude binary
    claudePath, err := exec.LookPath("claude")
    if err != nil {
        return fmt.Errorf("claude binary not found in PATH: %w", err)
    }

    // ... rest of existing launch logic
}
```

#### New: `cmd/stats.go`
```go
var statsCmd = &cobra.Command{
    Use:   "stats",
    Short: "View usage statistics and estimated costs",
    RunE:  runStats,
}

var (
    statsProfile  string
    statsModel    string
    statsSince    string
    statsUntil    string
    statsMonth    string
    statsToday    bool
    statsWeek     bool
    statsDetailed bool
    statsExport   string
)
```

#### `go.mod`
```go
require (
    // ... existing dependencies
    github.com/mattn/go-sqlite3 v1.14.22
)
```

### Implementation Tasks

- [ ] **Task 4.1**: Create `internal/usage/database.go`
  - Implement `Database` struct
  - Implement `NewDatabase()` with SQLite initialization
  - Implement schema creation (`Init()`)
  - Implement `InsertLaunch()`, `QueryLaunches()`, `Close()`
  - Add indexes for performance
  - **Dependencies**: None
  - **Estimated time**: 2-3 hours

- [ ] **Task 4.2**: Create `internal/usage/tracker.go`
  - Implement `Tracker` struct
  - Implement `NewTracker()`
  - Implement `TrackLaunch()` to insert usage records
  - Implement `GetStats()` to query and aggregate data
  - **Dependencies**: Task 4.1
  - **Estimated time**: 2 hours

- [ ] **Task 4.3**: Create `internal/pricing/calculator.go`
  - Define `ModelPrice` struct
  - Create `PricingTable` with current AWS Bedrock pricing
  - Implement `CalculateCost()` function
  - Implement `GetModelPrice()` lookup
  - **Dependencies**: None
  - **Estimated time**: 1-2 hours

- [ ] **Task 4.4**: Integrate tracking into launcher
  - Modify `internal/launcher/launcher.go`
  - Add `TrackLaunch()` call before launching Claude Code
  - Handle errors gracefully (don't fail launch if tracking fails)
  - **Dependencies**: Task 4.2
  - **Estimated time**: 30 minutes

- [ ] **Task 4.5**: Create `cmd/stats.go`
  - Implement `statsCmd` with all flags
  - Implement `runStats()` function
  - Parse date/time filters
  - Call tracker to get stats
  - **Dependencies**: Task 4.2, 4.3
  - **Estimated time**: 2-3 hours

- [ ] **Task 4.6**: Implement stats display formatting
  - Create formatted output for stats
  - Group by profile and model
  - Calculate percentages
  - Display estimated costs with disclaimer
  - Add "Most Active Days" section
  - **Dependencies**: Task 4.5
  - **Estimated time**: 1-2 hours

- [ ] **Task 4.7**: Implement date filtering
  - Support `--since`, `--until` date parsing
  - Support `--month`, `--today`, `--week` shortcuts
  - Validate date formats
  - **Dependencies**: Task 4.5
  - **Estimated time**: 1 hour

- [ ] **Task 4.8**: Implement CSV export
  - Add `--export` flag
  - Generate CSV with all launch data
  - Include headers: timestamp, profile, model, estimated_cost
  - **Dependencies**: Task 4.5
  - **Estimated time**: 1 hour

- [ ] **Task 4.9**: Add SQLite dependency
  - Update `go.mod` with `github.com/mattn/go-sqlite3`
  - Run `go mod tidy`
  - Test compilation
  - **Dependencies**: Task 4.1
  - **Estimated time**: 15 minutes

- [ ] **Task 4.10**: Register stats command
  - Add `statsCmd` to root in `cmd/root.go`
  - **Dependencies**: Task 4.5
  - **Estimated time**: 5 minutes

- [ ] **Task 4.11**: Testing
  - Test database creation and initialization
  - Test launch tracking (insert operations)
  - Test stats queries with various filters
  - Test date range filtering
  - Test CSV export
  - Test cost calculations
  - Test with missing pricing data (graceful handling)
  - **Dependencies**: All above tasks
  - **Estimated time**: 2-3 hours

- [ ] **Task 4.12**: Documentation
  - Update README.md with stats examples
  - Add PRICING.md with current pricing table
  - Document privacy (local storage only)
  - Add disclaimer about estimated vs. actual costs
  - **Dependencies**: Task 4.11
  - **Estimated time**: 1 hour

**Total estimated time for Feature 4**: 14-18 hours

---

## Implementation Order & Dependencies

### Phase 1: Multiple Named Profiles (Foundation)
All other features depend on this, especially profile-aware stats.
- **Tasks**: 1.1 → 1.2 → 1.3 → 1.4 → 1.5 → 1.6 → 1.7 → 1.8
- **Estimated time**: 10-13 hours

### Phase 2: Show Available Models Command
Independent of other features, can be done in parallel with Phase 3.
- **Tasks**: 2.1 → 2.2 → 2.3 → 2.4 → 2.5 → 2.6 → 2.7
- **Estimated time**: 6-8 hours
- **Dependencies**: Phase 1 (for profile support)

### Phase 3: Quick Model Switching
Can be done in parallel with Phase 2.
- **Tasks**: 3.1 → 3.2 → 3.3 → 3.4 → 3.5 → 3.6
- **Estimated time**: 4-5 hours
- **Dependencies**: Phase 1

### Phase 4: Cost Tracking & Usage Stats
Depends on Phase 1 for profile tracking.
- **Tasks**: 4.1 → 4.2 → 4.3 → 4.4 → 4.5 → 4.6 → 4.7 → 4.8 → 4.9 → 4.10 → 4.11 → 4.12
- **Estimated time**: 14-18 hours
- **Dependencies**: Phase 1

### Parallel Work Opportunities
- **Phase 2 + Phase 3** can be developed simultaneously after Phase 1
- **Phase 4** database layer (4.1, 4.3) can start while Phase 2/3 are in progress

---

## Total Estimated Time

| Feature | Estimated Time |
|---------|---------------|
| Feature 1: Multiple Named Profiles | 10-13 hours |
| Feature 2: Show Available Models | 6-8 hours |
| Feature 3: Quick Model Switching | 4-5 hours |
| Feature 4: Cost Tracking & Stats | 14-18 hours |
| **TOTAL** | **34-44 hours** |

With parallel development (Phases 2+3 in parallel after Phase 1):
- **Phase 1**: 10-13 hours
- **Phase 2+3 (parallel)**: 6-8 hours
- **Phase 4**: 14-18 hours
- **Optimized Total**: ~30-39 hours

---

## Testing Strategy

### Unit Tests
- Profile manager operations (CRUD)
- Database operations (inserts, queries)
- Cost calculations
- Model filtering and grouping

### Integration Tests
- Migration from old config.json
- Profile switching
- Launch with overrides
- End-to-end stats tracking

### Manual Testing
- AWS Bedrock connectivity
- Model availability across regions
- Cross-region routing
- Real Claude Code launches

---

## Documentation Updates

### Files to Update
1. **README.md**
   - Add profiles section
   - Add models command examples
   - Add override flags examples
   - Add stats command examples

2. **CONFIGURATION.md**
   - Add profile management section
   - Update configuration paths
   - Add migration guide

3. **New: PRICING.md**
   - Document pricing table
   - Explain cost estimation methodology
   - Add disclaimer

4. **TROUBLESHOOTING.md**
   - Add profile-related issues
   - Add database issues
   - Add model availability issues

---

## Migration & Backward Compatibility

### Version Bump
Recommend bumping to v0.3.0 (minor version bump for new features)

### Auto-migration Checklist
- [x] Detect old `~/.clauderock/config.json`
- [x] Create `~/.clauderock/profiles/` directory
- [x] Copy config to `profiles/default.json`
- [x] Set current profile to "default"
- [x] Rename old config to `config.json.bak`
- [x] Initialize empty `usage.db`

### Breaking Changes
None - all features are additive with backward compatibility via migration.

---

## Future Enhancements (Out of Scope)

These are nice-to-have features for later versions:
- Model performance benchmarking
- Cost budgets and alerts
- Shell completion (bash/zsh)
- Project-local `.clauderock.json` support
- Model aliases (e.g., `big` → `anthropic.claude-opus-4`)
- Configuration templates/presets
- Session management (status/stop/restart)
- Region latency benchmarking

---

## Notes

### Pricing Data Source
AWS Bedrock pricing: https://aws.amazon.com/bedrock/pricing/
Update `internal/pricing/calculator.go` when prices change.

### SQLite vs. Other Databases
SQLite chosen for:
- Zero configuration
- Serverless (no daemon)
- Single file storage
- Built-in Go support
- Fast for local usage tracking
- ACID transactions

### Privacy Considerations
All usage data stays local. Never transmitted. Users can:
- Delete `~/.clauderock/usage.db` anytime
- Query database directly with SQLite tools
- Export to CSV for external analysis
