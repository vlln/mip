package gitprobe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vlln/mip/internal/gitrewrite"
	"github.com/vlln/mip/internal/giturl"
)

// Result holds the probe outcome for a single mirror URL.
type Result struct {
	URL          string `json:"url"`
	Mirror       string `json:"mirror,omitempty"`
	OK           bool   `json:"ok"`
	AuthRequired bool   `json:"auth_required,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
	LatencyMS    int64  `json:"latency_ms"`
	Warning      string `json:"warning,omitempty"`
	Error        string `json:"error,omitempty"`
}

// Options controls probe behavior.
type Options struct {
	Timeout     time.Duration
	Concurrency int
}

// Candidates probes all mirror candidates concurrently.
func Candidates(ctx context.Context, candidates []gitrewrite.Candidate, kind giturl.Kind, options Options) []Result {
	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Second
	}
	if options.Concurrency <= 0 {
		options.Concurrency = 6
	}

	results := make([]Result, len(candidates))
	jobs := make(chan int)
	var wg sync.WaitGroup

	workerCount := options.Concurrency
	if workerCount > len(candidates) {
		workerCount = len(candidates)
	}

	client := &http.Client{Timeout: options.Timeout}
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				candidate := candidates[index]
				results[index] = probeOne(ctx, client, candidate, kind)
			}
		}()
	}

	for index := range candidates {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	return results
}

func probeOne(ctx context.Context, client *http.Client, candidate gitrewrite.Candidate, kind giturl.Kind) Result {
	start := time.Now()
	result := Result{URL: candidate.URL, Mirror: candidate.Mirror.Name}

	probeURL := candidate.URL
	method := http.MethodHead
	if kind == giturl.KindClone {
		probeURL = infoRefsURL(candidate.URL)
		method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, method, probeURL, nil)
	if err != nil {
		result.Error = err.Error()
		result.LatencyMS = elapsedMS(start)
		return result
	}

	req.Header.Set("User-Agent", "git/gip")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.LatencyMS = elapsedMS(start)
		return result
	}
	defer resp.Body.Close()

	// For git info/refs, read the response to confirm it's a valid git advertisement
	if kind == giturl.KindClone && resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		if !isGitAdvertisement(body) {
			result.StatusCode = resp.StatusCode
			result.Error = fmt.Sprintf("unexpected git response: %s", string(body[:min(len(body), 100)]))
			result.LatencyMS = elapsedMS(start)
			return result
		}
	}

	result.StatusCode = resp.StatusCode
	result.LatencyMS = elapsedMS(start)

	if resp.StatusCode == http.StatusUnauthorized {
		result.AuthRequired = true
		result.Warning = "authentication required"
		return result
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.OK = true
	} else {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

func isGitAdvertisement(body []byte) bool {
	// Git smart HTTP response starts with "001e# service=git-upload-pack" or similar
	s := string(body)
	return strings.Contains(s, "service=git-upload-pack") ||
		strings.Contains(s, "service=git-receive-pack") ||
		strings.Contains(s, "service=git-upload-archive")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func infoRefsURL(mirrorURL string) string {
	u := strings.TrimSuffix(mirrorURL, "/")
	u = strings.TrimSuffix(u, ".git")
	return u + "/info/refs?service=git-upload-pack"
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}