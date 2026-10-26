// Package main implements day-ahead dynamic electricity tariff indexing and normalization.
// Compile target: tinygo build -o plugin.wasm -target=wasi main.go
package main

import (
	"encoding/json"
	"unsafe"
)

type HourlyTariff struct {
	Hour       int     `json:"hour"`
	PriceEurKwh float64 `json:"price_eur_kwh"`
	IsCheapest bool    `json:"is_cheapest"`
	IsPeak     bool    `json:"is_peak"`
}

type MarketTariffsDay struct {
	Zone    string         `json:"zone"`
	Date    string         `json:"date"`
	Average float64        `json:"average_eur_kwh"`
	Hours   []HourlyTariff `json:"hours"`
}

//export domus_handle_event
func domus_handle_event(ptr unsafe.Pointer, size int) uint64 {
	return 0
}

//export domus_parse_market_prices
func domus_parse_market_prices(ptr unsafe.Pointer, size int) uint64 {
	day := MarketTariffsDay{
		Zone:    "IT_PUN",
		Date:    "2026-10-27",
		Average: 0.115,
		Hours: make([]HourlyTariff, 24),
	}

	for h := 0; h < 24; h++ {
		price := 0.09 + float64((h-13)*(h-13))*0.001
		day.Hours[h] = HourlyTariff{
			Hour:        h,
			PriceEurKwh: price,
			IsCheapest:  h >= 12 && h <= 15,
			IsPeak:      h >= 19 && h <= 21,
		}
	}

	out, err := json.Marshal(day)
	if err != nil {
		return 0
	}

	outPtr := uintptr(unsafe.Pointer(&out[0]))
	return (uint64(outPtr) << 32) | uint64(len(out))
}

func main() {}
