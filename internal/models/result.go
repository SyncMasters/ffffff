package models

import (
	"encoding/json"
	"time"

	"github.com/johan-larp/agentsearch/internal/security"
)

// ResultStatus describes the outcome of a source lookup.
type ResultStatus string

const (
	StatusFound    ResultStatus = "found"
	StatusNotFound ResultStatus = "not_found"
	StatusBlocked  ResultStatus = "blocked"
	StatusError    ResultStatus = "error"
)

// Result is a normalized observation from any source. Legacy fields remain for
// existing writers and callers; sensitive input is never a result payload.
type Result struct {
	// Source identifies the producing provider; SiteName is a legacy display alias.
	Source     string     `json:"source,omitempty"`
	SourceType SourceType `json:"source_type,omitempty"`
	// Target and TargetType form a safe reference, preserving the legacy target string.
	TargetType TargetType        `json:"target_type,omitempty"`
	Evidence   []Evidence        `json:"evidence,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	SiteName   string            `json:"site_name" csv:"site_name"`
	Target     string            `json:"target" csv:"target"`
	URL        string            `json:"url" csv:"url"`
	Found      bool              `json:"found" csv:"found"`
	Confidence int               `json:"confidence" csv:"confidence"`
	Status     ResultStatus      `json:"status" csv:"status"`
	Duration   time.Duration     `json:"duration" csv:"duration"`
	Error      string            `json:"error,omitempty" csv:"error"`
	FinalURL   string            `json:"final_url,omitempty" csv:"final_url"`
}

// NewResult creates a safe result reference without copying sensitive input.
func NewResult(source string, sourceType SourceType, target Target) Result {
	return Result{Source: source, SourceType: sourceType, Target: target.String(), TargetType: target.Type()}
}

// Normalized adapts legacy results and returns an output-safe copy. Explicit
// status is authoritative. Confidence and the existing status vocabulary remain
// unchanged. Source implementations must never put secrets in observations.
func (r Result) Normalized() Result { return r.Redacted().normalized() }
func (r Result) normalized() Result {
	if r.Source == "" {
		r.Source = r.SiteName
	}
	if r.SiteName == "" {
		r.SiteName = r.Source
	}
	if r.TargetType == "" && r.Target != "" {
		t, _ := LegacyTarget(r.Target)
		r.TargetType = t.Type()
	}
	if r.Status == "" {
		switch {
		case r.Error != "":
			r.Status = StatusError
		case r.Found:
			r.Status = StatusFound
		default:
			r.Status = StatusNotFound
		}
	}
	r.Found = r.Status == StatusFound
	return r
}

// Redacted makes a copy, including maps/slices. Known secrets must be passed by
// the source dispatcher while it still holds input; they are never retained.
func (r Result) Redacted(secrets ...string) Result {
	if r.TargetType.Sensitive() {
		secrets = append(append([]string(nil), secrets...), r.Target)
		r.Target = security.Redacted
	}
	clean := func(s string) string { return security.Redact(s, secrets...) }
	r.Source = clean(r.Source)
	r.SiteName = clean(r.SiteName)
	r.Target = clean(r.Target)
	r.URL = clean(r.URL)
	r.FinalURL = clean(r.FinalURL)
	r.Error = clean(r.Error)
	r.Metadata = security.Fields(r.Metadata, secrets...)
	if r.Evidence != nil {
		evidence := make([]Evidence, len(r.Evidence))
		for i, e := range r.Evidence {
			value := clean(e.Value)
			if security.SensitiveKey(e.Kind) {
				value = security.Redacted
			}
			evidence[i] = Evidence{Kind: clean(e.Kind), Value: value}
		}
		r.Evidence = evidence
	}
	return r
}

// MarshalJSON is a final safety net for callers not using the dispatcher.
func (r Result) MarshalJSON() ([]byte, error) {
	type plain Result
	return json.Marshal(plain(r.Normalized()))
}
