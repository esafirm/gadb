package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/esafirm/gadb/config"
)

// AnalysisRequest represents a crash analysis request
type AnalysisRequest struct {
	CrashSummary string
	StackTrace   string
	ProcessID    string
	ThreadID     string
}

// PerformanceRequest represents a performance analysis request
type PerformanceRequest struct {
	Logs      string
	IsStartup bool
}

// AnalysisResponse represents the AI analysis response
type AnalysisResponse struct {
	Cause       string   `json:"cause"`
	Suggestions []string `json:"suggestions"`
	Severity    string   `json:"severity"`    // "low", "medium", "high", "critical"
	Keywords    []string `json:"keywords"`    // Important keywords from the crash
	Category    string   `json:"category"`    // e.g., "NullPointerException", "OutOfMemoryError", etc.
	Fixable     bool     `json:"fixable"`     // Whether this is likely a code issue
	Explanation string   `json:"explanation"` // Detailed explanation
	NextSteps   string   `json:"nextSteps"`   // What to do next
}

// AnthropicRequest represents a request to Anthropic's API
type AnthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents a response from Anthropic's API
type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// OpenAIRequest represents a request to OpenAI's API
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

// OpenAIResponse represents a response from OpenAI's API
type OpenAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// AnalyzeCrash performs AI analysis on a crash
func AnalyzeCrash(request AnalysisRequest) (*AnalysisResponse, error) {
	aiConfig, err := config.GetAIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get AI config: %w", err)
	}

	switch strings.ToLower(aiConfig.Provider) {
	case "anthropic":
		if aiConfig.APIKey == "" {
			return nil, fmt.Errorf("AI API key not configured. Run 'gadb config --ai' to set it up")
		}
		return analyzeWithAnthropic(buildPrompt(request), aiConfig)
	case "openai":
		if aiConfig.APIKey == "" {
			return nil, fmt.Errorf("AI API key not configured. Run 'gadb config --ai' to set it up")
		}
		return analyzeWithOpenAI(buildPrompt(request), aiConfig)
	case "gemini":
		return analyzeWithGeminiCLI(buildPrompt(request), aiConfig)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", aiConfig.Provider)
	}
}

// AnalyzePerformance performs AI analysis on performance logs
func AnalyzePerformance(request PerformanceRequest) (*AnalysisResponse, error) {
	aiConfig, err := config.GetAIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get AI config: %w", err)
	}

	prompt := buildPerformancePrompt(request)

	switch strings.ToLower(aiConfig.Provider) {
	case "anthropic":
		if aiConfig.APIKey == "" {
			return nil, fmt.Errorf("AI API key not configured. Run 'gadb config --ai' to set it up")
		}
		return analyzeWithAnthropic(prompt, aiConfig)
	case "openai":
		if aiConfig.APIKey == "" {
			return nil, fmt.Errorf("AI API key not configured. Run 'gadb config --ai' to set it up")
		}
		return analyzeWithOpenAI(prompt, aiConfig)
	case "gemini":
		return analyzeWithGeminiCLI(prompt, aiConfig)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", aiConfig.Provider)
	}
}

