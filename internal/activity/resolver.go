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
		if name := ResolveName(ctx, a.ActivityID, r); name != "" {
			return name
		}
	}
	return a.Name()
}

// ResolveName resolves an activity ID to a human-readable name via the resolver.
// Returns empty string on failure or when the resolver has no mapping.
func ResolveName(ctx context.Context, id string, r Resolver) string {
	if id == "" || r == nil {
		return ""
	}
	name, err := r.Resolve(ctx, id)
	if err != nil || name == "" {
		return ""
	}
	return name
}
