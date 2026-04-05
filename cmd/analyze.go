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
	"github.com/esafirm/gadb/config"
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
	perfOnly    bool
	startupOnly bool
	packageName string
)

// analyzeCmd represents the analyze command
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze device logs with AI insights",
	Long:  `Analyze logcat output to detect crashes, anomalies, performance issues, and provide actionable insights.`,
	Run: func(cmd *cobra.Command, args []string) {
		analyzeLogs()
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)

	analyzeCmd.Flags().BoolVarP(&crashesOnly, "crashes", "c", false, "Focus on crash logs only")
	analyzeCmd.Flags().BoolVarP(&perfOnly, "performance", "p", false, "Analyze performance issues")
	analyzeCmd.Flags().BoolVarP(&startupOnly, "startup", "s", false, "Analyze app startup performance")
	analyzeCmd.Flags().BoolVarP(&recentOnly, "recent", "r", false, "Only analyze recent logs")
	analyzeCmd.Flags().StringVarP(&timeRange, "time", "t", "", "Time range for logs (e.g., '5m', '1h')")
	analyzeCmd.Flags().BoolVarP(&useAI, "ai", "a", false, "Use AI for analysis (requires API key or OAuth)")
	analyzeCmd.Flags().StringVar(&aiProvider, "provider", "gemini", "AI provider: gemini, anthropic, or openai")
	analyzeCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	analyzeCmd.Flags().StringVarP(&packageName, "package", "k", "", "Filter logs by package name (e.g., 'com.example.app')")
}

func analyzeLogs() {
	color.Cyan("Analyzing device logs...")

	// Get package name from flag or config
	if packageName == "" {
		packageName = config.GetPackageNameOrDefault(func() string {
			return ""
		})
	}

	// Default to crashes if no specific analysis requested
	if !crashesOnly && !perfOnly && !startupOnly {
		crashesOnly = true
	}

	if verbose {
		color.HiBlack("Crashes only: %v", crashesOnly)
		color.HiBlack("Performance only: %v", perfOnly)
		color.HiBlack("Startup only: %v", startupOnly)
		color.HiBlack("Recent only: %v", recentOnly)
		color.HiBlack("Time range: %s", timeRange)
		color.HiBlack("Use AI: %v", useAI)
		color.HiBlack("Package: %s", packageName)
	}

	// Get logcat output
	logs := captureLogs()

	if crashesOnly {
		analyzeCrashes(logs)
	}

	if perfOnly || startupOnly {
		analyzePerformance(logs, startupOnly)
	}

	if !crashesOnly && !perfOnly && !startupOnly {
		analyzeAllLogs(logs)
	}
}

func captureLogs() string {
	var logcatArgs []string

	// Build logcat command arguments
	logcatArgs = append(logcatArgs, "logcat", "-d") // -d for dump current buffer

	var pid string
	if packageName != "" {
		pid = getPidForPackage(packageName)
		if pid != "" && verbose {
			color.HiBlack("Found PID %s for package %s", pid, packageName)
		}
	}

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

func getPidForPackage(pkg string) string {
	result := adb.RunOnly("adb", "shell", "pidof", pkg)
	if result.Error != nil {
		return ""
	}
	return strings.TrimSpace(string(result.Output))
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
	color.Yellow("Full log analysis not yet implemented. Use --crashes or --performance flags.")
}

func analyzePerformance(logs string, isStartup bool) {
	if verbose {
		color.HiBlack("Extracting performance info from logs (length: %d)...", len(logs))
	}

	perfLogs := extractPerformanceLogs(logs, isStartup)

	if perfLogs == "" {
		color.Green("✓ No obvious performance issues found in the logs!")
		return
	}

	if isStartup {
		color.Yellow("🚀 App Startup Performance Analysis:")
	} else {
		color.Yellow("⚡ Performance Analysis:")
	}

	if useAI {
		color.HiBlack("   Requesting AI analysis...")
		request := ai.PerformanceRequest{
			Logs:      perfLogs,
			IsStartup: isStartup,
		}

		response, err := ai.AnalyzePerformance(request)
		if err != nil {
			color.Red("   ❌ AI analysis failed: %v", err)
			fmt.Println()
			color.HiBlack("   Relevant logs found:")
			fmt.Println(perfLogs)
			return
		}

		fmt.Println()
		color.Cyan("   🤖 AI Analysis:")
		fmt.Printf("   %s\n", ai.FormatAnalysisResponse(response))
	} else {
		fmt.Println()
		color.HiBlack("   Relevant logs found:")
		fmt.Println(perfLogs)
		fmt.Println()
		color.Yellow("💡 Run with --ai flag to get AI-powered performance analysis")
	}
}

func extractPerformanceLogs(logs string, isStartup bool) string {
	var relevantLines []string
	lines := strings.Split(logs, "\n")

	perfPatterns := []string{
		"Displayed",      // Activity launch time
		"Slow operation", // General slow operations
		"The application may be doing too much work", // Choreographer/Main thread
		"GC_",                              // Dalvik/ART GC
		"GC ",                              // General GC
		"freed",                            // Memory freed
		"ANR in",                           // ANRs
		"wait for the debugger",            // Debugger wait
		"Taking too long",                  // General timeout
		"long time",                        // General slowness
		"Waited for",                       // Wait time
		"Skipped",                          // Choreographer skipped frames
		"Background concurrent copying GC", // Heavy GC
	}

	startupPatterns := []string{
		"ActivityManager: Start proc",
		"ActivityManager: Displayed",
		"ActivityThread: bindApplication",
		"ActivityThread: installProvider",
		"Zygote: Process",
	}

	patterns := perfPatterns
	if isStartup {
		patterns = startupPatterns
	}

	// If we have a package name, try to get its PID to broaden filtering
	pid := ""
	if packageName != "" {
		pid = getPidForPackage(packageName)
	}

	for _, line := range lines {
		// If filtering by package, line must contain package name or PID
		if packageName != "" {
			if !strings.Contains(line, packageName) && (pid == "" || !strings.Contains(line, pid)) {
				// Special case: "Displayed" lines often contain the component name but might not match full package
				// "ActivityManager: Displayed com.example/.MainActivity: +345ms"
				if !strings.Contains(line, "Displayed") {
					continue
				}
			}
		}

		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				relevantLines = append(relevantLines, line)
				break
			}
		}
	}

	// Limit to last 50 relevant lines to avoid overwhelming the AI
	if len(relevantLines) > 50 {
		relevantLines = relevantLines[len(relevantLines)-50:]
	}

	return strings.Join(relevantLines, "\n")
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
			// Save previous crash if exists and matches package filter
			if currentCrash != nil && currentCrash.StackTrace != "" {
				if shouldIncludeCrash(currentCrash, packageName) {
					crashes = append(crashes, *currentCrash)
				}
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
					if shouldIncludeCrash(currentCrash, packageName) {
						crashes = append(crashes, *currentCrash)
					}
					currentCrash = nil
				}
				collectingStackTrace = false
			}
		}
	}

	// Add the last crash if we were collecting one and it matches the package filter
	if currentCrash != nil && currentCrash.StackTrace != "" {
		if shouldIncludeCrash(currentCrash, packageName) {
			crashes = append(crashes, *currentCrash)
		}
	}

	return crashes
}

func shouldIncludeCrash(crash *Crash, pkg string) bool {
	if pkg == "" {
		return true
	}
	// Check if package name is mentioned in summary or stack trace
	return strings.Contains(crash.Summary, pkg) || strings.Contains(crash.StackTrace, pkg)
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
