// Package mesh provides RF mesh topology, link quality and direct binding management.
package mesh

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrBindingNotFound = errors.New("binding: direct link not found")
)

type ZigbeeBinding struct {
	ID               string    `json:"id"`
	SourceDeviceID   string    `json:"source_device_id"`
	SourceDeviceName string    `json:"source_device_name"`
	TargetDeviceID   string    `json:"target_device_id"`
	TargetDeviceName string    `json:"target_device_name"`
	Clusters         []string  `json:"clusters"`
	BoundAt          time.Time `json:"bound_at"`
}

type BindingService struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewBindingService(db *sql.DB) *BindingService {
	return &BindingService{db: db}
}

// CreateBinding records a direct hardware binding between a Zigbee switch and lamp/group.
func (s *BindingService) CreateBinding(ctx context.Context, sourceID, sourceName, targetID, targetName string, clusters []string) (*ZigbeeBinding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(sourceID) == "" || strings.TrimSpace(targetID) == "" {
		return nil, fmt.Errorf("binding: source and target device IDs are required")
	}

	if len(clusters) == 0 {
		clusters = []string{"genOnOff"}
	}

	clustersJSON, err := json.Marshal(clusters)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	b := &ZigbeeBinding{
		ID:               uuid.NewString(),
		SourceDeviceID:   sourceID,
		SourceDeviceName: sourceName,
		TargetDeviceID:   targetID,
		TargetDeviceName: targetName,
		Clusters:         clusters,
		BoundAt:          now,
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO zigbee_bindings (
			id, source_device_id, source_device_name, target_device_id, target_device_name,
			clusters_json, bound_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, b.ID, b.SourceDeviceID, b.SourceDeviceName, b.TargetDeviceID, b.TargetDeviceName, string(clustersJSON), b.BoundAt)
	if err != nil {
		return nil, fmt.Errorf("binding: failed to insert: %w", err)
	}

	return b, nil
}

// RemoveBinding deletes a binding.
func (s *BindingService) RemoveBinding(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, "DELETE FROM zigbee_bindings WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrBindingNotFound
	}
	return nil
}

// ListBindings returns all direct bindings.
func (s *BindingService) ListBindings(ctx context.Context) ([]ZigbeeBinding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_device_id, source_device_name, target_device_id, target_device_name, clusters_json, bound_at
		FROM zigbee_bindings
		ORDER BY bound_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ZigbeeBinding
	for rows.Next() {
		var b ZigbeeBinding
		var clustersJSON string

		if err := rows.Scan(
			&b.ID, &b.SourceDeviceID, &b.SourceDeviceName, &b.TargetDeviceID, &b.TargetDeviceName,
			&clustersJSON, &b.BoundAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal([]byte(clustersJSON), &b.Clusters)
		list = append(list, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if list == nil {
		list = []ZigbeeBinding{}
	}
	return list, nil
}
