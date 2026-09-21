package breaker

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

func TestNormalizeConfig(t *testing.T) {
	t.Parallel()

	customReadyToTrip := func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= 2
	}

	tests := []struct {
		name    string
		cfg     BreakerConfig
		wantErr bool
		assert  func(t *testing.T, got BreakerConfig)
	}{
		{
			name:    "missing name",
			cfg:     BreakerConfig{},
			wantErr: true,
		},
		{
			name: "sets defaults",
			cfg:  BreakerConfig{Name: "payments"},
			assert: func(t *testing.T, got BreakerConfig) {
				t.Helper()
				if got.Timeout != 30*time.Second {
					t.Fatalf("Timeout = %s, want %s", got.Timeout, 30*time.Second)
				}
				if got.Interval != 60*time.Second {
					t.Fatalf("Interval = %s, want %s", got.Interval, 60*time.Second)
				}
				if got.MaxRequests != 1 {
					t.Fatalf("MaxRequests = %d, want 1", got.MaxRequests)
				}
				if got.ReadyToTrip == nil {
					t.Fatal("ReadyToTrip = nil, want default function")
				}
				if got.ReadyToTrip(gobreaker.Counts{ConsecutiveFailures: 4}) {
					t.Fatal("ReadyToTrip(4 failures) = true, want false")
				}
				if !got.ReadyToTrip(gobreaker.Counts{ConsecutiveFailures: 5}) {
					t.Fatal("ReadyToTrip(5 failures) = false, want true")
				}
			},
		},
		{
			name: "keeps explicit values",
			cfg: BreakerConfig{
				Name:        "payments",
				Timeout:     2 * time.Second,
				Interval:    3 * time.Second,
				MaxRequests: 7,
				ReadyToTrip: customReadyToTrip,
			},
			assert: func(t *testing.T, got BreakerConfig) {
				t.Helper()
				if got.Timeout != 2*time.Second {
					t.Fatalf("Timeout = %s, want %s", got.Timeout, 2*time.Second)
				}
				if got.Interval != 3*time.Second {
					t.Fatalf("Interval = %s, want %s", got.Interval, 3*time.Second)
				}
				if got.MaxRequests != 7 {
					t.Fatalf("MaxRequests = %d, want 7", got.MaxRequests)
				}
				if !got.ReadyToTrip(gobreaker.Counts{ConsecutiveFailures: 2}) {
					t.Fatal("custom ReadyToTrip() = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeConfig(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("normalizeConfig() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeConfig() error = %v, want nil", err)
			}
			tt.assert(t, got)
		})
	}
}

func TestRegisterBreaker(t *testing.T) {
	resetBreakers(t)

	b, err := RegisterBreaker(BreakerConfig{
		Name:   "auth",
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("RegisterBreaker() error = %v, want nil", err)
	}
	if b == nil {
		t.Fatal("RegisterBreaker() breaker = nil")
	}
	if b.Name() != "auth" {
		t.Fatalf("breaker Name() = %q, want %q", b.Name(), "auth")
	}

	got, ok := GetBreaker("auth")
	if !ok {
		t.Fatal("GetBreaker() ok = false, want true")
	}
	if got != b {
		t.Fatal("GetBreaker() returned different breaker")
	}
}

func TestRegisterBreakerRejectsInvalidAndDuplicate(t *testing.T) {
	resetBreakers(t)

	if b, err := RegisterBreaker(BreakerConfig{}); err == nil {
		t.Fatalf("RegisterBreaker() error = nil, want error; breaker = %#v", b)
	}

	if _, err := RegisterBreaker(BreakerConfig{Name: "auth"}); err != nil {
		t.Fatalf("RegisterBreaker() first error = %v, want nil", err)
	}
	if b, err := RegisterBreaker(BreakerConfig{Name: "auth"}); err == nil {
		t.Fatalf("RegisterBreaker() duplicate error = nil, want error; breaker = %#v", b)
	} else if !strings.Contains(err.Error(), "breaker already registered: auth") {
		t.Fatalf("duplicate error = %q, want registered message", err.Error())
	}
}

func TestGetBreakerMissing(t *testing.T) {
	resetBreakers(t)

	b, ok := GetBreaker("missing")
	if ok {
		t.Fatal("GetBreaker() ok = true, want false")
	}
	if b != nil {
		t.Fatal("GetBreaker() breaker != nil, want nil")
	}
}

func TestMustGetBreaker(t *testing.T) {
	resetBreakers(t)

	registered, err := RegisterBreaker(BreakerConfig{Name: "auth"})
	if err != nil {
		t.Fatalf("RegisterBreaker() error = %v", err)
	}

	got := MustGetBreaker("auth")
	if got != registered {
		t.Fatal("MustGetBreaker() returned different breaker")
	}
}

func TestMustGetBreakerPanicsWhenMissing(t *testing.T) {
	resetBreakers(t)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("MustGetBreaker() did not panic")
		}
		if got := r.(string); got != "breaker not found: missing" {
			t.Fatalf("panic = %q, want %q", got, "breaker not found: missing")
		}
	}()

	MustGetBreaker("missing")
}

func TestRegisteredBreakerUsesReadyToTrip(t *testing.T) {
	resetBreakers(t)

	b, err := RegisterBreaker(BreakerConfig{
		Name: "auth",
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})
	if err != nil {
		t.Fatalf("RegisterBreaker() error = %v", err)
	}

	_, err = b.Execute(func() (interface{}, error) {
		return nil, errors.New("dependency failed")
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want dependency error")
	}
	if b.State() != gobreaker.StateOpen {
		t.Fatalf("State() = %s, want %s", b.State(), gobreaker.StateOpen)
	}
}

func resetBreakers(t *testing.T) {
	t.Helper()

	mu.Lock()
	breakers = make(map[string]*gobreaker.CircuitBreaker)
	mu.Unlock()
}
