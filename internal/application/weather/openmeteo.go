package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

type HourlyForecast struct {
	Time            time.Time `json:"time"`
	Temperature     float64   `json:"temperature"`
	RainProbability int       `json:"rain_probability"`
	DirectRadiation float64   `json:"direct_radiation"` // W/m^2
}

type WeatherReport struct {
	CurrentTemp        float64          `json:"current_temp"`
	CurrentWeatherCode int              `json:"current_weather_code"`
	Humidity           int              `json:"humidity"`
	WindSpeed          float64          `json:"wind_speed"`
	Hourly             []HourlyForecast `json:"hourly"`
	FetchedAt          time.Time        `json:"fetched_at"`
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type openMeteoResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		Windspeed   float64 `json:"windspeed"`
		Weathercode int     `json:"weathercode"`
	} `json:"current_weather"`
	Hourly struct {
		Time              []string  `json:"time"`
		Temperature2m     []float64 `json:"temperature_2m"`
		PrecipitationProb []int     `json:"precipitation_probability"`
		DirectRadiation   []float64 `json:"direct_normal_irradiance"`
	} `json:"hourly"`
}

// FetchForecast fetches live weather and irradiance from Open-Meteo or produces fallback simulation.
func (c *Client) FetchForecast(ctx context.Context, lat, lon float64) (*WeatherReport, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current_weather=true&hourly=temperature_2m,precipitation_probability,direct_normal_irradiance&forecast_days=2&timezone=auto", lat, lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err == nil {
		resp, err := c.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var om openMeteoResponse
			if err := json.NewDecoder(resp.Body).Decode(&om); err == nil {
				return c.mapResponse(&om), nil
			}
		}
	}

	// Fallback deterministic simulation based on coordinates and current time
	return c.fallbackForecast(lat, lon), nil
}

func (c *Client) mapResponse(om *openMeteoResponse) *WeatherReport {
	report := &WeatherReport{
		CurrentTemp:        om.CurrentWeather.Temperature,
		CurrentWeatherCode: om.CurrentWeather.Weathercode,
		WindSpeed:          om.CurrentWeather.Windspeed,
		Humidity:           65,
		Hourly:             make([]HourlyForecast, 0, len(om.Hourly.Time)),
		FetchedAt:          time.Now().UTC(),
	}

	for i, tStr := range om.Hourly.Time {
		t, _ := time.Parse("2006-01-02T15:04", tStr)
		temp := 18.0
		if i < len(om.Hourly.Temperature2m) {
			temp = om.Hourly.Temperature2m[i]
		}
		rain := 0
		if i < len(om.Hourly.PrecipitationProb) {
			rain = om.Hourly.PrecipitationProb[i]
		}
		rad := 0.0
		if i < len(om.Hourly.DirectRadiation) {
			rad = om.Hourly.DirectRadiation[i]
		}

		report.Hourly = append(report.Hourly, HourlyForecast{
			Time:            t,
			Temperature:     temp,
			RainProbability: rain,
			DirectRadiation: rad,
		})
	}

	return report
}

func (c *Client) fallbackForecast(lat, lon float64) *WeatherReport {
	now := time.Now().UTC().Truncate(time.Hour)
	// Modulate baseline temperature based on latitude (cooler at high latitudes)
	baseTemp := 25.0 - (math.Abs(lat) * 0.2)
	if baseTemp < 5.0 {
		baseTemp = 5.0
	}
	// Solar peak radiation adjusted by latitude
	maxRad := math.Max(600.0, 950.0-math.Abs(lat)*4.0)

	report := &WeatherReport{
		CurrentTemp:        math.Round(baseTemp*10) / 10,
		CurrentWeatherCode: 1, // Mainly clear
		Humidity:           55,
		WindSpeed:          12.0 + math.Mod(math.Abs(lon), 5.0),
		Hourly:             make([]HourlyForecast, 0, 48),
		FetchedAt:          now,
	}

	for i := 0; i < 48; i++ {
		t := now.Add(time.Duration(i) * time.Hour)
		hour := t.Hour()
		// Solar bell curve between 06:00 and 19:00 peaking at 13:00
		rad := 0.0
		if hour >= 6 && hour <= 19 {
			rad = maxRad * math.Sin(float64(hour-6)/13.0*math.Pi)
		}
		report.Hourly = append(report.Hourly, HourlyForecast{
			Time:            t,
			Temperature:     baseTemp - 3.0 + 6.0*math.Sin(float64(hour-4)/24.0*2*math.Pi),
			RainProbability: 10,
			DirectRadiation: math.Max(0, rad),
		})
	}

	return report
}
