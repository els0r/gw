package activity

import (
	"context"

	"github.com/els0r/gw/internal/session"
)

// Resolver maps an external activity ID to a human-readable name.
type Resolver interface {
	Resolve(ctx context.Context, id string) (string, error)
}

// Nop is a resolver that always returns empty, signaling fallback to branch name.
type Nop struct{}

// Resolve always returns empty string for the nop resolver.
func (Nop) Resolve(_ context.Context, _ string) (string, error) {
	return "", nil
}

// DisplayName resolves a human-readable name for an activity.
// Tries the resolver first; falls back to the branch-derived name.
func DisplayName(ctx context.Context, a session.Activity, r Resolver) string {
	if a.ActivityID != "" && r != nil {
		name, err := r.Resolve(ctx, a.ActivityID)
		if err == nil && name != "" {
			return name
		}
	}
	return a.Name()
}
