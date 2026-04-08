package activity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	earlyDefaultBase = "https://api.early.app/api/v4"
	tokenMaxAge      = 23 * time.Hour
)

// ErrNotFound indicates the activity ID was not found.
var ErrNotFound = errors.New("activity not found")

// EarlyResolver resolves EARLY activity IDs to human-readable names.
type EarlyResolver struct {
	baseURL string
	token   string
	client  *http.Client

	once  sync.Once
	cache map[string]string
	err   error
}

// NewEarlyResolver creates a resolver using the given bearer token.
func NewEarlyResolver(token string) *EarlyResolver {
	return &EarlyResolver{
		baseURL: earlyDefaultBase,
		token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Resolve returns the activity name for the given EARLY activity ID.
func (r *EarlyResolver) Resolve(ctx context.Context, id string) (string, error) {
	r.once.Do(func() {
		r.cache, r.err = r.fetchActivities(ctx)
	})
	if r.err != nil {
		return "", fmt.Errorf("fetching activities: %w", r.err)
	}

	name, ok := r.cache[id]
	if !ok {
		return "", ErrNotFound
	}
	return name, nil
}

type earlyActivitiesResponse struct {
	Activities []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"activities"`
}

func (r *EarlyResolver) fetchActivities(ctx context.Context) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/activities", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var body earlyActivitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	m := make(map[string]string, len(body.Activities))
	for _, a := range body.Activities {
		m[a.ID] = a.Name
	}
	return m, nil
}

// EarlyToken reads a cached EARLY bearer token from stateDir/early_token,
// or exchanges API credentials for a fresh one. Shares the token cache
// with the shell hooks in lib.sh.
func EarlyToken(ctx context.Context, stateDir, apiKey, apiSecret string) (string, error) {
	cachePath := filepath.Join(stateDir, "early_token")

	// try cached token
	if info, err := os.Stat(cachePath); err == nil {
		age := time.Since(info.ModTime())
		if age < tokenMaxAge {
			data, err := os.ReadFile(cachePath)
			if err == nil {
				tok := strings.TrimSpace(string(data))
				if tok != "" {
					return tok, nil
				}
			}
		}
	}

	// exchange credentials
	tok, err := earlySignIn(ctx, apiKey, apiSecret)
	if err != nil {
		return "", err
	}

	// cache for next time
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return tok, nil // non-fatal: token works, cache just won't persist
	}
	_ = os.WriteFile(cachePath, []byte(tok+"\n"), 0o600)

	return tok, nil
}

type earlySignInRequest struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
}

type earlySignInResponse struct {
	Token string `json:"token"`
}

func earlySignIn(ctx context.Context, apiKey, apiSecret string) (string, error) {
	body, err := json.Marshal(earlySignInRequest{APIKey: apiKey, APISecret: apiSecret})
	if err != nil {
		return "", fmt.Errorf("marshaling sign-in request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, earlyDefaultBase+"/developer/sign-in", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("creating sign-in request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sign-in request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sign-in returned status %d", resp.StatusCode)
	}

	var result earlySignInResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding sign-in response: %w", err)
	}
	if result.Token == "" {
		return "", errors.New("sign-in returned empty token")
	}

	return result.Token, nil
}
