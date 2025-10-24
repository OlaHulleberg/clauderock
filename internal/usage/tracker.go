package usage

import (
	"fmt"
	"time"

	"github.com/OlaHulleberg/clauderock/internal/monitoring"
)

type Tracker struct {
	db *Database
}

func NewTracker() (*Tracker, error) {
	db, err := NewDatabase()
	if err != nil {
		return nil, err
	}

	return &Tracker{db: db}, nil
}

type SessionInfo struct {
	StartTime          time.Time
	EndTime            time.Time
	ProfileName        string
	WorkingDirectory   string
	AWSProfile         string
	Region             string
	CrossRegion        string
	Model              string
	ModelProfileID     string
	FastModel          string
	FastModelProfileID string
	ExitCode           int
}

func (t *Tracker) TrackSession(info SessionInfo) error {
	// Try to find and parse the JSONL file
	var metrics *monitoring.SessionMetrics
	if info.WorkingDirectory != "" {
		jsonlPath, err := monitoring.FindSessionJSONL(info.WorkingDirectory, info.StartTime)
		if err == nil {
			metrics, err = monitoring.ParseSessionJSONL(jsonlPath)
			if err != nil {
				// Log error but don't fail - we can still track basic session info
				fmt.Printf("Warning: failed to parse session JSONL: %v\n", err)
			}
		} else {
			fmt.Printf("Warning: failed to find session JSONL: %v\n", err)
		}
	}

	// Build session record
	session := Session{
		StartTime:        info.StartTime,
		EndTime:          info.EndTime,
		DurationSeconds:  int(info.EndTime.Sub(info.StartTime).Seconds()),
		ProfileName:      info.ProfileName,
		WorkingDirectory: info.WorkingDirectory,
		Model:            info.Model,
		ExitCode:         info.ExitCode,
	}

	// Add metrics if we successfully parsed the JSONL
	if metrics != nil {
		session.SessionUUID = metrics.SessionUUID
		session.TotalRequests = metrics.TotalRequests
		session.TotalInputTokens = metrics.TotalInputTokens
		session.TotalOutputTokens = metrics.TotalOutputTokens
		session.CacheReadTokens = metrics.CacheReadTokens
		session.CacheCreationTokens = metrics.CacheCreationTokens
		session.AvgTPM = metrics.AvgTPM
		session.PeakTPM = metrics.PeakTPM
		session.P95TPM = metrics.P95TPM
		session.AvgRPM = metrics.AvgRPM
		session.PeakRPM = metrics.PeakRPM
		session.P95RPM = metrics.P95RPM
		session.CacheHitRate = metrics.CacheHitRate
	}

	return t.db.InsertSession(session)
}

type SessionStats struct {
	TotalSessions       int
	TotalDurationHours  float64
	AvgSessionMinutes   float64
	TotalRequests       int64
	TotalInputTokens    int64
	TotalOutputTokens   int64
	AvgTPM              float64
	PeakTPM             float64
	P95TPM              float64
	AvgRPM              float64
	PeakRPM             float64
	P95RPM              float64
	AvgCacheHitRate     float64
	ModelBreakdown      map[string]int
	ProfileBreakdown    map[string]int
	TopSessions         []Session
}

func (t *Tracker) GetSessionStats(filter QueryFilter) (*SessionStats, error) {
	sessions, err := t.db.QuerySessions(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}

	stats := &SessionStats{
		TotalSessions:    len(sessions),
		ModelBreakdown:   make(map[string]int),
		ProfileBreakdown: make(map[string]int),
		TopSessions:      []Session{},
	}

	if len(sessions) == 0 {
		return stats, nil
	}

	var totalDurationSeconds int64
	var totalCacheHitRate float64
	var allTPMs []float64
	var allRPMs []float64

	for _, session := range sessions {
		totalDurationSeconds += int64(session.DurationSeconds)
		stats.TotalRequests += int64(session.TotalRequests)
		stats.TotalInputTokens += session.TotalInputTokens
		stats.TotalOutputTokens += session.TotalOutputTokens
		totalCacheHitRate += session.CacheHitRate

		stats.ModelBreakdown[session.Model]++
		stats.ProfileBreakdown[session.ProfileName]++

		// Collect TPM and RPM values for aggregation
		if session.AvgTPM > 0 {
			allTPMs = append(allTPMs, session.AvgTPM)
		}
		if session.AvgRPM > 0 {
			allRPMs = append(allRPMs, session.AvgRPM)
		}

		// Track peak values
		if session.PeakTPM > stats.PeakTPM {
			stats.PeakTPM = session.PeakTPM
		}
		if session.PeakRPM > stats.PeakRPM {
			stats.PeakRPM = session.PeakRPM
		}
	}

	// Calculate averages
	stats.TotalDurationHours = float64(totalDurationSeconds) / 3600.0
	stats.AvgSessionMinutes = float64(totalDurationSeconds) / float64(len(sessions)) / 60.0
	stats.AvgCacheHitRate = totalCacheHitRate / float64(len(sessions))

	// Calculate average TPM/RPM
	if len(allTPMs) > 0 {
		var sum float64
		for _, tpm := range allTPMs {
			sum += tpm
		}
		stats.AvgTPM = sum / float64(len(allTPMs))
	}

	if len(allRPMs) > 0 {
		var sum float64
		for _, rpm := range allRPMs {
			sum += rpm
		}
		stats.AvgRPM = sum / float64(len(allRPMs))
	}

	// Calculate P95 from all sessions
	if len(sessions) > 0 {
		var allP95TPMs []float64
		var allP95RPMs []float64
		for _, session := range sessions {
			if session.P95TPM > 0 {
				allP95TPMs = append(allP95TPMs, session.P95TPM)
			}
			if session.P95RPM > 0 {
				allP95RPMs = append(allP95RPMs, session.P95RPM)
			}
		}

		if len(allP95TPMs) > 0 {
			var sum float64
			for _, p95 := range allP95TPMs {
				sum += p95
			}
			stats.P95TPM = sum / float64(len(allP95TPMs))
		}

		if len(allP95RPMs) > 0 {
			var sum float64
			for _, p95 := range allP95RPMs {
				sum += p95
			}
			stats.P95RPM = sum / float64(len(allP95RPMs))
		}
	}

	// Get top 5 sessions by TPM
	if len(sessions) >= 5 {
		stats.TopSessions = sessions[:5]
	} else {
		stats.TopSessions = sessions
	}

	return stats, nil
}

func (t *Tracker) Close() error {
	return t.db.Close()
}
