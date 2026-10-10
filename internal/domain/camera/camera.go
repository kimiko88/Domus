package camera

import (
	"time"
)

type StreamType string

const (
	StreamTypeWebRTC StreamType = "webrtc"
	StreamTypeHLS    StreamType = "hls"
	StreamTypeRTSP   StreamType = "rtsp"
)

type Camera struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	AreaID           *string    `json:"area_id,omitempty"`
	StreamURL        string     `json:"stream_url"`
	SnapshotURL      string     `json:"snapshot_url,omitempty"`
	StreamType       StreamType `json:"stream_type"`
	DetectionEnabled bool       `json:"detection_enabled"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type DetectionEvent struct {
	ID           string    `json:"id"`
	CameraID     string    `json:"camera_id"`
	Label        string    `json:"label"` // e.g. "person", "vehicle", "animal", "package"
	Score        float64   `json:"score"` // e.g. 0.85
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	DetectedAt   time.Time `json:"detected_at"`
}

type WebRTCSession struct {
	CameraID  string `json:"camera_id"`
	OfferSDP  string `json:"offer_sdp"`
	AnswerSDP string `json:"answer_sdp"`
}
