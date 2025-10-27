package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/OlaHulleberg/clauderock/internal/pricing"
	"github.com/OlaHulleberg/clauderock/internal/usage"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

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

// Styles for stats output
var (
	headerStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	sectionStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	labelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	valueStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	highlightStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	costStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	mutedStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boxStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("8")).Padding(0, 1)
	separatorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// formatNumber formats an integer with thousand separators
func formatNumber(n int64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", n)
}

// formatFloat formats a float with thousand separators
func formatFloat(f float64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%.0f", f)
}

// progressBar creates a visual progress bar for percentages
func progressBar(percentage float64, width int) string {
	filled := int(percentage / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return mutedStyle.Render("[") + bar + mutedStyle.Render("]")
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "View usage statistics and estimated costs",
	Long: `View usage statistics and estimated costs.

Shows session counts, token usage, TPM/RPM metrics, and cost estimates
based on actual token usage from tracked sessions.

Examples:
  clauderock stats
  clauderock stats --profile work-dev
  clauderock stats --model anthropic.claude-sonnet-4-5
  clauderock stats --since 2025-10-01
  clauderock stats --month 2025-10
  clauderock stats --today
  clauderock stats --export report.csv`,
	RunE: runStats,
}

func init() {
	// Registered by manage.go

	statsCmd.Flags().StringVar(&statsProfile, "profile", "", "Filter by profile name")
	statsCmd.Flags().StringVar(&statsModel, "model", "", "Filter by model")
	statsCmd.Flags().StringVar(&statsSince, "since", "", "Filter sessions since date (YYYY-MM-DD)")
	statsCmd.Flags().StringVar(&statsUntil, "until", "", "Filter sessions until date (YYYY-MM-DD)")
	statsCmd.Flags().StringVar(&statsMonth, "month", "", "Filter by month (YYYY-MM)")
	statsCmd.Flags().BoolVar(&statsToday, "today", false, "Show today's stats only")
	statsCmd.Flags().BoolVar(&statsWeek, "week", false, "Show this week's stats")
	statsCmd.Flags().BoolVar(&statsDetailed, "detailed", false, "Show detailed output")
	statsCmd.Flags().StringVar(&statsExport, "export", "", "Export to CSV file")
}

func runStats(cmd *cobra.Command, args []string) error {
	tracker, err := usage.NewTracker()
	if err != nil {
		return fmt.Errorf("failed to create tracker: %w", err)
	}
	defer tracker.Close()

	// Build filter
	filter := usage.QueryFilter{
		ProfileName: statsProfile,
		Model:       statsModel,
	}

	// Parse date filters
	if statsToday {
		now := time.Now()
		filter.StartDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		filter.EndDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	} else if statsWeek {
		now := time.Now()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		startOfWeek := now.AddDate(0, 0, -(weekday - 1))
		filter.StartDate = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())
		filter.EndDate = now
	} else if statsMonth != "" {
		monthDate, err := time.Parse("2006-01", statsMonth)
		if err != nil {
			return fmt.Errorf("invalid month format, use YYYY-MM: %w", err)
		}
		filter.StartDate = time.Date(monthDate.Year(), monthDate.Month(), 1, 0, 0, 0, 0, monthDate.Location())
		filter.EndDate = filter.StartDate.AddDate(0, 1, 0).Add(-time.Second)
	} else {
		if statsSince != "" {
			sinceDate, err := time.Parse("2006-01-02", statsSince)
			if err != nil {
				return fmt.Errorf("invalid since date format, use YYYY-MM-DD: %w", err)
			}
			filter.StartDate = sinceDate
		}
		if statsUntil != "" {
			untilDate, err := time.Parse("2006-01-02", statsUntil)
			if err != nil {
				return fmt.Errorf("invalid until date format, use YYYY-MM-DD: %w", err)
			}
			filter.EndDate = untilDate
		}
	}

	// Get session stats (new detailed view)
	sessionStats, err := tracker.GetSessionStats(filter)
	if err != nil {
		return fmt.Errorf("failed to get session stats: %w", err)
	}

	// Export to CSV if requested
	if statsExport != "" {
		if err := exportSessionsToCSV(tracker, filter, statsExport); err != nil {
			return fmt.Errorf("failed to export: %w", err)
		}
		fmt.Printf("Exported to %s\n", statsExport)
		return nil
	}

	// Display session stats
	displaySessionStats(sessionStats, filter)

	return nil
}

