# Contributing

Thank you for improving Agent Gateway Go.

## Workflow

1. Open an issue describing the behavior and operational impact.
2. Create a focused branch and keep changes small.
3. Add tests for successful, failure, timeout, and concurrency paths where applicable.
4. Run `make test`, `make race`, `make vet`, and `make benchmark`.
5. Update the architecture, threat model, or benchmark documentation when an invariant changes.

## Design principles

- Prefer the standard library unless a dependency has a clear operational benefit.
- Bound input, concurrency, time, memory, and label cardinality.
- Never log prompts, responses, tool arguments, credentials, or headers.
- Keep the offline benchmark deterministic.
- Make errors useful to operators but safe for callers.

## Pull requests

Explain the problem, the chosen tradeoff, how it was tested, and any security or compatibility impact. CI must pass. New exported APIs should include Go documentation comments.
