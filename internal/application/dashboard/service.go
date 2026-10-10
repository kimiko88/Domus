package dashboard

import (
	"context"
	"strings"

	"github.com/domus-hub/domus/internal/domain/dashboard"
	"github.com/google/uuid"
)

// Service manages custom widget dashboards and kiosk configurations.
type Service struct {
	repo dashboard.Repository
}

// NewService constructs a dashboard service.
func NewService(repo dashboard.Repository) *Service {
	return &Service{repo: repo}
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
