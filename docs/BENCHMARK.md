# Benchmark methodology

## Purpose

The benchmark answers a narrow, objective question: did an agent choose the exact expected function with the exact expected JSON arguments, and did it abstain when no call was warranted?

It is not an LLM-as-judge benchmark. A test either satisfies a deterministic predicate or it does not.

## Suite schema

Each case has one or more steps. A step provides a prompt, tool definitions, and either:

- `expected_call`, containing a function name and JSON object arguments, or
- `abstain: true`, requiring zero tool calls.

Argument object key order does not matter. Names, values, types, array order, and nested structure do. Additional arguments fail the exact match.

## Metrics

| Metric | Definition |
|---|---|
| Exact accuracy | Correct expected-call steps / all expected-call steps |
| Abstention accuracy | Steps with zero calls / all abstention steps |
| Multi-step success | Multi-step cases where every step passes / all multi-step cases |
| Errors | Provider or gateway failures observed while running cases |
| p50 / p95 | Percentiles of end-to-end case duration |

Percentiles use the nearest lower ranked observed value. Small suites are intentionally easy to inspect; larger suites should add bootstrap confidence intervals.

## Regression gates

Defaults:

```text
exact accuracy >= 0.95
abstention accuracy >= 0.95
multi-step success >= 0.90
p95 case latency <= 250ms
```

The benchmark exits with status 1 when a gate fails and status 2 for invalid input. CI uses the deterministic mock suite so regressions are repeatable.

## Failure injection

`-fail-every N` causes the primary mock to fail every Nth call. A backup provider should preserve benchmark accuracy while metrics and error paths exercise failover. `-quality N` uses a stable prompt hash to make a repeatable percentage of tool selections incorrect.

## Interpreting results

Mock results validate gateway behavior and scoring implementation, not real-world model intelligence. For a real provider:

1. Create a versioned, held-out suite with representative tools.
2. Run enough cases to quantify uncertainty.
3. Separate correctness, safety, latency, and cost gates.
4. Record model and prompt-template versions.
5. Review failed cases, not only aggregate scores.
6. Do not send sensitive production prompts to third-party evaluators.
