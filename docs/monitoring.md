# Monitoring AI usage

The normal loopback `/metrics` endpoint (default `127.0.0.1:9101`) exports AI
transport metrics when AI is enabled. Restart Arabica after updating the binary.
Import `grafana/arabica-prometheus.json` for usage, latency, failures and
reporting coverage panels. Series appear after the first upstream call.

| Metric                                                     | Meaning                                                                                                                                                       |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `arabica_ai_requests_total{provider,model,outcome,status}` | Every attempted upstream call, including failures. Status is `0` when no HTTP response arrived; outcome is `success` or a bounded transport failure category. |
| `arabica_ai_request_duration_seconds{provider,model}`      | Histogram including response reading/decoding, on successes and failures.                                                                                     |
| `arabica_ai_tokens_total{provider,model,type}`             | Reported `input`, `output`, and `cached_input` tokens. Input **includes** cached input; output includes reasoning tokens.                                     |
| `arabica_ai_usage_reports_total{provider,model,state}`     | `reported`, `missing`, `invalid`, or `unavailable` (no decodable 2xx body).                                                                                   |
| `arabica_ai_request_log_errors_total`                      | Failed SQLite request-log writes. No DID labels.                                                                                                              |

Provider/model labels come only from server configuration. There are no DIDs,
chat IDs, prompts, replies, URLs, credentials or provider-supplied model labels.
Each provider call counts independently, including tool round trips and calls
from quota-exempt accounts. Local validation/quota rejection is not an upstream
call. These metrics do not change quota enforcement or record persistence.

Usage comes from OpenAI-compatible `usage.prompt_tokens`, `completion_tokens`
and optional `prompt_tokens_details.cached_tokens`. Missing or malformed usage
never fails an otherwise valid completion and is **not** treated as zero usage.
Valid usage on an invalid assistant response is still counted. Timeouts, dropped
connections, non-2xx responses and oversized/undecodable bodies may have
incurred charges without reporting tokens. Consult provider billing for
authoritative spend. This is monitoring, not a billing ledger or hard budget
enforcement.

## Per-DID request history

When AI is enabled, server startup creates `ai_request_log` in the shared SQLite
database and attaches it independently of daily quota enforcement. **Rows are
kept indefinitely.** There is no automatic deletion or retention setting yet;
start-time indexes support adding bounded retention later. Database and backup
sizes grow with call volume, so monitor disk space as usage grows. Quota cleanup
does not delete this history.

Each upstream call appends one row after completion, including failures,
cancellations and calls by administrators or other quota-exempt accounts:

| Columns                                                | Meaning                                                                                                                                        |
| ------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `id`                                                   | Unique local row ID; repeated client request IDs are not deduplicated because another upstream call can incur another charge.                  |
| `started_at_ms`, `duration_ns`                         | UTC Unix start time in milliseconds, elapsed call time in nanoseconds.                                                                         |
| `did`                                                  | Caller DID checked against the authenticated record store, not a model/browser-supplied owner.                                                 |
| `request_id`, `trace_id`                               | Client request correlation ID and server OTel trace ID (empty when no valid span exists). Not sent to the provider.                            |
| `provider`, `model`                                    | Server-configured provider and model.                                                                                                          |
| `outcome`, `http_status`                               | Transport outcome and HTTP status, using the metric conventions above. A `success` can still contain a draft rejected by app-level validation. |
| `usage_state`                                          | `reported`, `missing`, `invalid`, or `unavailable`.                                                                                            |
| `input_tokens`, `output_tokens`, `cached_input_tokens` | Provider counts; all three are NULL unless usage is reported. Cached input is a subset of input.                                               |

No prompts, responses, library contents, images, API keys or provider error
bodies are stored. The database (and its backups) now contains durable account
activity metadata: restrict operator access accordingly. There is no public
query endpoint or new admin UI. No historical rows can be backfilled from the
old character quotas.

For example, find the accounts driving usage over the last 30 days:

```sql
SELECT did, provider, model,
       COUNT(*) AS requests,
       SUM(outcome != 'success') AS failed_requests,
       SUM(input_tokens) AS input_tokens,
       SUM(output_tokens) AS output_tokens,
       SUM(cached_input_tokens) AS cached_input_tokens,
       SUM(usage_state != 'reported') AS calls_without_token_counts
FROM ai_request_log
WHERE started_at_ms >= unixepoch('now', '-30 days') * 1000
GROUP BY did, provider, model
ORDER BY input_tokens DESC;
```

Filter by `did` and start time for indexed per-account lookups. NULL sums mean
no reported usage, not zero spending. These are operator queries, never scans on
the chat request path. Inserts are constant-sized and maintain time, account/
time, and account/request-ID indexes.

Writes run synchronously with a separate two-second timeout so client
cancellation does not erase usage history. A write failure increments
`arabica_ai_request_log_errors_total` and emits a content-free error; it does
not discard a successful reply or retry the provider. Startup schema failures
stop startup. This is an operational log, **not an exactly-once billing
ledger**: abrupt termination before the completion callback, storage failures,
or a provider billing a failed call without reporting usage can leave gaps.
Failed writes are not queued for retry. Provider billing remains authoritative.

## Alerts

`grafana/ai-alerts.yml` supplies Prometheus rules for provider errors, rate
limits, incomplete usage reporting, high token volume, slow responses and failed
request-log writes. `nix run .#monitoring` loads them into the local Prometheus.
For production, add the file to your Prometheus `rule_files` and reload the
server. Tune thresholds (especially the example one-million-tokens/hour warning)
to your traffic and budget.

**Notification delivery needs an operator-configured Alertmanager or Grafana
contact point.** The local stack evaluates alerts but does not send
notifications. Include deployment/cluster labels in grouping when using a shared
multi-cluster Prometheus. Token metrics cannot detect all provider-side charges
or missing scrapes; retain the provider's own spending alerts and hard limits.

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
`cached_input` as independent charges. Prices must be updated when the
configured model or contract changes. Estimates are lower bounds when usage is
unavailable. Prometheus owns retention; counters reset on process restart and
`increase` handles observed resets, but unsampled calls can still be lost.
