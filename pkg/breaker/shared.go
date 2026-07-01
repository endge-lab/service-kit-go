package breaker

import (
	"errors"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// BreakerConfig is breaker configuration
type BreakerConfig struct {
	Name        string                             // name of breaker
	Timeout     time.Duration                      // timeout
	ReadyToTrip func(counts gobreaker.Counts) bool // ready to trip
	MaxRequests uint32                             // max requests
	Interval    time.Duration                      // interval
	Logger      *zap.Logger                        // logger
}

// Internal stores all registered breakers
var (
	mu       sync.RWMutex
	breakers = make(map[string]*gobreaker.CircuitBreaker)
)

// RegisterBreaker registers new breaker
func RegisterBreaker(cfg BreakerConfig) (*gobreaker.CircuitBreaker, error) {

	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	settings := gobreaker.Settings{
		Name:        cfg.Name,
		Timeout:     cfg.Timeout,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		ReadyToTrip: cfg.ReadyToTrip,
		OnStateChange: func(name string, from, to gobreaker.State) {
			if cfg.Logger != nil {
				cfg.Logger.Warn("circuit breaker state change",
					zap.String("name", name),
					zap.String("from", from.String()),
					zap.String("to", to.String()))
			}
		},
	}

	breaker := gobreaker.NewCircuitBreaker(settings)

	mu.Lock()
	defer mu.Unlock()

	if _, exists := breakers[cfg.Name]; exists {
		return nil, errors.New("breaker already registered: " + cfg.Name)
	}

	breakers[cfg.Name] = breaker

	return breaker, nil
}

// GetBreaker returns breaker by name
func GetBreaker(name string) (*gobreaker.CircuitBreaker, bool) {
	mu.RLock()
	defer mu.RUnlock()
	b, ok := breakers[name]

	return b, ok
}

func normalizeConfig(cfg BreakerConfig) (BreakerConfig, error) {
	if cfg.Name == "" {
		return BreakerConfig{}, errors.New("breaker name is required")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	if cfg.Interval <= 0 {
		cfg.Interval = 60 * time.Second
	}

	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 1
	}

	if cfg.ReadyToTrip == nil {
		cfg.ReadyToTrip = func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		}
	}

	return cfg, nil
}

// MustGetBreaker analogous to GetBreaker but panics if not found
func MustGetBreaker(name string) *gobreaker.CircuitBreaker {
	b, ok := GetBreaker(name)
	if !ok {
		panic("breaker not found: " + name)
	}

	return b
}
