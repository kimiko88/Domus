# ADR-004: Structured JSON DSL for Automations and Scenes

## Status
Accepted

## Date
2026-10-03

## Context
Home automation systems require a mechanism for users to define event triggers, condition predicates, and automated action sequences. Common industry approaches include:
1. Full scripting languages (Python, Lua, JavaScript) evaluated dynamically.
2. Template-based declarative YAML with embedded Jinja expressions.
3. Structured, strongly-typed JSON Schema DSL.

On edge devices like Raspberry Pi 4, embedding dynamic scripting engines (like embedded V8 or Python) introduces significant security attack surfaces (arbitrary code execution), memory overhead, and interpreter garbage collection spikes. Furthermore, YAML/Jinja engines suffer from fragile indentation and runtime evaluation exceptions that cannot be validated prior to persistence.

## Decision
We adopt a **Structured JSON DSL** for automations and scenes:
1. **Triggers**: Represented as strongly-typed JSON objects specifying platform (`state`, `numeric_state`, `time`, `event`), target entity, from/to states, or numeric thresholds (`above`, `below`).
2. **Conditions**: Declarative logic nodes supporting state equality, numeric comparisons, time windows, and recursive boolean compositions (`and`, `or`, `not`).
3. **Actions**: Explicit service dispatch (`service`, `entity_id`, `data`), delays (`delay_seconds`), or custom events.
4. **Execution Control**: Configurable per-rule modes (`single`, `restart`, `queued`, `parallel`) and cooldown throttling to guard against cascading feedback loops.

## Consequences
### Positive
- **Security**: Zero remote code execution risk; rules cannot execute arbitrary system calls or escape runtime bounds.
- **Predictability & Validation**: Rules are deterministically validated upon creation/update via JSON Schema and domain invariant checks.
- **Frontend Interoperability**: Direct representation in Vue 3 visual builders and JSON editors without intermediate transpilations.
- **Performance**: High-speed evaluation in compiled Go without runtime interpreter overhead.

### Negative
- Less expressive than arbitrary Turing-complete scripts for complex mathematical computations (mitigated by extensible future filter operators).
