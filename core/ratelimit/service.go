package ratelimit

import (
    "context"
)

// Service provides rate limit checks
type Service struct {
    storage Storage
    config  Config
}

// NewService constructs a rate limit service
func NewService(storage Storage, cfg Config) *Service {
    if !cfg.Enabled {
        // ensure a default rule even if disabled; callers can skip checks when disabled
        cfg.DefaultRule = defaultConfig().DefaultRule
    }
    // If default rule not provided, fill it but preserve custom Rules
    if cfg.DefaultRule.Window == 0 || cfg.DefaultRule.Max <= 0 {
        cfg.DefaultRule = defaultConfig().DefaultRule
    }
    return &Service{storage: storage, config: cfg}
}

// CheckLimit returns true if within limit; false if exceeded
func (s *Service) CheckLimit(ctx context.Context, key string, rule Rule) (bool, error) {
    if !s.config.Enabled {
        return true, nil
    }
    if rule.Window == 0 || rule.Max <= 0 {
        rule = s.config.DefaultRule
    }
    count, err := s.storage.Increment(ctx, key, rule.Window)
    if err != nil {
        return false, err
    }
    return count <= rule.Max, nil
}

// CheckLimitForPath uses a configured per-path rule if present, otherwise DefaultRule
func (s *Service) CheckLimitForPath(ctx context.Context, key, path string) (bool, error) {
    rule, ok := s.config.Rules[path]
    if !ok {
        rule = s.config.DefaultRule
    }
    return s.CheckLimit(ctx, key, rule)
}