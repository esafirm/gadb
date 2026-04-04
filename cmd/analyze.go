// Copyright © 2019 Esa Firman esafirm21@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	adb "github.com/esafirm/gadb/adb"
	"github.com/esafirm/gadb/ai"
	color "github.com/fatih/color"

	"github.com/spf13/cobra"
)

var (
	crashesOnly bool
	recentOnly  bool
	timeRange   string
	useAI       bool
	aiProvider  string
	verbose     bool
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze device logs with AI insights",
	Long:  `Analyze logcat output to detect crashes, anomalies, and provide actionable insights.`,
	Run: func(cmd *cobra.Command, args []string) {
		analyzeLogs()
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().BoolVarP(&crashesOnly, "crashes", "c", true, "Focus on crash logs only")
	analyzeCmd.Flags().BoolVarP(&recentOnly, "recent", "r", false, "Only analyze recent logs")
	analyzeCmd.Flags().StringVarP(&timeRange, "time", "t", "", "Time range for logs (e.g., '5m', '1h')")
	analyzeCmd.Flags().BoolVarP(&useAI, "ai", "a", false, "Use AI for analysis (requires API key or OAuth)")
	analyzeCmd.Flags().StringVar(&aiProvider, "provider", "gemini", "AI provider: gemini, anthropic, or openai")
	analyzeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
}

func analyzeLogs() {
	color.Cyan("Analyzing device logs...")

	if verbose {
		color.HiBlack("Crashes only: %v", crashesOnly)
		color.HiBlack("Recent only: %v", recentOnly)
		color.HiBlack("Time range: %s", timeRange)
		color.HiBlack("Use AI: %v", useAI)
	}

	// Get logcat output
	logs := captureLogs()

	if crashesOnly {
		analyzeCrashes(logs)
	} else {
		analyzeAllLogs(logs)
	}
}

func captureLogs() string {
	var logcatArgs []string

	// Build logcat command arguments
	logcatArgs = append(logcatArgs, "logcat", "-d") // -d for dump current buffer

	if recentOnly {
		logcatArgs = append(logcatArgs, "-t", "500") // Last 500 lines
	}

	if timeRange != "" {
		logcatArgs = append(logcatArgs, "-t", parseTimeRange(timeRange))
	}

	if verbose {
		color.HiBlack("Running: adb %s", strings.Join(logcatArgs, " "))
	}

	// Execute logcat command with timeout
	result := adb.RunOnlyWithTimeout(5*time.Second, "adb", logcatArgs...)

	output := string(result.Output)
	if strings.Contains(output, "- waiting for device -") {
		color.Red("No device connected. Please connect a device and try again.")
		os.Exit(1)
	}

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "context deadline exceeded") {
			color.Red("Command timed out. This often happens when no device is connected or ADB is hanging.")
			os.Exit(1)
		}
		color.Red("Failed to capture logs: %v", result.Error)
		os.Exit(1)
	}

	return output
}

func parseTimeRange(timeRange string) string {
	// Parse time range like "5m", "1h", "30s"
	// For now, we'll convert to approximate line count
	// 5m ≈ 5000 lines, 1h ≈ 60000 lines, etc.

	if strings.HasSuffix(timeRange, "s") {
		return "100" // ~100 lines per second
	} else if strings.HasSuffix(timeRange, "m") {
		return "5000" // ~5000 lines per minute
	} else if strings.HasSuffix(timeRange, "h") {
		return "300000" // ~300k lines per hour
	}

	// Extract the number part
	var numStr string
	for i, char := range timeRange {
		if char < '0' || char > '9' {
			numStr = timeRange[:i]
			break
		}
	}

	return numStr
}

func analyzeCrashes(logs string) {
	if verbose {
		color.HiBlack("Extracting crashes from logs (length: %d)...", len(logs))
	}

	crashes := extractCrashes(logs)

	if len(crashes) == 0 {
		color.Green("✓ No crashes found in the logs!")
		return
	}

	color.Yellow("⚠ Found %d crash(es):\n", len(crashes))

	for i, crash := range crashes {
		fmt.Println()
		color.Red("%d. %s", i+1, crash.Summary)

		// Display crash details
		if !crash.Timestamp.IsZero() {
			color.HiBlack("   Time: %s", crash.Timestamp.Format("2006-01-02 15:04:05"))
		}

		if crash.ProcessID != "" {
			color.HiBlack("   PID: %s", crash.ProcessID)
		}

		if crash.ThreadID != "" {
			color.HiBlack("   TID: %s", crash.ThreadID)
		}

		// Display stack trace
		fmt.Println()
		color.HiBlack("   Stack Trace:")
		stackLines := strings.Split(crash.StackTrace, "\n")
		for _, line := range stackLines {
			if line != "" {
				fmt.Printf("   %s\n", color.HiBlackString(line))
			}
		}

		// AI Analysis if enabled
		if useAI {
			aiAnalysis := analyzeWithAI(crash)
			fmt.Println()
			color.Cyan("   🤖 AI Analysis:")
			fmt.Printf("   %s\n", aiAnalysis)
		}

		// Add separator between crashes
		if i < len(crashes)-1 {
			fmt.Println()
			color.HiBlack("   " + strings.Repeat("-", 60))
		}
	}

	fmt.Println()
	color.Yellow("💡 Run with --ai flag to get AI-powered crash analysis")
	color.HiBlack("   Supported providers: gemini (OAuth), anthropic, openai (API key required)")
}

