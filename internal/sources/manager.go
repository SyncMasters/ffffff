package sources

import (
	"context"
	"errors"
	"fmt"

	"github.com/johan-larp/agentsearch/internal/models"
	"github.com/johan-larp/agentsearch/internal/security"
)

var ErrUnsupportedTarget = errors.New("no source supports target type")

type Manager struct{ registry *Registry }

func NewManager(registry *Registry) *Manager { return &Manager{registry: registry} }

// Search dispatches only supported operations. Providers own their concurrency:
// a local lookup is not forced through website jobs or HTTP rate limiting.
// Provider errors do not discard partial results or prevent subsequent providers.
func (m *Manager) Search(ctx context.Context, target models.Target, emit Emit) error {
	if !target.Valid() {
		return fmt.Errorf("invalid target")
	}
	if target.Type().Sensitive() {
		defer target.Secret().Destroy()
	}
	if emit == nil {
		return fmt.Errorf("nil result consumer")
	}
	if m.registry == nil {
		return ErrUnsupportedTarget
	}
	var failures []error
	supported := false
	for _, s := range m.registry.All() {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if !Supports(s, target.Type()) {
			continue
		}
		supported = true
		var consumerErr error
		deliver := func(r models.Result) error {
			if consumerErr != nil {
				return consumerErr
			}
			if r.Source == "" {
				r.Source = s.Name()
			}
			r.SourceType = s.Type()
			r.Target = target.String()
			r.TargetType = target.Type()
			if target.Type().Sensitive() {
				r = r.Redacted(target.Value())
			}
			consumerErr = emit(r.Normalized())
			return consumerErr
		}
		var err error
		switch target.Type() {
		case models.TargetDomain:
			err = s.(DomainSearcher).SearchDomain(ctx, target.Value(), deliver)
		case models.TargetUsername:
			err = s.(UsernameSearcher).SearchUsername(ctx, target.Value(), deliver)
		case models.TargetEmail:
			err = s.(EmailSearcher).SearchEmail(ctx, target.Value(), deliver)
		case models.TargetPassword:
			err = s.(PasswordSearcher).SearchPassword(ctx, target.Secret(), deliver)
		case models.TargetPasswordHash:
			err = s.(PasswordHashSearcher).SearchPasswordHash(ctx, target.Secret(), deliver)
		}
		if consumerErr != nil {
			return consumerErr
		}
		if err != nil {
			// Do not retain a provider error containing sensitive input downstream.
			message := security.Redact(err.Error())
			if target.Type().Sensitive() {
				message = security.Redact(message, target.Value())
			}
			failures = append(failures, fmt.Errorf("source %s: %s", s.Name(), message))
		}
	}
	if !supported {
		return ErrUnsupportedTarget
	}
	if err := ctx.Err(); err != nil {
		failures = append(failures, err)
	}
	return errors.Join(failures...)
}
