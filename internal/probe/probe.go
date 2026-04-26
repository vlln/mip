package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/vlln/mip/internal/ref"
	"github.com/vlln/mip/internal/rewrite"
)

type Result struct {
	Image       string `json:"image"`
	Mirror      string `json:"mirror,omitempty"`
	OK          bool   `json:"ok"`
	StatusCode  int    `json:"status_code,omitempty"`
	LatencyMS   int64  `json:"latency_ms"`
	Error       string `json:"error,omitempty"`
	Digest      string `json:"digest,omitempty"`
	IndexDigest string `json:"index_digest,omitempty"`
	MediaType   string `json:"media_type,omitempty"`
	Platform    string `json:"platform,omitempty"`
	PlatformHit bool   `json:"platform_hit,omitempty"`
}

type Options struct {
	Timeout     time.Duration
	Concurrency int
	Platform    string
}

func Candidates(ctx context.Context, candidates []rewrite.Candidate, options Options) []Result {
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
				results[index] = Image(ctx, client, candidate.Image, candidate.Mirror.Name, options.Platform)
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

func Image(ctx context.Context, client *http.Client, image string, mirrorName string, platform string) Result {
	start := time.Now()
	result := Result{Image: image, Mirror: mirrorName, Platform: platform}

	parsed, err := ref.Parse(image)
	if err != nil {
		result.Error = err.Error()
		result.LatencyMS = elapsedMS(start)
		return result
	}

	manifestURL := manifestURL(parsed)
	manifest, err := requestManifest(ctx, client, manifestURL)
	if err != nil {
		result.Error = err.Error()
		result.StatusCode = manifest.StatusCode
		result.LatencyMS = elapsedMS(start)
		return result
	}

	result.OK = manifest.StatusCode >= 200 && manifest.StatusCode < 300
	result.StatusCode = manifest.StatusCode
	result.Digest = manifest.Digest
	result.IndexDigest = manifest.Digest
	result.MediaType = manifest.MediaType
	if platform != "" {
		selected, ok := selectPlatformDigest(manifest.Body, platform)
		if ok {
			result.Digest = selected
			result.PlatformHit = true
		} else if isIndexMediaType(manifest.MediaType) {
			result.OK = false
			result.Error = "platform not found in manifest list"
		}
	}
	result.LatencyMS = elapsedMS(start)
	return result
}

type manifestResponse struct {
	StatusCode int
	Digest     string
	MediaType  string
	Body       []byte
}

func requestManifest(ctx context.Context, client *http.Client, manifestURL string) (manifestResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return manifestResponse{}, err
	}
	setManifestHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return manifestResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	if resp.StatusCode != http.StatusUnauthorized {
		manifest := manifestResponse{
			StatusCode: resp.StatusCode,
			Digest:     resp.Header.Get("Docker-Content-Digest"),
			MediaType:  responseMediaType(resp, body),
			Body:       body,
		}
		return manifest, statusErr(resp.StatusCode)
	}

	authHeader := resp.Header.Get("WWW-Authenticate")
	token, err := fetchBearerToken(ctx, client, authHeader)
	if err != nil {
		return manifestResponse{StatusCode: resp.StatusCode}, err
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return manifestResponse{}, err
	}
	setManifestHeaders(req)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = client.Do(req)
	if err != nil {
		return manifestResponse{}, err
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(io.LimitReader(resp.Body, 4<<20))

	manifest := manifestResponse{
		StatusCode: resp.StatusCode,
		Digest:     resp.Header.Get("Docker-Content-Digest"),
		MediaType:  responseMediaType(resp, body),
		Body:       body,
	}
	return manifest, statusErr(resp.StatusCode)
}

func fetchBearerToken(ctx context.Context, client *http.Client, authHeader string) (string, error) {
	params, ok := parseBearerChallenge(authHeader)
	if !ok {
		return "", fmt.Errorf("unsupported auth challenge")
	}

	realm := params["realm"]
	if realm == "" {
		return "", fmt.Errorf("auth challenge missing realm")
	}

	tokenURL, err := url.Parse(realm)
	if err != nil {
		return "", err
	}
	query := tokenURL.Query()
	for _, key := range []string{"service", "scope"} {
		if value := params[key]; value != "" {
			query.Set(key, value)
		}
	}
	tokenURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token request failed: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	token := extractJSONToken(string(body))
	if token == "" {
		return "", fmt.Errorf("token response missing token")
	}
	return token, nil
}

func parseBearerChallenge(header string) (map[string]string, bool) {
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return nil, false
	}

	header = strings.TrimSpace(header[len("Bearer "):])
	params := map[string]string{}
	for _, part := range splitChallengeParams(header) {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		params[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"`)
	}
	return params, true
}

func splitChallengeParams(input string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	for _, r := range input {
		switch r {
		case '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case ',':
			if inQuote {
				current.WriteRune(r)
				continue
			}
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func extractJSONToken(body string) string {
	var response struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		return ""
	}
	if response.Token != "" {
		return response.Token
	}
	if response.AccessToken != "" {
		return response.AccessToken
	}
	return ""
}

type manifestList struct {
	MediaType string `json:"mediaType"`
	Manifests []struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Platform  struct {
			Architecture string   `json:"architecture"`
			OS           string   `json:"os"`
			Variant      string   `json:"variant"`
			OSVersion    string   `json:"os.version"`
			OSFeatures   []string `json:"os.features"`
		} `json:"platform"`
	} `json:"manifests"`
}

type mediaTypedManifest struct {
	MediaType string `json:"mediaType"`
}

type platformSpec struct {
	OS           string
	Architecture string
	Variant      string
}

func selectPlatformDigest(body []byte, platform string) (string, bool) {
	want, ok := parsePlatform(platform)
	if !ok {
		return "", false
	}

	var list manifestList
	if err := json.Unmarshal(body, &list); err != nil {
		return "", false
	}
	if len(list.Manifests) == 0 {
		return "", false
	}
	for _, manifest := range list.Manifests {
		if manifest.Platform.OS != want.OS {
			continue
		}
		if normalizeArch(manifest.Platform.Architecture) != normalizeArch(want.Architecture) {
			continue
		}
		if want.Variant != "" && manifest.Platform.Variant != want.Variant {
			continue
		}
		if manifest.Digest != "" {
			return manifest.Digest, true
		}
	}
	return "", false
}

func parsePlatform(platform string) (platformSpec, bool) {
	parts := strings.Split(platform, "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] == "" || parts[1] == "" {
		return platformSpec{}, false
	}
	spec := platformSpec{OS: parts[0], Architecture: normalizeArch(parts[1])}
	if len(parts) == 3 {
		spec.Variant = parts[2]
	}
	return spec, true
}

func normalizeArch(arch string) string {
	switch arch {
	case "x86_64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		return arch
	}
}

func isIndexMediaType(mediaType string) bool {
	switch mediaType {
	case "application/vnd.oci.image.index.v1+json", "application/vnd.docker.distribution.manifest.list.v2+json":
		return true
	default:
		return false
	}
}

func responseMediaType(resp *http.Response, body []byte) string {
	if mediaType := resp.Header.Get("Content-Type"); mediaType != "" {
		return strings.Split(mediaType, ";")[0]
	}
	var manifest mediaTypedManifest
	if err := json.Unmarshal(body, &manifest); err == nil {
		return manifest.MediaType
	}
	return ""
}

func manifestURL(image ref.Reference) string {
	reference := image.Tag
	if image.Digest != "" {
		reference = image.Digest
	}
	return fmt.Sprintf("https://%s/v2/%s/manifests/%s", image.Registry, image.Repository, reference)
}

func setManifestHeaders(req *http.Request) {
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.v1+json",
	}, ", "))
	req.Header.Set("User-Agent", "mip/0.1")
}

func statusErr(status int) error {
	if status >= 200 && status < 300 {
		return nil
	}
	return fmt.Errorf("HTTP %d", status)
}

func elapsedMS(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
