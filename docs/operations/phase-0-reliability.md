# Phase 0 – Reliability and fail-safe behavior

This document records the production-safety gates introduced by the Phase 0 reliability branch.

## Automation runtime
- State triggers evaluate both `from` and `to` using the previous persisted/observed state.
- `for_seconds` is implemented for state and numeric-state triggers. A delayed trigger is cancelled if the state leaves its target condition.
- State observations are seeded from the entity repository after startup, so `from` filters can work on the first event after a restart.
- Numeric-state triggers fire when entering the configured range.
- Sunrise and sunset triggers use the configured home coordinates and local timezone. They do not fire when the solar calculation reports no distinct event (polar day/night).
- Geofence entry/exit triggers use the transition states `just_arrived` and `just_left`, avoiding repeated firing on steady `home`/`away` updates.
- Unsupported combinations such as `for_seconds` on a time/event trigger are rejected during validation.

## Energy optimizer
- The generated sample curve is marked `Source=synthetic_estimate` and `IsEstimate=true`.
- Estimated or stale schedules never turn loads on.
- Automatic control accepts only a current, complete 24-hour schedule with an explicit source.
- `MaxPriceKWh` is enforced when creating the schedule and again at actuation time. Zero is a valid free-only ceiling; `-1` explicitly disables the ceiling. Non-finite values and other negative values normalize to zero.
- The built-in preview curve uses the system's local timezone for Italian F1/F2/F3 bands.
- Cost and savings estimates require a configured `PowerKW`; without it, the amounts remain zero rather than assuming an arbitrary 1 kW load.
- When a previously optimizer-activated load loses its trusted schedule, Domus attempts to turn it off. If event dispatch fails, the internal active state is retained so the next evaluation retries.

A real provider for GME/PUN, Nord Pool or Octopus still needs to be implemented and connected to `SetTariffSchedule`. Until then the generated curve is for preview only.

## Camera inference
- Deterministic/hash-generated face embeddings and canned object/plate detections have been removed.
- Coral, Hailo and CPU descriptors report unavailable until a real inference runtime and model are loaded.
- Inference methods return `ErrInferenceModelNotConfigured`; NVR consumers must treat this as unavailable, not as a negative detection.

## Matter and HomeKit
- Matter QR/manual-code paths no longer return fabricated commissioning payloads or synthetic nodes. The adapter refuses to start/commission until a real controller, secure sessions, fabric persistence and cluster transport are connected. The HTTP commissioning endpoint returns `503` for valid codes while the controller is unavailable.
- HomeKit no longer claims to start a bridge without an HAP server, mDNS advertising and persistent pairing store. Its default setup code is generated instead of using a repository-wide static PIN. The HTTP pairing endpoint returns `503` and does not disclose setup credentials until the bridge is implemented.
- Entity mapping helpers remain testable, but neither integration should be described as production-ready.

## Scripts and scenes
- Script `queued` mode now serializes executions per script through a bounded queue; parallel mode remains separately bounded.
- Scene dispatch errors are propagated. `SceneActivated` is emitted only after every command request has been accepted by the event bus; this is not a physical-state acknowledgement.

## Validation
CI runs the repository's Go tests/vet, frontend tests/build, installer, Docker, Android, iOS and OpenDisplay bridge checks. Passing CI does not replace hardware-in-the-loop validation for radios, thermostats, cameras, Matter or HomeKit.
