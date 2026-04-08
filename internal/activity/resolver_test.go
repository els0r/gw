package activity

import (
	"context"
	"testing"

	"github.com/els0r/gw/internal/session"
)

type stubResolver struct {
	names map[string]string
	err   error
}

func (s *stubResolver) Resolve(_ context.Context, id string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	name, ok := s.names[id]
	if !ok {
		return "", ErrNotFound
	}
	return name, nil
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		name       string
		activity   session.Activity
		resolver   Resolver
		want       string
	}{
		{
			name:     "resolved from activity ID",
			activity: session.Activity{ID: "feature-firewall-hits", ActivityID: "act-1"},
			resolver: &stubResolver{names: map[string]string{"act-1": "Deep Engineering"}},
			want:     "Deep Engineering",
		},
		{
			name:     "fallback to branch name when ID not found",
			activity: session.Activity{ID: "feature-firewall-hits", ActivityID: "act-missing"},
			resolver: &stubResolver{names: map[string]string{}},
			want:     "Firewall Hits",
		},
		{
			name:     "fallback when no activity ID",
			activity: session.Activity{ID: "fix-null-pointer"},
			resolver: &stubResolver{names: map[string]string{}},
			want:     "Null Pointer",
		},
		{
			name:     "fallback when resolver returns error",
			activity: session.Activity{ID: "chore-deps", ActivityID: "act-1"},
			resolver: &stubResolver{err: context.DeadlineExceeded},
			want:     "Deps",
		},
		{
			name:     "fallback when resolver is nil",
			activity: session.Activity{ID: "admin-misc", ActivityID: "act-1"},
			resolver: nil,
			want:     "Admin Misc",
		},
		{
			name:     "nop resolver always falls back",
			activity: session.Activity{ID: "feature-login", ActivityID: "act-1"},
			resolver: Nop{},
			want:     "Login",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DisplayName(context.Background(), tt.activity, tt.resolver)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
