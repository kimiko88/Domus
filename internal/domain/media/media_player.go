package media

import "time"

type Platform string

const (
	PlatformCast    Platform = "cast"
	PlatformAirPlay Platform = "airplay"
	PlatformDLNA    Platform = "dlna"
	PlatformLocal   Platform = "local"
)

type PlayerState string

const (
	StateIdle    PlayerState = "idle"
	StatePlaying PlayerState = "playing"
	StatePaused  PlayerState = "paused"
)

type MediaPlayer struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	DeviceID      string      `json:"device_id,omitempty"`
	AreaID        *string     `json:"area_id,omitempty"`
	Platform      Platform    `json:"platform"`
	State         PlayerState `json:"state"`
	Volume        float64     `json:"volume"` // 0.0 to 1.0
	Muted         bool        `json:"muted"`
	CurrentTitle  string      `json:"current_title,omitempty"`
	CurrentArtist string      `json:"current_artist,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

type BroadcastAnnouncement struct {
	AudioURL        string   `json:"audio_url"`
	Message         string   `json:"message"`
	Volume          float64  `json:"volume"`                      // e.g. 0.8
	TargetPlayerIDs []string `json:"target_player_ids,omitempty"` // empty for all
}
