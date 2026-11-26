# Phase 2 – Plugin runtime hardening

This phase closes the first runtime-hardening items in the integration architecture roadmap. It strengthens lifecycle safety and introduces an administrator-only API to update an instance's encrypted configuration. It does not add third-party code execution or claim OS-level isolation for native Go adapters.

## Supervisor lifecycle

- Lifecycle operations are serialized independently from the state lock: register, start, stop, dynamic start, unregister, restart and replacement cannot race one another.
- Starting and stopping are idempotent. A stopped supervisor can be started again without duplicate monitor loops.
- An adapter is not removed from the supervision tree until `Stop` succeeds. A failed stop remains visible as degraded and can be retried.
- If shutdown reports a failed stop, the next supervisor start retries stopping that adapter before invoking `Start` again. Health probes skip adapters whose stop failed.
- Invalid adapters and invalid breaker policies are rejected at registration time.
- Degraded health probes respect the configured health interval instead of being retried every monitor tick.
- Adapter replacement keeps the prior adapter registered; if a replacement fails to start, the supervisor stops the candidate and attempts to restore the original adapter.

## Instance configuration update

Administrators can update a plugin instance through:

```http
PUT /api/v1/plugins/instances/{id}/config
Content-Type: application/json

{
  "config": {
    "broker_url": "tcp://192.168.1.20:1883",
    "username": "domus",
    "password": "new-secret"
  }
}
```

The manager validates the new settings against the registered manifest, encrypts them before persistence, and replaces the runtime adapter without returning the submitted credentials. The previous encrypted configuration and adapter are retained when validation or replacement startup fails. If durable persistence fails after a successful replacement, the manager attempts to restore the prior adapter and leaves the configuration unchanged.

Disabled instances can also be reconfigured: the candidate is validated and released without starting background work, and only the encrypted configuration is persisted.

The API is registered inside the existing administrator-only `/api/v1/plugins` route group. Request fields are strict, trailing JSON is rejected, and body size is bounded.

## Regression tests added

- Supervisor Start/Stop idempotence and restart after a normal stop.
- Stop failure retains a registered adapter; explicit unregister retry succeeds after the failure is cleared.
- A failed stop during shutdown is retried before the next Start.
- Adapter replacement rollback restores the previous adapter when the candidate fails to start.
- Invalid adapter identities and breaker policies are rejected.
- Manager reconfiguration encrypts the new config, masks credentials in its public response and retains the old adapter/config after replacement failure.
- Enable failures do not persist an enabled state or leave an untracked adapter.
- HTTP reconfiguration validates input, persists encrypted values and never returns secret fields.

## Remaining validation

The test suite still needs hardware-in-the-loop checks for actual gateway and device transports. The Go plugin registry continues to load built-in factories compiled into Domus. Native Go plugins execute with the Domus process privileges; this phase does not introduce a community sandbox, WASI runtime or capability enforcement at the OS boundary.
