# Monitoring AI usage

The normal loopback `/metrics` endpoint (default `127.0.0.1:9101`) exports AI
transport metrics when AI is enabled. Restart Arabica after updating the binary.
Import `grafana/arabica-prometheus.json` for usage, latency, failures and reporting
coverage panels. Series appear after the first upstream call.

| Metric | Meaning |
| --- | --- |
| `arabica_ai_requests_total{provider,model,outcome,status}` | Every attempted upstream call, including failures. Status is `0` when no HTTP response arrived; outcome is `success` or a bounded transport failure category. |
| `arabica_ai_request_duration_seconds{provider,model}` | Histogram including response reading/decoding, on successes and failures. |
| `arabica_ai_tokens_total{provider,model,type}` | Reported `input`, `output`, and `cached_input` tokens. Input **includes** cached input; output includes reasoning tokens. |
| `arabica_ai_usage_reports_total{provider,model,state}` | `reported`, `missing`, `invalid`, or `unavailable` (no decodable 2xx body). |

Provider/model labels come only from server configuration. There are no DIDs,
chat IDs, prompts, replies, URLs, credentials or provider-supplied model labels.
Each provider call counts independently, including tool round trips and calls
from quota-exempt accounts. Local validation/quota rejection is not an upstream
call. These metrics do not change quota enforcement or record persistence.

Usage comes from OpenAI-compatible `usage.prompt_tokens`, `completion_tokens`
and optional `prompt_tokens_details.cached_tokens`. Missing or malformed usage
never fails an otherwise valid completion and is **not** treated as zero usage.
Valid usage on an invalid assistant response is still counted. Timeouts, dropped
connections, non-2xx responses and oversized/undecodable bodies may have incurred
charges without reporting tokens. Consult provider billing for authoritative
spend. This is monitoring, not a billing ledger or hard budget enforcement.

## Alerts

`grafana/ai-alerts.yml` supplies Prometheus rules for provider errors, rate limits,
incomplete usage reporting, high token volume and slow responses. `nix run
.#monitoring` loads them into the local Prometheus. For production, add the file
to your Prometheus `rule_files` and reload the server. Tune thresholds (especially
the example one-million-tokens/hour warning) to your traffic and budget.

**Notification delivery needs an operator-configured Alertmanager or Grafana
contact point.** The local stack evaluates alerts but does not send notifications.
Include deployment/cluster labels in grouping when using a shared multi-cluster
Prometheus. Token metrics cannot detect all provider-side charges or missing
scrapes; retain the provider's own spending alerts and hard limits.

Check the rules with `promtool check rules grafana/ai-alerts.yml` and their
behavior with `promtool test rules grafana/ai-alerts.test.yml`.

## Estimated dollars

Prices are deliberately not baked into the application. They vary by model,
provider, caching and contract. For one configured provider/model, use this
PromQL expression after replacing `PROVIDER`, `MODEL`, and the three price
placeholders with current USD prices **per million tokens**:

```promql
(
  sum(increase(arabica_ai_tokens_total{provider="PROVIDER",model="MODEL",type="input"}[24h]))
  - sum(increase(arabica_ai_tokens_total{provider="PROVIDER",model="MODEL",type="cached_input"}[24h]))
) * INPUT_USD_PER_MILLION / 1e6
+ sum(increase(arabica_ai_tokens_total{provider="PROVIDER",model="MODEL",type="cached_input"}[24h]))
  * CACHED_INPUT_USD_PER_MILLION / 1e6
+ sum(increase(arabica_ai_tokens_total{provider="PROVIDER",model="MODEL",type="output"}[24h]))
  * OUTPUT_USD_PER_MILLION / 1e6
```

Create a recording rule per model, then sum those estimates for an aggregate
spend alert. Use a rolling hour for burn-rate alerts and rolling 24 hours for
budget warnings; neither is a calendar-day invoice. Never sum `input` and
`cached_input` as independent charges. Prices must be updated when the configured
model or contract changes. Estimates are lower bounds when usage is unavailable.
Prometheus owns retention; counters reset on process restart and `increase`
handles observed resets, but unsampled calls can still be lost.