func analyzeWithAnthropic(prompt string, aiConfig config.AIConfig) (*AnalysisResponse, error) {

	anthropicReq := AnthropicRequest{
		Model:     aiConfig.Model,
		MaxTokens: aiConfig.MaxTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := "https://api.anthropic.com/v1/messages"
	if aiConfig.Endpoint != "" {
		endpoint = aiConfig.Endpoint
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", aiConfig.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	return parseAnalysisResponse(anthropicResp.Content[0].Text)
}

func analyzeWithOpenAI(prompt string, aiConfig config.AIConfig) (*AnalysisResponse, error) {
	openaiReq := OpenAIRequest{
		Model: aiConfig.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert Android performance analyzer. Analyze logcat output and provide actionable insights on performance issues, startup time, and resource usage.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   aiConfig.MaxTokens,
		Temperature: aiConfig.Temperature,
	}

	jsonData, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := "https://api.openai.com/v1/chat/completions"
	if aiConfig.Endpoint != "" {
		endpoint = aiConfig.Endpoint
	}

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+aiConfig.APIKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var openaiResp OpenAIResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return parseAnalysisResponse(openaiResp.Choices[0].Message.Content)
}

func buildPrompt(request AnalysisRequest) string {
	return fmt.Sprintf(`Analyze this Android crash log and provide actionable insights:

Crash Summary: %s
Stack Trace:
%s
Process ID: %s
Thread ID: %s

Please provide a JSON response in the following format:
{
  "cause": "Brief explanation of what caused the crash",
  "suggestions": ["List of 2-3 actionable suggestions to fix the issue"],
  "severity": "low|medium|high|critical",
  "keywords": ["keyword1", "keyword2"],
  "category": "e.g., NullPointerException, OutOfMemoryError, etc.",
  "fixable": true/false,
  "explanation": "Detailed technical explanation",
  "nextSteps": "What should the developer do next?"
}

Focus on being concise and practical. The suggestions should be actionable steps the developer can take.`, request.CrashSummary, request.StackTrace, request.ProcessID, request.ThreadID)
}

func buildPerformancePrompt(request PerformanceRequest) string {
	analysisType := "performance issues"
	if request.IsStartup {
		analysisType = "app startup performance"
	}

	return fmt.Sprintf(`Analyze these Android logcat logs for %s and provide actionable insights.
Look for:
- Slow operations (e.g., "Displayed ... +345ms", "Slow operation", etc.)
- Blocking UI threads (ANRs, "The application may be doing too much work on its main thread")
- Resource contention or heavy GC (Garbage Collection)
- Network latencies or failures
- Activity/Fragment lifecycle delays

Logs:
%s

Please provide a JSON response in the following format:
{
  "cause": "Summary of the most significant performance bottleneck identified",
  "suggestions": ["List of 2-3 actionable suggestions to improve performance"],
  "severity": "low|medium|high|critical",
  "keywords": ["keyword1", "keyword2"],
  "category": "e.g., StartupTime, MainThreadBlocking, MemoryManagement, NetworkLatency",
  "fixable": true/false,
  "explanation": "Detailed technical analysis of the performance logs provided",
  "nextSteps": "What should the developer do next to optimize?"
}

Focus on being concise and practical.`, analysisType, request.Logs)
}

func analyzeWithGeminiCLI(prompt string, aiConfig config.AIConfig) (*AnalysisResponse, error) {

	// Build gemini-cli command
	var cmd *exec.Cmd
	if aiConfig.Model != "" {
		// Use specified model
		cmd = exec.Command("gemini", "chat", "--model", aiConfig.Model)
	} else {
		// Use default model
		cmd = exec.Command("gemini", "chat")
	}

	// Set input to prompt
	cmd.Stdin = strings.NewReader(prompt)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run gemini-cli: %w\nMake sure gemini-cli is installed: https://github.com/google/generative-ai-cli", err)
	}

	// Parse the response
	response := string(output)
	return parseAnalysisResponse(response)
}

func parseAnalysisResponse(aiResponse string) (*AnalysisResponse, error) {
	// Try to extract JSON from the response
	startIdx := strings.Index(aiResponse, "{")
	if startIdx == -1 {
		return nil, fmt.Errorf("no JSON found in AI response")
	}

	endIdx := strings.LastIndex(aiResponse, "}")
	if endIdx == -1 {
		return nil, fmt.Errorf("malformed JSON in AI response")
	}

	jsonStr := aiResponse[startIdx : endIdx+1]

	var response AnalysisResponse
	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
	}

	return &response, nil
}

// FormatAnalysisResponse formats the AI analysis for display
func FormatAnalysisResponse(response *AnalysisResponse) string {
	var builder strings.Builder

	// Severity indicator
	severityIcon := "✓"
	switch response.Severity {
	case "high":
		severityIcon = "⚠"
	case "critical":
		severityIcon = "✗"
	}

	builder.WriteString(fmt.Sprintf("%s Severity: %s\n", severityIcon, response.Severity))
	builder.WriteString(fmt.Sprintf("Category: %s\n", response.Category))
	builder.WriteString(fmt.Sprintf("Fixable: %t\n\n", response.Fixable))

	builder.WriteString("🔍 Cause:\n")
	builder.WriteString(fmt.Sprintf("   %s\n\n", response.Cause))

	builder.WriteString("💡 Suggestions:\n")
	for i, suggestion := range response.Suggestions {
		builder.WriteString(fmt.Sprintf("   %d. %s\n", i+1, suggestion))
	}
	builder.WriteString("\n")

	if response.Keywords != nil && len(response.Keywords) > 0 {
		builder.WriteString("🏷 Keywords: ")
		for i, keyword := range response.Keywords {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(keyword)
		}
		builder.WriteString("\n\n")
	}

	builder.WriteString("📋 Next Steps:\n")
	builder.WriteString(fmt.Sprintf("   %s\n", response.NextSteps))

	return builder.String()
}
