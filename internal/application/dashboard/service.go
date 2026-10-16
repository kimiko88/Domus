package dashboard

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	"github.com/domus-hub/domus/internal/domain/dashboard"
	"github.com/google/uuid"
)

// Service manages custom widget dashboards and kiosk configurations.
type Service struct {
	repo      dashboard.Repository
	secretKey []byte
}

// NewService constructs a dashboard service.
func NewService(repo dashboard.Repository) *Service {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	return &Service{
		repo:      repo,
		secretKey: key,
	}
}

// WithSecretKey configures a deterministic secret key for share token signing.
func (s *Service) WithSecretKey(key []byte) *Service {
	if len(key) > 0 {
		s.secretKey = key
	}
	return s
}

// Create validates and saves a new dashboard.
func (s *Service) Create(ctx context.Context, d *dashboard.Dashboard) error {
	if strings.TrimSpace(d.Title) == "" {
		return dashboard.ErrInvalidTitle
	}
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	if d.Icon == "" {
		d.Icon = "layout-dashboard"
	}
	return s.repo.Create(ctx, d)
}

// GetByID returns a dashboard by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*dashboard.Dashboard, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns all configured dashboards.
func (s *Service) List(ctx context.Context) ([]*dashboard.Dashboard, error) {
	return s.repo.List(ctx)
}

// Update updates an existing dashboard.
func (s *Service) Update(ctx context.Context, d *dashboard.Dashboard) error {
	if strings.TrimSpace(d.Title) == "" {
		return dashboard.ErrInvalidTitle
	}
	return s.repo.Update(ctx, d)
}

// Delete removes a dashboard.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// SetDefault designates a dashboard as the default homepage.
func (s *Service) SetDefault(ctx context.Context, id string) error {
	return s.repo.SetDefault(ctx, id)
}

// CreateShareToken generates a signed temporary share token for guest or iframe display.
func (s *Service) CreateShareToken(ctx context.Context, id string, duration time.Duration, readOnly bool) (string, *dashboard.ShareTokenPayload, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", nil, err
	}

	if duration <= 0 {
		duration = 24 * time.Hour
	}

	now := time.Now().UTC()
	payload := dashboard.ShareTokenPayload{
		DashboardID: d.ID,
		Title:       d.Title,
		ReadOnly:    readOnly,
		CreatedAt:   now,
		ExpiresAt:   now.Add(duration),
	}

	token, err := dashboard.GenerateShareToken(payload, s.secretKey)
	if err != nil {
		return "", nil, err
	}

	return token, &payload, nil
}

// GetShared verifies the share token and returns the targeted dashboard.
func (s *Service) GetShared(ctx context.Context, token string) (*dashboard.Dashboard, *dashboard.ShareTokenPayload, error) {
	payload, err := dashboard.ValidateShareToken(token, s.secretKey)
	if err != nil {
		return nil, nil, err
	}

	d, err := s.repo.GetByID(ctx, payload.DashboardID)
	if err != nil {
		return nil, nil, err
	}

	return d, payload, nil
}
