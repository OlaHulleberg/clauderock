package usage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

type Session struct {
	ID                  int64
	StartTime           time.Time
	EndTime             time.Time
	DurationSeconds     int
	ProfileName         string
	WorkingDirectory    string
	Model               string
	SessionUUID         string
	TotalRequests       int
	TotalInputTokens    int64
	TotalOutputTokens   int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	AvgTPM              float64
	PeakTPM             float64
	P95TPM              float64
	AvgRPM              float64
	PeakRPM             float64
	P95RPM              float64
	CacheHitRate        float64
	ExitCode            int
}

func NewDatabase() (*Database, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dbPath := filepath.Join(home, ".clauderock", "usage.db")

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{db: db}

	if err := d.Init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return d, nil
}

func (d *Database) Init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		duration_seconds INTEGER NOT NULL,
		profile_name TEXT NOT NULL,
		working_directory TEXT,
		model TEXT NOT NULL,
		session_uuid TEXT,
		total_requests INTEGER DEFAULT 0,
		total_input_tokens INTEGER DEFAULT 0,
		total_output_tokens INTEGER DEFAULT 0,
		cache_read_tokens INTEGER DEFAULT 0,
		cache_creation_tokens INTEGER DEFAULT 0,
		avg_tpm REAL DEFAULT 0,
		peak_tpm REAL DEFAULT 0,
		p95_tpm REAL DEFAULT 0,
		avg_rpm REAL DEFAULT 0,
		peak_rpm REAL DEFAULT 0,
		p95_rpm REAL DEFAULT 0,
		cache_hit_rate REAL DEFAULT 0,
		exit_code INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_session_start_time ON sessions(start_time);
	CREATE INDEX IF NOT EXISTS idx_session_profile_name ON sessions(profile_name);
	CREATE INDEX IF NOT EXISTS idx_session_model ON sessions(model);
	CREATE INDEX IF NOT EXISTS idx_session_uuid ON sessions(session_uuid);
	`

	_, err := d.db.Exec(schema)
	return err
}

type QueryFilter struct {
	ProfileName string
	StartDate   time.Time
	EndDate     time.Time
	Model       string
}

func (d *Database) InsertSession(session Session) error {
	query := `
	INSERT INTO sessions (
		start_time, end_time, duration_seconds, profile_name, working_directory,
		model, session_uuid, total_requests, total_input_tokens, total_output_tokens,
		cache_read_tokens, cache_creation_tokens, avg_tpm, peak_tpm, p95_tpm,
		avg_rpm, peak_rpm, p95_rpm, cache_hit_rate, exit_code
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		session.StartTime,
		session.EndTime,
		session.DurationSeconds,
		session.ProfileName,
		session.WorkingDirectory,
		session.Model,
		session.SessionUUID,
		session.TotalRequests,
		session.TotalInputTokens,
		session.TotalOutputTokens,
		session.CacheReadTokens,
		session.CacheCreationTokens,
		session.AvgTPM,
		session.PeakTPM,
		session.P95TPM,
		session.AvgRPM,
		session.PeakRPM,
		session.P95RPM,
		session.CacheHitRate,
		session.ExitCode,
	)

	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	return nil
}

func (d *Database) QuerySessions(filter QueryFilter) ([]Session, error) {
	query := "SELECT id, start_time, end_time, duration_seconds, profile_name, working_directory, model, session_uuid, total_requests, total_input_tokens, total_output_tokens, cache_read_tokens, cache_creation_tokens, avg_tpm, peak_tpm, p95_tpm, avg_rpm, peak_rpm, p95_rpm, cache_hit_rate, exit_code FROM sessions WHERE 1=1"
	args := []interface{}{}

	if filter.ProfileName != "" {
		query += " AND profile_name = ?"
		args = append(args, filter.ProfileName)
	}

	if !filter.StartDate.IsZero() {
		query += " AND start_time >= ?"
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		query += " AND start_time <= ?"
		args = append(args, filter.EndDate)
	}

	if filter.Model != "" {
		query += " AND model = ?"
		args = append(args, filter.Model)
	}

	query += " ORDER BY start_time DESC"

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID,
			&s.StartTime,
			&s.EndTime,
			&s.DurationSeconds,
			&s.ProfileName,
			&s.WorkingDirectory,
			&s.Model,
			&s.SessionUUID,
			&s.TotalRequests,
			&s.TotalInputTokens,
			&s.TotalOutputTokens,
			&s.CacheReadTokens,
			&s.CacheCreationTokens,
			&s.AvgTPM,
			&s.PeakTPM,
			&s.P95TPM,
			&s.AvgRPM,
			&s.PeakRPM,
			&s.P95RPM,
			&s.CacheHitRate,
			&s.ExitCode,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

// CountSessions returns the total number of sessions in the database
func (d *Database) CountSessions() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}
	return count, nil
}

// ClearSessions deletes all session records from the database
func (d *Database) ClearSessions() error {
	_, err := d.db.Exec("DELETE FROM sessions")
	if err != nil {
		return fmt.Errorf("failed to clear sessions: %w", err)
	}
	return nil
}