func analyzeAllLogs(logs string) {
	// For full log analysis, we'll need more sophisticated logic
	color.Yellow("Full log analysis not yet implemented. Use --crashes flag for crash analysis.")
}

// Crash represents a detected crash in the logs
type Crash struct {
	Summary    string
	StackTrace string
	Timestamp  time.Time
	ProcessID  string
	ThreadID   string
}

func extractCrashes(logs string) []Crash {
	var crashes []Crash
	lines := strings.Split(logs, "\n")

	var currentCrash *Crash
	var collectingStackTrace bool

	crashPatterns := []string{
		"FATAL EXCEPTION",
		"AndroidRuntime: FATAL EXCEPTION",
		"Process: com.android.internal.os.ProcessZygote",
		"ANR in",
		"Application Not Responding",
		"SIGSEGV",
		"SIGABRT",
		"SIGBUS",
		"SIGILL",
		"DEBUG: Crash",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect crashes using patterns
		isCrashStart := false
		for _, pattern := range crashPatterns {
			if strings.Contains(line, pattern) {
				isCrashStart = true
				break
			}
		}

		if isCrashStart {
			// Save previous crash if exists
			if currentCrash != nil && currentCrash.StackTrace != "" {
				crashes = append(crashes, *currentCrash)
			}

			currentCrash = &Crash{
				Summary:    line,
				StackTrace: "",
				Timestamp:  extractTimestamp(line),
			}
			collectingStackTrace = true
			continue
		}

		// Extract process and thread info
		if currentCrash != nil && (strings.Contains(line, "PID:") || strings.Contains(line, "Process:")) {
			extractProcessInfo(line, currentCrash)
			continue
		}

		// Collect stack trace for current crash
		if collectingStackTrace && currentCrash != nil {
			// Stack trace lines typically start with "at " or contain "Caused by:"
			if strings.HasPrefix(line, "at ") || strings.Contains(line, "Caused by:") {
				currentCrash.StackTrace += line + "\n"
			} else if line == "" || strings.HasPrefix(line, "-----") || strings.Contains(line, "DEBUG") {
				// End of stack trace
				if currentCrash.StackTrace != "" {
					crashes = append(crashes, *currentCrash)
					currentCrash = nil
				}
				collectingStackTrace = false
			}
		}
	}

	// Add the last crash if we were collecting one
	if currentCrash != nil && currentCrash.StackTrace != "" {
		crashes = append(crashes, *currentCrash)
	}

	return crashes
}

func extractTimestamp(line string) time.Time {
	// Try to extract timestamp from log line (format: MM-DD HH:MM:SS.mmm)
	// This is a simplified version - you may want to enhance this
	if len(line) >= 18 {
		timestampStr := line[:18]
		if timestamp, err := time.Parse("01-02 15:04:05.000", timestampStr); err == nil {
			return timestamp
		}
	}
	return time.Now()
}

func extractProcessInfo(line string, crash *Crash) {
	// Extract PID from lines like "PID: 12345"
	if strings.Contains(line, "PID:") {
		parts := strings.Split(line, "PID:")
		if len(parts) > 1 {
			pid := strings.TrimSpace(parts[1])
			if spaceIdx := strings.Index(pid, " "); spaceIdx != -1 {
				pid = pid[:spaceIdx]
			}
			crash.ProcessID = pid
		}
	}

	// Extract process name from lines like "Process: com.example.app"
	if strings.Contains(line, "Process:") {
		parts := strings.Split(line, "Process:")
		if len(parts) > 1 {
			processName := strings.TrimSpace(parts[1])
			if spaceIdx := strings.Index(processName, " "); spaceIdx != -1 {
				processName = processName[:spaceIdx]
			}
			// If we haven't set process ID yet, try to get it from this line
			if crash.ProcessID == "" && strings.Contains(processName, "(") {
				parenIdx := strings.Index(processName, "(")
				if parenIdx > 0 {
					crash.ProcessID = strings.TrimSpace(processName[parenIdx+1 : len(processName)-1])
				}
			}
		}
	}
}

func analyzeWithAI(crash Crash) string {
	color.HiBlack("   Requesting AI analysis...")

	request := ai.AnalysisRequest{
		CrashSummary: crash.Summary,
		StackTrace:   crash.StackTrace,
		ProcessID:    crash.ProcessID,
		ThreadID:     crash.ThreadID,
	}

	response, err := ai.AnalyzeCrash(request)
	if err != nil {
		return fmt.Sprintf("❌ AI analysis failed: %v\n   Make sure your AI is configured properly with 'gadb config --ai'", err)
	}

	return ai.FormatAnalysisResponse(response)
}
