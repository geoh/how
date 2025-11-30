package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Error types
type ApiError struct {
	Message string
}

func (e *ApiError) Error() string {
	return e.Message
}

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

type ContentError struct {
	Message string
}

func (e *ContentError) Error() string {
	return e.Message
}

type ApiTimeoutError struct {
	Message string
}

func (e *ApiTimeoutError) Error() string {
	return e.Message
}

// Request and Response structures for Gemini API
type geminiRequest struct {
	Contents []content `json:"contents"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates     []candidate     `json:"candidates,omitempty"`
	PromptFeedback *promptFeedback `json:"promptFeedback,omitempty"`
}

type candidate struct {
	Content       content `json:"content"`
	FinishReason  string  `json:"finishReason,omitempty"`
	SafetyRatings []struct {
		Category    string `json:"category"`
		Probability string `json:"probability"`
	} `json:"safetyRatings,omitempty"`
}

type promptFeedback struct {
	BlockReason string `json:"blockReason,omitempty"`
}

// GenerateResponse generates a response from the Gemini API
func GenerateResponse(apiKey, prompt string, maxRetries int) (string, error) {
	modelName := os.Getenv("HOW_MODEL")
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	// Remove "models/" prefix if present in environment variable
	modelName = strings.TrimPrefix(modelName, "models/")

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)

	// Create request body
	reqBody := geminiRequest{
		Contents: []content{
			{
				Parts: []part{
					{Text: prompt},
				},
			},
		},
	}

	timeout := 30 * time.Second
	client := &http.Client{
		Timeout: timeout + 5*time.Second,
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Marshal request body
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return "", &ApiError{Message: fmt.Sprintf("Failed to marshal request: %v", err)}
		}

		// Create HTTP request
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return "", &ApiError{Message: fmt.Sprintf("Failed to create request: %v", err)}
		}

		req.Header.Set("Content-Type", "application/json")

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
				if attempt == maxRetries-1 {
					return "", &ApiTimeoutError{Message: "API request timed out"}
				}
				time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
				continue
			}
			return "", &ApiError{Message: fmt.Sprintf("Request failed: %v", err)}
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return "", &ApiError{Message: fmt.Sprintf("Failed to read response: %v", err)}
		}

		// Check for HTTP errors
		if resp.StatusCode == 429 {
			if attempt == maxRetries-1 {
				return "", &ApiError{Message: "Rate limit exceeded"}
			}
			time.Sleep(time.Duration(1<<uint(attempt)+1) * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return "", &ApiError{Message: fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body))}
		}

		// Parse response
		var geminiResp geminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return "", &ApiError{Message: fmt.Sprintf("Failed to parse response: %v", err)}
		}

		// Check for blocked content
		if geminiResp.PromptFeedback != nil && geminiResp.PromptFeedback.BlockReason != "" {
			return "", &ContentError{Message: fmt.Sprintf("Blocked: %s", geminiResp.PromptFeedback.BlockReason)}
		}

		// Extract text from response
		if len(geminiResp.Candidates) == 0 {
			return "", &ContentError{Message: "Empty response from API"}
		}

		if len(geminiResp.Candidates[0].Content.Parts) == 0 {
			return "", &ContentError{Message: "No content parts in response"}
		}

		text := geminiResp.Candidates[0].Content.Parts[0].Text
		text = strings.TrimSpace(text)

		if text == "" {
			return "", &ContentError{Message: "Empty response from API"}
		}

		return text, nil
	}

	return "", &ApiError{Message: "Max retries exceeded"}
}
