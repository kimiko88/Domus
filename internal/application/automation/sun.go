// Package automation provides rule engine DSL, triggers, conditions, sun calculations, and dry-run execution.
package automation

import (
	"math"
	"time"
)

// SunPosition contains the astronomical position of the sun.
type SunPosition struct {
	Elevation float64   `json:"elevation"` // degrees above horizon (-90 to +90)
	Azimuth   float64   `json:"azimuth"`   // degrees from true north (0 to 360)
	IsDay     bool      `json:"is_day"`    // true if elevation > -0.833 degrees
	Sunrise   time.Time `json:"sunrise"`
	Sunset    time.Time `json:"sunset"`
	SolarNoon time.Time `json:"solar_noon"`
}

// SunEngine calculates local sun positions and solar events without external APIs.
type SunEngine struct {
	Latitude  float64
	Longitude float64
}

// NewSunEngine creates a sun calculation engine for given GPS coordinates.
func NewSunEngine(lat, lon float64) *SunEngine {
	return &SunEngine{Latitude: lat, Longitude: lon}
}

// Calculate returns the solar elevation, azimuth, sunrise, and sunset for time t.
func (e *SunEngine) Calculate(t time.Time) SunPosition {
	utc := t.UTC()
	latRad := e.Latitude * (math.Pi / 180.0)

	// Day of year and hour of day
	dayOfYear := float64(utc.YearDay())
	hourOfDay := float64(utc.Hour()) + float64(utc.Minute())/60.0 + float64(utc.Second())/3600.0

	// Fractional year (radians)
	gamma := (2.0 * math.Pi / 365.0) * (dayOfYear - 1.0 + (hourOfDay-12.0)/24.0)

	// Equation of time (minutes)
	eqtime := 229.18 * (0.000075 + 0.001868*math.Cos(gamma) - 0.032077*math.Sin(gamma) -
		0.014615*math.Cos(2.0*gamma) - 0.040849*math.Sin(2.0*gamma))

	// Solar declination (radians)
	decl := 0.006918 - 0.399912*math.Cos(gamma) + 0.070257*math.Sin(gamma) -
		0.006758*math.Cos(2.0*gamma) + 0.000907*math.Sin(2.0*gamma) -
		0.002697*math.Cos(3.0*gamma) + 0.00148*math.Sin(3.0*gamma)

	// True solar time (minutes)
	timeOffset := eqtime + 4.0*e.Longitude
	tst := hourOfDay*60.0 + timeOffset

	// Solar hour angle (degrees)
	haDeg := tst/4.0 - 180.0
	haRad := haDeg * (math.Pi / 180.0)

	// Solar zenith angle (radians)
	cosZenith := math.Sin(latRad)*math.Sin(decl) + math.Cos(latRad)*math.Cos(decl)*math.Cos(haRad)
	cosZenith = math.Max(-1.0, math.Min(1.0, cosZenith))
	zenith := math.Acos(cosZenith)

	// Elevation in degrees
	elevation := 90.0 - (zenith * (180.0 / math.Pi))

	// Solar azimuth angle (degrees from north)
	sinAzimuth := -(math.Sin(haRad) * math.Cos(decl)) / math.Sin(zenith)
	cosAzimuth := (math.Sin(latRad)*math.Cos(zenith) - math.Sin(decl)) / (math.Cos(latRad) * math.Sin(zenith))
	azimuthRad := math.Atan2(sinAzimuth, cosAzimuth)
	azimuthDeg := azimuthRad * (180.0 / math.Pi)
	if azimuthDeg < 0.0 {
		azimuthDeg += 360.0
	}

	// Sunrise and Sunset calculation (zenith = 90.833 degrees accounting for atmospheric refraction)
	const sunsetZenith = 90.833 * (math.Pi / 180.0)
	cosHourAngle := (math.Cos(sunsetZenith) - math.Sin(latRad)*math.Sin(decl)) / (math.Cos(latRad) * math.Cos(decl))

	midnight := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	noonMinutes := 720.0 - 4.0*e.Longitude - eqtime
	solarNoon := midnight.Add(time.Duration(noonMinutes * float64(time.Minute)))

	var sunrise, sunset time.Time
	if cosHourAngle >= -1.0 && cosHourAngle <= 1.0 {
		haSunsetDeg := math.Acos(cosHourAngle) * (180.0 / math.Pi)
		sunsetMinutes := 720.0 - 4.0*e.Longitude - eqtime + haSunsetDeg*4.0
		sunriseMinutes := 720.0 - 4.0*e.Longitude - eqtime - haSunsetDeg*4.0

		sunrise = midnight.Add(time.Duration(sunriseMinutes * float64(time.Minute)))
		sunset = midnight.Add(time.Duration(sunsetMinutes * float64(time.Minute)))
	} else if cosHourAngle < -1.0 {
		// Polar day
		sunrise = midnight
		sunset = midnight.Add(24 * time.Hour)
	} else {
		// Polar night
		sunrise = midnight
		sunset = midnight
	}

	return SunPosition{
		Elevation: math.Round(elevation*100) / 100,
		Azimuth:   math.Round(azimuthDeg*100) / 100,
		IsDay:     elevation > -0.833,
		Sunrise:   sunrise.In(t.Location()),
		Sunset:    sunset.In(t.Location()),
		SolarNoon: solarNoon.In(t.Location()),
	}
}
