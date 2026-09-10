package models

// SourceType describes provenance, not a concrete integration.
type SourceType string

const (
	SourceWebsite SourceType = "website"
	SourceAPI     SourceType = "api"
	SourceLocal   SourceType = "local"
)

// Evidence contains only non-secret observations, never raw response bodies.
type Evidence struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}
