package detector

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/johan-larp/agentsearch/internal/models"
)

// DetectionResult describes the interpretation of a server response.
type DetectionResult struct {
	Found      bool
	Confidence int
	Status     models.ResultStatus
}

// Engine applies declarative site detection rules.
type Engine struct{}

// New creates a detection engine.
func New() *Engine {
	return &Engine{}
}

// Analyze interprets an HTTP response using SiteConfig rules.
// Rules are applied in this order:
//  1. WAF and protection indicators
//  2. Redirect rules for the final URL
//  3. The configured check type
func (e *Engine) Analyze(site models.SiteConfig, resp *http.Response, body, finalURL string) DetectionResult {
	// Detect WAF responses first.
	if isWAF(resp, body, site.WAFIndicators) {
		return DetectionResult{Status: models.StatusBlocked, Confidence: 0}
	}

	// Honor explicit protection markers.
	for _, p := range site.Protection {
		if p == "cf_js_challenge" || p == "custom_bot_protection" {
			// An explicit protection marker forces a blocked result,
			// even if response-based WAF heuristics do not match.
			return DetectionResult{Status: models.StatusBlocked, Confidence: 0}
		}
	}

	// Check the final URL for redirect failure patterns.
	for _, pattern := range site.RedirectFailurePatterns {
		if strings.Contains(finalURL, pattern) {
			return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
		}
	}

	// Apply the configured detection strategy.
	switch site.CheckType {
	case "status_code":
		return e.checkStatusCode(site, resp)
	case "message":
		return e.checkMessage(site, body)
	case "response_url":
		return e.checkResponseURL(site, finalURL)
	case "header":
		return e.checkHeaders(site, resp)
	default:
		// Use status heuristics when no check type is configured.
		if site.ErrorCode != 0 && resp.StatusCode == site.ErrorCode {
			return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return DetectionResult{Found: true, Status: models.StatusFound, Confidence: calculateBaseConfidence(site, resp, body)}
		}
		return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
	}
}

func (e *Engine) checkStatusCode(site models.SiteConfig, resp *http.Response) DetectionResult {
	if site.ErrorCode != 0 && resp.StatusCode == site.ErrorCode {
		return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
	}
	// Treat 2xx and 3xx responses as evidence of a profile.
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		conf := calculateBaseConfidence(site, resp, "")
		return DetectionResult{Found: true, Status: models.StatusFound, Confidence: conf}
	}
	return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
}

func (e *Engine) checkMessage(site models.SiteConfig, body string) DetectionResult {
	// Absence indicators take precedence over presence indicators.
	for _, s := range site.AbsenceStrs {
		if strings.Contains(body, s) {
			return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
		}
	}
	for _, reStr := range site.AbsenceRegexes {
		if matched, _ := regexp.MatchString(reStr, body); matched {
			return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
		}
	}

	// Check presence indicators next.
	for _, s := range site.PresenceStrs {
		if strings.Contains(body, s) {
			conf := calculateBaseConfidence(site, nil, body)
			return DetectionResult{Found: true, Status: models.StatusFound, Confidence: conf}
		}
	}
	for _, reStr := range site.PresenceRegexes {
		if matched, _ := regexp.MatchString(reStr, body); matched {
			conf := calculateBaseConfidence(site, nil, body)
			return DetectionResult{Found: true, Status: models.StatusFound, Confidence: conf}
		}
	}

	// Preserve uncertain matches with a low confidence score.
	conf := calculateBaseConfidence(site, nil, body)
	if conf < 30 {
		conf = 30
	}
	return DetectionResult{Found: true, Status: models.StatusFound, Confidence: conf}
}

func (e *Engine) checkResponseURL(site models.SiteConfig, finalURL string) DetectionResult {
	if site.ErrorURL != "" && strings.Contains(finalURL, site.ErrorURL) {
		return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
	}
	return DetectionResult{Found: true, Status: models.StatusFound, Confidence: calculateBaseConfidence(site, nil, "")}
}

func (e *Engine) checkHeaders(site models.SiteConfig, resp *http.Response) DetectionResult {
	for k, v := range site.RequiredHeaders {
		if resp.Header.Get(k) == v {
			conf := calculateBaseConfidence(site, resp, "")
			return DetectionResult{Found: true, Status: models.StatusFound, Confidence: conf}
		}
	}
	return DetectionResult{Status: models.StatusNotFound, Confidence: 0}
}

// isWAF identifies known protection responses and configured WAF indicators.
func isWAF(resp *http.Response, body string, cfg models.WAFConfig) bool {
	lowerBody := strings.ToLower(body)

	// Built-in heuristics.
	if resp.Header.Get("CF-RAY") != "" {
		return true
	}
	if strings.Contains(strings.ToLower(resp.Header.Get("Server")), "cloudflare") {
		return true
	}
	if resp.StatusCode == 403 && (strings.Contains(lowerBody, "cloudflare") || strings.Contains(lowerBody, "ray id")) {
		return true
	}
	if resp.StatusCode == 429 {
		return true
	}
	if strings.Contains(lowerBody, "ddos-guard") || strings.Contains(lowerBody, "incapsula") || strings.Contains(lowerBody, "sucuri") {
		return true
	}

	// Site-specific WAF rules.
	for _, code := range cfg.StatusCodes {
		if resp.StatusCode == code {
			return true
		}
	}
	for _, key := range cfg.HeaderKeys {
		if resp.Header.Get(key) != "" {
			return true
		}
	}
	for _, sub := range cfg.BodySubstrings {
		if strings.Contains(lowerBody, strings.ToLower(sub)) {
			return true
		}
	}

	return false
}

// calculateBaseConfidence combines the site weight and response heuristics.
func calculateBaseConfidence(site models.SiteConfig, resp *http.Response, body string) int {
	score := site.Weight
	if score == 0 {
		score = 10
	}

	if resp != nil && resp.StatusCode == 200 {
		score += 30
	}

	// A substantial body contributes a small confidence bonus.
	if len(body) > 200 {
		score += 10
	}

	if score > 100 {
		score = 100
	}
	return score
}
