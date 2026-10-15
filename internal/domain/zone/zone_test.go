package zone_test

import (
	"testing"

	domzone "github.com/domus-hub/domus/internal/domain/zone"
)

func TestZoneContains(t *testing.T) {
	// Rome Colosseum coordinates
	colosseum := domzone.Zone{
		ID:        "zone_colosseum",
		Name:      "Colosseum",
		Latitude:  41.8902,
		Longitude: 12.4922,
		Radius:    200,
		Passive:   false,
	}

	// Point 50 meters away
	nearLat := 41.8905
	nearLon := 12.4925
	if !colosseum.Contains(nearLat, nearLon) {
		t.Errorf("expected near point to be inside 200m zone")
	}

	// Point 2km away (Vatican area)
	farLat := 41.9029
	farLon := 12.4534
	if colosseum.Contains(farLat, farLon) {
		t.Errorf("expected Vatican to be outside Colosseum zone")
	}
}

func TestHaversineDistance(t *testing.T) {
	// Milan (45.4642, 9.1900) to Rome (41.9028, 12.4964) is ~477 km
	dist := domzone.HaversineDistance(45.4642, 9.1900, 41.9028, 12.4964)
	if dist < 470000 || dist > 485000 {
		t.Errorf("expected Milan-Rome distance ~477km, got %f meters", dist)
	}
}
