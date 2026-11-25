// Package weather_openmeteo adapts Open-Meteo to the typed Domus ForecastProvider
// contract and the supervised plugin lifecycle.
package weather_openmeteo

import (
	"context"
	"errors"
	"strings"
	"sync"

	appweather "github.com/domus-hub/domus/internal/application/weather"
)

const TypeID = "weather_openmeteo"

type Adapter struct {
	id string
	client *appweather.Client
	mu sync.RWMutex
	active bool
}

var _ appweather.ForecastProvider = (*Adapter)(nil)

func NewAdapter(id string) *Adapter {
	id = strings.TrimSpace(id)
	if id == "" { id = TypeID }
	return &Adapter{id: id, client: appweather.NewClient()}
}

func (a *Adapter) ID() string { return a.id }
func (a *Adapter) Type() string { return TypeID }

func (a *Adapter) Start(ctx context.Context) error {
	if err := ctx.Err(); err != nil { return err }
	a.mu.Lock()
	a.active = true
	a.mu.Unlock()
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	a.active = false
	a.mu.Unlock()
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	if err := ctx.Err(); err != nil { return err }
	a.mu.RLock()
	active := a.active
	a.mu.RUnlock()
	if !active { return errors.New("weather_openmeteo: provider is stopped") }
	return nil
}

func (a *Adapter) FetchForecast(ctx context.Context, lat, lon float64) (*appweather.WeatherReport, error) {
	if err := a.HealthCheck(ctx); err != nil { return nil, err }
	return a.client.FetchForecast(ctx, lat, lon)
}
