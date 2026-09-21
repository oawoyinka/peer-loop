// Package aiclient is a minimal client for the Anthropic Messages API,
// used to generate session recaps. It uses only the standard library —
// no SDK dependency — since it's a single endpoint.
package aiclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const apiURL = "https://api.anthropic.com/v1/messages"

// ErrNoAPIKey is returned when ANTHROPIC_API_KEY isn't set. Callers should
// treat this as "feature unavailable" rather than a hard failure.
var ErrNoAPIKey = errors.New("ANTHROPIC_API_KEY is not set")

type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient reads the API key from the ANTHROPIC_API_KEY environment
// variable. It's fine to construct this even if the key is unset —
// GenerateRecap will just return ErrNoAPIKey when called.
func NewClient() *Client {
	return &Client{
		apiKey:     os.Getenv("ANTHROPIC_API_KEY"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Available() bool {
	return c.apiKey != ""
}

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateRecap turns free-text session notes into a short, structured
// recap: what was covered, and 2-3 suggested next steps. Returns
// ErrNoAPIKey if no key is configured, so callers can show a friendly
// "AI recaps aren't set up" message instead of a raw error.
func (c *Client) GenerateRecap(skillName, notes string) (string, error) {
	if !c.Available() {
		return "", ErrNoAPIKey
	}
	if notes == "" {
		return "", errors.New("no session notes to summarize")
	}

	prompt := fmt.Sprintf(
		"You are summarizing a peer-mentoring session on the topic %q for a learning platform called PeerLoop. "+
			"Here are the raw notes one participant took during the session:\n\n%s\n\n"+
			"Write a short recap with two sections: \"What we covered\" (2-4 bullet points) and "+
			"\"Suggested next steps\" (2-3 bullet points). Keep it concise and specific to the notes given. "+
			"Do not invent details that aren't implied by the notes.",
		skillName, notes,
	)

	reqBody := messagesRequest{
		Model:     "claude-sonnet-5",
		MaxTokens: 500,
		Messages:  []message{{Role: "user", Content: prompt}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed messagesResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("parsing Anthropic response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("Anthropic API error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API returned status %d", resp.StatusCode)
	}
	if len(parsed.Content) == 0 {
		return "", errors.New("Anthropic API returned no content")
	}

	return parsed.Content[0].Text, nil
}
