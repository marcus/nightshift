package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Ollama struct {
	client   *http.Client
	dataPath string
	mu       sync.RWMutex

	lastScrape   time.Time
	sessionPct   float64
	weeklyPct    float64
	sessionReset time.Time
	weeklyReset  time.Time
	scrapeError  error
}

func NewOllama() *Ollama {
	home, _ := os.UserHomeDir()
	return NewOllamaWithPath(filepath.Join(home, ".ollama"))
}

func NewOllamaWithPath(dataPath string) *Ollama {
	jar, _ := cookiejar.New(nil)
	return &Ollama{
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		dataPath: dataPath,
	}
}

func (o *Ollama) Name() string {
	return "ollama"
}

func (o *Ollama) Execute(ctx context.Context, task Task) (Result, error) {
	return Result{}, nil
}

func (o *Ollama) Cost() (inputCents, outputCents int64) {
	return 0, 0
}

func (o *Ollama) GetUsedPercent(mode string, weeklyBudget int64) (float64, error) {
	if err := o.scrapeIfNeeded(context.Background()); err != nil {
		return 0, fmt.Errorf("scraping ollama usage: %w", err)
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	switch mode {
	case "daily":
		return o.sessionPct, nil
	case "weekly":
		return o.weeklyPct, nil
	default:
		return 0, fmt.Errorf("invalid mode: %s (must be 'daily' or 'weekly')", mode)
	}
}

func (o *Ollama) GetResetTime(mode string) (time.Time, error) {
	if err := o.scrapeIfNeeded(context.Background()); err != nil {
		return time.Time{}, fmt.Errorf("scraping ollama usage: %w", err)
	}

	o.mu.RLock()
	defer o.mu.RUnlock()

	switch mode {
	case "daily":
		return o.sessionReset, nil
	case "weekly":
		return o.weeklyReset, nil
	default:
		return time.Time{}, fmt.Errorf("invalid mode: %s", mode)
	}
}

func (o *Ollama) LastUsedPercentSource() string {
	return "ollama-web-scrape"
}

func (o *Ollama) scrapeIfNeeded(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if time.Since(o.lastScrape) < 5*time.Minute {
		return o.scrapeError
	}

	sessionPct, weeklyPct, sessionReset, weeklyReset, err := scrapeOllamaSettings(ctx, o.client)
	o.lastScrape = time.Now()
	o.scrapeError = err

	if err == nil {
		o.sessionPct = sessionPct
		o.weeklyPct = weeklyPct
		o.sessionReset = sessionReset
		o.weeklyReset = weeklyReset
	}

	return err
}

func scrapeOllamaSettings(ctx context.Context, client *http.Client) (sessionPct, weeklyPct float64, sessionReset, weeklyReset time.Time, err error) {
	if err := loadOllamaCookies(client); err != nil {
		return 0, 0, time.Time{}, time.Time{}, fmt.Errorf("load cookies: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://ollama.com/settings", nil)
	if err != nil {
		return 0, 0, time.Time{}, time.Time{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, time.Time{}, time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, 0, time.Time{}, time.Time{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, time.Time{}, time.Time{}, err
	}

	return parseOllamaSettingsHTML(string(body))
}

func parseOllamaSettingsHTML(html string) (sessionPct, weeklyPct float64, sessionReset, weeklyReset time.Time, err error) {
	sessionPctRe := regexp.MustCompile(`<span class="text-sm">Session usage</span>\s*<span class="text-sm">(\d+(?:\.\d+)?)% used</span>`)
	weeklyPctRe := regexp.MustCompile(`<span class="text-sm">Weekly usage</span>\s*<span class="text-sm">(\d+(?:\.\d+)?)% used</span>`)
	resetTimeRe := regexp.MustCompile(`<div[^>]*class="text-xs text-neutral-500 mt-1 local-time"[^>]*data-time="([^"]+)"`)

	if m := sessionPctRe.FindStringSubmatch(html); len(m) == 2 {
		sessionPct, _ = parsePct(m[1])
	}

	if m := weeklyPctRe.FindStringSubmatch(html); len(m) == 2 {
		weeklyPct, _ = parsePct(m[1])
	}

	resetTimes := resetTimeRe.FindAllStringSubmatch(html, -1)
	if len(resetTimes) >= 1 {
		sessionReset, _ = time.Parse(time.RFC3339, resetTimes[0][1])
	}
	if len(resetTimes) >= 2 {
		weeklyReset, _ = time.Parse(time.RFC3339, resetTimes[1][1])
	}

	return sessionPct, weeklyPct, sessionReset, weeklyReset, nil
}

func parsePct(value string) (float64, error) {
	var pct float64
	_, err := fmt.Sscanf(strings.TrimSpace(value), "%f", &pct)
	if err != nil {
		return 0, fmt.Errorf("parse percent: %w", err)
	}
	if pct < 0 || pct > 100 {
		return 0, fmt.Errorf("percent out of range: %.2f", pct)
	}
	return pct, nil
}

func loadOllamaCookies(client *http.Client) error {
	home, _ := os.UserHomeDir()
	cookiesPath := filepath.Join(home, ".ollama", "cookies.txt")

	data, err := os.ReadFile(cookiesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cookie file not found at %s - run 'nightshift ollama auth' to set up authentication", cookiesPath)
		}
		return fmt.Errorf("read cookie file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	u, _ := url.Parse("https://ollama.com")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 7 {
			continue
		}

		domain := parts[0]
		secure := parts[3] == "TRUE"
		name := parts[5]
		value := parts[6]

		if !strings.Contains(domain, "ollama.com") {
			continue
		}

		client.Jar.SetCookies(u, []*http.Cookie{
			{
				Name:     name,
				Value:    value,
				Secure:   secure,
				HttpOnly: true,
				Path:     "/",
			},
		})
	}

	return nil
}