# Grafana Dashboards

Importable Grafana dashboard definitions for monitoring Arabica.

## Dashboards

### `arabica-logs.json` - Log-Based Metrics

Queries structured JSON logs via **Loki**. No code changes needed - works with existing zerolog output.

**Prerequisite:** Ship Arabica logs to Loki (e.g., via Promtail, Alloy, or Docker log driver). Logs must be in JSON format (`LOG_FORMAT=json`).

**Log selector:** The dashboard uses a template variable (`$log_selector`) with three presets:

- `unit="arabica.service"` (default) - NixOS/systemd journal via Promtail
- `syslog_identifier="arabica"` - journald syslog identifier
- `app="arabica"` - Docker log driver or custom labels

Select the matching option from the dropdown at the top of the dashboard, or type a custom value.

**Sections:**

- **Overview** - stat panels for total requests, errors, logins, reports, join requests
- **HTTP Traffic** - requests by status/method, top paths, response latency
- **Firehose** - events by collection/operation, errors, backfill activity
- **Authentication & Users** - login success/failure, join requests, invites
- **Moderation** - reports, hide/unhide/block actions, permission denials
- **PDS & ATProto** - PDS request volume/latency/errors by method and collection
- **Errors & Warnings** - error/warn timeline + recent error log viewer

### `arabica-prometheus.json` - Prometheus Metrics

Queries instrumented Prometheus counters, histograms, and gauges exposed at `/metrics`.

**Prerequisite:** Arabica exposes a `/metrics` endpoint (Prometheus format). Configure Prometheus to scrape it.

**Sections:**

- **Overview** - request rate, error rate, p95 latency, firehose connection, events/s, cache hit rate
- **HTTP Traffic** - request rate by status/path, latency percentiles (p50/p95/p99), latency by path
- **Firehose** - events by collection/operation, error rate, connection state
- **PDS / ATProto** - PDS request rate by method/collection, latency by method, error rate
- **Feed Cache** - cache hits vs misses, hit rate over time

### Importing

1. In Grafana, go to **Dashboards > Import**
2. Upload the JSON file or paste its contents
3. Select your data source (Loki or Prometheus) when prompted
4. For the Loki dashboard, select the correct log selector from the dropdown (defaults to `unit="arabica.service"` for NixOS systemd)
