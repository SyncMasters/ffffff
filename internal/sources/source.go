// Package sources defines provider contracts independent of HTTP, workers and output.
package sources

import (
	"context"
	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/security"
)

// Source names are stable registry identifiers. Type describes provenance.
type Source interface {
	Name() string
	Type() models.SourceType
}

// Emit provides streaming backpressure. A source must call it serially, stop
// when it returns an error, and not call it after its search method returns.
// Partial results are valid even when the search subsequently returns an error.
type Emit func(models.Result) error

type UsernameSearcher interface {
	SearchUsername(context.Context, string, Emit) error
}
type EmailSearcher interface {
	SearchEmail(context.Context, string, Emit) error
}
type PasswordSearcher interface {
	SearchPassword(context.Context, security.Secret, Emit) error
}
type PasswordHashSearcher interface {
	SearchPasswordHash(context.Context, security.Secret, Emit) error
}

// Capabilities derives support from implemented interfaces, so declarations
// cannot drift away from the operations actually provided.
func Capabilities(s Source) []models.TargetType {
	var out []models.TargetType
	if _, ok := s.(UsernameSearcher); ok {
		out = append(out, models.TargetUsername)
	}
	if _, ok := s.(EmailSearcher); ok {
		out = append(out, models.TargetEmail)
	}
	if _, ok := s.(PasswordSearcher); ok {
		out = append(out, models.TargetPassword)
	}
	if _, ok := s.(PasswordHashSearcher); ok {
		out = append(out, models.TargetPasswordHash)
	}
	return out
}
func Supports(s Source, kind models.TargetType) bool {
	for _, capability := range Capabilities(s) {
		if capability == kind {
			return true
		}
	}
	return false
}
