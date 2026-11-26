# Phase 1 – Regression coverage and localization safeguards

This phase builds automated regression coverage around the reliability behavior introduced in Phase 0. It does not turn experimental integrations into production-ready features.

## Backend regression tests

- Numeric-state triggers fire only when a value enters the configured range, including attribute-based thresholds.
- Geofence triggers fire only on `just_arrived` and `just_left` transition events, not on stable `home` or `away` states.
- Flexible-load cost and savings are checked against an explicit average power in kW and a known 24-hour tariff schedule.
- Queued scripts reject executions when their configured bounded capacity is exhausted.

## Frontend regression tests

- The energy dashboard displays the synthetic-tariff warning and never invents cost or savings before power is configured.
- A valid power change is persisted using the flexible-load API contract and the values returned by the server are rendered.
- A failed save reloads the server value instead of leaving an optimistic value in the interface.

## Localization safeguards

Domus currently ships Italian, English, French, Spanish, German and Russian catalogs. Existing recursive key-parity tests remain in place. The localization suite now also checks that all four energy safety/configuration strings added in Phase 0 are non-empty and translated in every supported language, rather than silently falling back to English.

## CI validation

The existing CI jobs already execute `go test ./...` and `npm run test:run`; the new backend and frontend cases are therefore release gates without adding a second, overlapping test workflow. The frontend build remains part of CI to catch TypeScript and Vue template regressions.

Hardware-in-the-loop validation is still required for radio transports, real thermostats, cameras, Matter, HomeKit and physical load actuation. These unit and UI tests cannot verify external hardware behavior.
