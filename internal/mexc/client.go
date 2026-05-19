package mexc

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxRetries = 3
	backoffCap = 30 // seconds
)

var (
	ErrAuth      = errors.New("MEXC_AUTH: credentials rejected")
	ErrRateLimit = errors.New("MEXC_RATE_LIMIT: 429 after retries")
)

type Client struct {
	BaseURL   string
	APIKey    string
	APISecret string
	HTTP      *http.Client
	sleep     func(seconds int)
	now       func() time.Time
}

func New(baseURL, apiKey, apiSecret string) *Client {
	return &Client{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		APISecret: apiSecret,
		HTTP:      &http.Client{Timeout: 15 * time.Second},
		sleep:     func(s int) { time.Sleep(time.Duration(s) * time.Second) },
		now:       time.Now,
	}
}

// signRaw returns hex(HMAC_SHA256(secret, input)).
func (c *Client) signRaw(s string) string {
	h := hmac.New(sha256.New, []byte(c.APISecret))
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// sign returns hex(HMAC_SHA256(secret, apiKey+timestamp+payload)).
// payload is sorted-query-string for GET, raw body for POST.
func (c *Client) sign(timestamp, payload string) string {
	return c.signRaw(c.APIKey + timestamp + payload)
}

// sortedQuery returns the canonical query string for signing.
func sortedQuery(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + params[k]
	}
	return strings.Join(parts, "&")
}

// do executes a signed request. params goes to query for GET, JSON body for POST/DELETE.
// out is unmarshalled into *out.
func (c *Client) do(ctx context.Context, method, path string, params map[string]string, body any, out any) error {
	var payload string
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyBytes = b
		payload = string(b)
	} else if len(params) > 0 {
		payload = sortedQuery(params)
	}

	for attempt := 0; ; attempt++ {
		url := c.BaseURL + path
		if method == http.MethodGet && payload != "" {
			url += "?" + payload
		}
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		ts := strconv.FormatInt(c.now().UnixMilli(), 10)
		req.Header.Set("ApiKey", c.APIKey)
		req.Header.Set("Request-Time", ts)
		req.Header.Set("Signature", c.sign(ts, payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			if attempt >= maxRetries {
				return fmt.Errorf("after retries: %w", err)
			}
			c.sleep(backoff(attempt))
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 401 {
			return fmt.Errorf("%w: %s", ErrAuth, string(respBody))
		}
		if resp.StatusCode == 429 {
			if attempt >= maxRetries {
				return ErrRateLimit
			}
			wait := 1
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if n, err := strconv.Atoi(ra); err == nil {
					wait = n
				}
			}
			c.sleep(wait)
			continue
		}
		if resp.StatusCode >= 500 {
			if attempt >= maxRetries {
				return fmt.Errorf("server error %d after retries: %s", resp.StatusCode, string(respBody))
			}
			c.sleep(backoff(attempt))
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
		}
		if out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decode response: %w (body: %s)", err, string(respBody))
			}
		}
		return nil
	}
}

func backoff(attempt int) int {
	n := 1 << attempt
	if n > backoffCap {
		n = backoffCap
	}
	return n
}
