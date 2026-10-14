package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/domus-hub/domus/internal/domain/media"
	"github.com/google/uuid"
)

// MediaRepository implements media.Repository for SQLite.
type MediaRepository struct {
	db *sql.DB
}

// NewMediaRepository creates a new SQLite-backed MediaRepository.
func NewMediaRepository(db *sql.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

// SavePlayer inserts or updates a media player.
func (r *MediaRepository) SavePlayer(ctx context.Context, p *media.MediaPlayer) error {
	if p.ID == "" {
		p.ID = "media_" + uuid.NewString()[:8]
	}
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}

	mutedInt := 0
	if p.Muted {
		mutedInt = 1
	}

	query := `
		INSERT INTO media_players (id, name, device_id, area_id, platform, state, volume, muted, current_title, current_artist, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			device_id = excluded.device_id,
			area_id = excluded.area_id,
			platform = excluded.platform,
			state = excluded.state,
			volume = excluded.volume,
			muted = excluded.muted,
			current_title = excluded.current_title,
			current_artist = excluded.current_artist
	`
	_, err := r.db.ExecContext(
		ctx, query,
		p.ID,
		p.Name,
		p.DeviceID,
		p.AreaID,
		string(p.Platform),
		string(p.State),
		p.Volume,
		mutedInt,
		p.CurrentTitle,
		p.CurrentArtist,
		p.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("sqlite: failed to save media player: %w", err)
	}
	return nil
}

// GetPlayer retrieves a media player by ID.
func (r *MediaRepository) GetPlayer(ctx context.Context, id string) (*media.MediaPlayer, error) {
	query := `
		SELECT id, name, device_id, area_id, platform, state, volume, muted, current_title, current_artist, created_at
		FROM media_players
		WHERE id = ?
	`
	var (
		p             media.MediaPlayer
		deviceID      sql.NullString
		areaID        sql.NullString
		platform      string
		state         string
		mutedInt      int
		currentTitle  sql.NullString
		currentArtist sql.NullString
		createdStr    string
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID,
		&p.Name,
		&deviceID,
		&areaID,
		&platform,
		&state,
		&p.Volume,
		&mutedInt,
		&currentTitle,
		&currentArtist,
		&createdStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("media player not found: %s", id)
		}
		return nil, fmt.Errorf("sqlite: failed to get media player: %w", err)
	}

	p.Platform = media.Platform(platform)
	p.State = media.PlayerState(state)
	p.Muted = mutedInt == 1
	p.DeviceID = deviceID.String
	if areaID.Valid {
		p.AreaID = &areaID.String
	}
	p.CurrentTitle = currentTitle.String
	p.CurrentArtist = currentArtist.String
	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		p.CreatedAt = t
	}

	return &p, nil
}

// ListPlayers returns all registered media players.
func (r *MediaRepository) ListPlayers(ctx context.Context) ([]*media.MediaPlayer, error) {
	query := `
		SELECT id, name, device_id, area_id, platform, state, volume, muted, current_title, current_artist, created_at
		FROM media_players
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to list media players: %w", err)
	}
	defer rows.Close()

	var players []*media.MediaPlayer
	for rows.Next() {
		var (
			p             media.MediaPlayer
			deviceID      sql.NullString
			areaID        sql.NullString
			platform      string
			state         string
			mutedInt      int
			currentTitle  sql.NullString
			currentArtist sql.NullString
			createdStr    string
		)
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&deviceID,
			&areaID,
			&platform,
			&state,
			&p.Volume,
			&mutedInt,
			&currentTitle,
			&currentArtist,
			&createdStr,
		); err != nil {
			return nil, fmt.Errorf("sqlite: scan media player error: %w", err)
		}

		p.Platform = media.Platform(platform)
		p.State = media.PlayerState(state)
		p.Muted = mutedInt == 1
		p.DeviceID = deviceID.String
		if areaID.Valid {
			p.AreaID = &areaID.String
		}
		p.CurrentTitle = currentTitle.String
		p.CurrentArtist = currentArtist.String
		if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
			p.CreatedAt = t
		}
		players = append(players, &p)
	}

	if players == nil {
		players = []*media.MediaPlayer{}
	}
	return players, nil
}

// DeletePlayer removes a media player by ID.
func (r *MediaRepository) DeletePlayer(ctx context.Context, id string) error {
	query := `DELETE FROM media_players WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("sqlite: failed to delete media player: %w", err)
	}
	return nil
}
