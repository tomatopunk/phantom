# Phantom — Coding Standards (Skill Style)

All code in this project MUST follow these conventions.

## 1. Comments

- **Language**: Use **English** for all code comments.
- **Density**: Do not comment every line. Add concise English comments at **key points** only:
  - Non-obvious branches and protocol boundaries
  - Concurrency and synchronization
  - Error recovery and eBPF safety checks
- Keep comments short; prefer one line when possible.

## 2. Single Responsibility

- One function does **one thing**. Name it so that purpose is clear.
- Avoid combining parsing, validation, business logic, and formatting in a single function.
- Split long flows into small, focused helpers.

## 3. Naming

- **Intent**: A function name should make it obvious **why** it exists.
- **Pattern**: Prefer `verb + domain_object + [intent]`, e.g.:
  - `validateHookSource`, `buildTraceExecutionPlan`, `emitBreakpointHitEvent`
  - `parseBreakpointSpec`, `attachKprobeAtSymbol`, `readEventFromRingbuf`
- Use clear variable names; avoid single-letter names except in tiny scopes (e.g. loop index).

## 4. Structure and Cleanliness

- Limit function length and nesting; prefer early returns.
- Minimize shared mutable state; pass explicit parameters.
- Use consistent error wrapping (e.g. `fmt.Errorf("context: %w", err)`); avoid duplicated error messages.

## 5. Layer Boundaries

- **CLI**, **Session**, **Probe**, **Runtime**, and **MCP** must not depend on each other’s implementation details.
- Communicate across layers via well-defined interfaces (e.g. `SessionManager`, `ProbePlanner`, `EventStream`).

## 6. Quality Gates

- Before commit: run `go fmt`, `go vet`, and project static checks.
- Maintain minimal unit tests for core paths.
- In PR review: check naming intent, single-responsibility, and presence of key English comments.