func displaySessionStats(stats *usage.SessionStats, filter usage.QueryFilter) {
	// Determine time period for header
	timePeriod := "All Time"
	if !filter.StartDate.IsZero() || !filter.EndDate.IsZero() {
		if !filter.StartDate.IsZero() && !filter.EndDate.IsZero() {
			timePeriod = fmt.Sprintf("%s to %s",
				filter.StartDate.Format("2006-01-02"),
				filter.EndDate.Format("2006-01-02"))
		} else if !filter.StartDate.IsZero() {
			timePeriod = fmt.Sprintf("Since %s", filter.StartDate.Format("2006-01-02"))
		} else {
			timePeriod = fmt.Sprintf("Until %s", filter.EndDate.Format("2006-01-02"))
		}
	}

	// Header
	fmt.Println(headerStyle.Render("📊 Session Statistics") + " " + mutedStyle.Render("("+timePeriod+")"))
	fmt.Println()

	if stats.TotalSessions == 0 {
		fmt.Println(mutedStyle.Render("No sessions found matching the criteria."))
		return
	}

	// Overall session metrics
	overallContent := fmt.Sprintf(
		"%s %s\n%s %.2f hours\n%s %.1f minutes",
		labelStyle.Render("Total Sessions:"),
		valueStyle.Render(formatNumber(int64(stats.TotalSessions))),
		labelStyle.Render("Total Coding Time:"),
		stats.TotalDurationHours,
		labelStyle.Render("Average Session:"),
		stats.AvgSessionMinutes,
	)
	fmt.Println(boxStyle.Render(overallContent))
	fmt.Println()

	// Token metrics
	fmt.Println(sectionStyle.Render("▸ Token Usage"))
	fmt.Println()
	fmt.Printf("  %s %s\n", labelStyle.Render("Total Requests:"), valueStyle.Render(formatNumber(stats.TotalRequests)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Input Tokens:"), valueStyle.Render(formatNumber(stats.TotalInputTokens)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Output Tokens:"), valueStyle.Render(formatNumber(stats.TotalOutputTokens)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Total Tokens:"), highlightStyle.Render(formatNumber(stats.TotalInputTokens+stats.TotalOutputTokens)))
	fmt.Println()

	// TPM/RPM metrics
	fmt.Println(sectionStyle.Render("▸ Tokens Per Minute (TPM)"))
	fmt.Println()
	fmt.Printf("  %s %s\n", labelStyle.Render("Average:"), valueStyle.Render(formatFloat(stats.AvgTPM)+" TPM"))
	fmt.Printf("  %s %s\n", labelStyle.Render("Peak:"), highlightStyle.Render(formatFloat(stats.PeakTPM)+" TPM"))
	fmt.Printf("  %s %s\n", labelStyle.Render("P95:"), valueStyle.Render(formatFloat(stats.P95TPM)+" TPM"))
	fmt.Println()

	fmt.Println(sectionStyle.Render("▸ Requests Per Minute (RPM)"))
	fmt.Println()
	fmt.Printf("  %s %s\n", labelStyle.Render("Average:"), valueStyle.Render(fmt.Sprintf("%.1f RPM", stats.AvgRPM)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Peak:"), highlightStyle.Render(fmt.Sprintf("%.1f RPM", stats.PeakRPM)))
	fmt.Printf("  %s %s\n", labelStyle.Render("P95:"), valueStyle.Render(fmt.Sprintf("%.1f RPM", stats.P95RPM)))
	fmt.Println()

	// Cache efficiency
	fmt.Println(sectionStyle.Render("▸ Cache Efficiency"))
	fmt.Println()
	cacheColor := highlightStyle
	if stats.AvgCacheHitRate < 30 {
		cacheColor = mutedStyle
	}
	fmt.Printf("  %s %s\n", labelStyle.Render("Average Hit Rate:"), cacheColor.Render(fmt.Sprintf("%.1f%%", stats.AvgCacheHitRate)))
	fmt.Println()

	// Display by profile
	if len(stats.ProfileBreakdown) > 0 && filter.ProfileName == "" {
		fmt.Println(sectionStyle.Render("▸ By Profile"))
		fmt.Println()
		displayBreakdown(stats.ProfileBreakdown, stats.TotalSessions)
		fmt.Println()
	}

	// Display by model
	if len(stats.ModelBreakdown) > 0 {
		fmt.Println(sectionStyle.Render("▸ By Model"))
		fmt.Println()
		displayBreakdown(stats.ModelBreakdown, stats.TotalSessions)
		fmt.Println()
	}

	// Display top sessions
	if len(stats.TopSessions) > 0 {
		fmt.Println(sectionStyle.Render("▸ Top Sessions by Activity"))
		fmt.Println()
		for i, session := range stats.TopSessions {
			fmt.Printf("  %s %s - %s avg TPM, %s min %s\n",
				mutedStyle.Render(fmt.Sprintf("%d.", i+1)),
				valueStyle.Render(session.StartTime.Format("Jan 02 15:04")),
				highlightStyle.Render(formatFloat(session.AvgTPM)),
				valueStyle.Render(fmt.Sprintf("%d", session.DurationSeconds/60)),
				mutedStyle.Render("("+session.Model+")"))
		}
		fmt.Println()
	}

	// Display estimated costs
	fmt.Println(sectionStyle.Render("▸ Estimated Costs"))
	fmt.Println(mutedStyle.Render("  Based on actual token usage"))
	fmt.Println()

	totalCost := 0.0
	for model, count := range stats.ModelBreakdown {
		// Calculate average tokens per session for this model
		var modelInputTokens, modelOutputTokens int64
		var modelSessions int

		db, err := usage.NewDatabase()
		if err == nil {
			defer db.Close()
			modelFilter := filter
			modelFilter.Model = model
			sessions, err := db.QuerySessions(modelFilter)
			if err == nil {
				for _, s := range sessions {
					modelInputTokens += s.TotalInputTokens
					modelOutputTokens += s.TotalOutputTokens
					modelSessions++
				}
			}
		}

		if modelSessions > 0 {
			cost := pricing.CalculateCost(model, modelInputTokens, modelOutputTokens)
			totalCost += cost
			fmt.Printf("  %s %s %s\n",
				labelStyle.Render(model+":"),
				costStyle.Render(fmt.Sprintf("$%.2f", cost)),
				mutedStyle.Render(fmt.Sprintf("(%d sessions)", count)))
		}
	}

	if totalCost > 0 {
		fmt.Println()
		fmt.Printf("  %s %s\n",
			labelStyle.Render("Total Estimated Cost:"),
			costStyle.Render(fmt.Sprintf("$%.2f", totalCost)))
	}
}

func displayBreakdown(breakdown map[string]int, total int) {
	// Sort by count descending
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range breakdown {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	for _, item := range sorted {
		percentage := float64(item.Value) / float64(total) * 100
		bar := progressBar(percentage, 20)
		fmt.Printf("  %s %s %s %s\n",
			valueStyle.Render(item.Key+":"),
			bar,
			highlightStyle.Render(fmt.Sprintf("%.1f%%", percentage)),
			mutedStyle.Render(fmt.Sprintf("(%d sessions)", item.Value)))
	}
}

func exportSessionsToCSV(tracker *usage.Tracker, filter usage.QueryFilter, filename string) error {
	// Get raw sessions
	db, err := usage.NewDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	sessions, err := db.QuerySessions(filter)
	if err != nil {
		return err
	}

	// Create CSV file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Start Time",
		"Duration (min)",
		"Profile Name",
		"Model",
		"Requests",
		"Input Tokens",
		"Output Tokens",
		"Avg TPM",
		"Peak TPM",
		"P95 TPM",
		"Avg RPM",
		"Peak RPM",
		"P95 RPM",
		"Cache Hit Rate %",
		"Estimated Cost",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data
	for _, session := range sessions {
		cost := pricing.CalculateCost(session.Model, session.TotalInputTokens, session.TotalOutputTokens)
		row := []string{
			session.StartTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%d", session.DurationSeconds/60),
			session.ProfileName,
			session.Model,
			fmt.Sprintf("%d", session.TotalRequests),
			fmt.Sprintf("%d", session.TotalInputTokens),
			fmt.Sprintf("%d", session.TotalOutputTokens),
			fmt.Sprintf("%.0f", session.AvgTPM),
			fmt.Sprintf("%.0f", session.PeakTPM),
			fmt.Sprintf("%.0f", session.P95TPM),
			fmt.Sprintf("%.1f", session.AvgRPM),
			fmt.Sprintf("%.1f", session.PeakRPM),
			fmt.Sprintf("%.1f", session.P95RPM),
			fmt.Sprintf("%.1f", session.CacheHitRate),
			fmt.Sprintf("%.2f", cost),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
