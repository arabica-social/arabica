{ pkgs }:
{
  type = "app";
  program = toString (pkgs.writeShellScript "monitoring-stack" ''
    set -euo pipefail

    TMPDIR=$(mktemp -d)
    trap 'echo "Shutting down monitoring stack..."; kill $(jobs -p) 2>/dev/null; rm -rf "$TMPDIR"' EXIT INT TERM

    mkdir -p "$TMPDIR"/{grafana/{data,plugins,logs},prometheus,loki,tempo}

    # Prometheus config
    cat > "$TMPDIR/prometheus/prometheus.yml" <<'EOF'
    global:
      scrape_interval: 15s

    scrape_configs:
      - job_name: arabica
        static_configs:
          - targets: ['localhost:9101']
    EOF

    # Loki config
    cat > "$TMPDIR/loki/loki.yaml" <<'EOF'
    auth_enabled: false
    server:
      http_listen_port: 3100
    common:
      ring:
        instance_addr: 127.0.0.1
        kvstore:
          store: inmemory
      replication_factor: 1
      path_prefix: /tmp/loki
    schema_config:
      configs:
        - from: 2020-10-24
          store: tsdb
          object_store: filesystem
          schema: v13
          index:
            prefix: index_
            period: 24h
    storage_config:
      filesystem:
        directory: /tmp/loki/chunks
    EOF

    # Tempo config
    cat > "$TMPDIR/tempo/tempo.yaml" <<'EOF'
    server:
      http_listen_port: 3200
    distributor:
      receivers:
        otlp:
          protocols:
            http:
              endpoint: 0.0.0.0:4318
            grpc:
              endpoint: 0.0.0.0:4317
    storage:
      trace:
        backend: local
        local:
          path: /tmp/tempo/traces
        wal:
          path: /tmp/tempo/wal
    EOF

    # Grafana datasources
    mkdir -p "$TMPDIR/grafana/provisioning/datasources"
    cat > "$TMPDIR/grafana/provisioning/datasources/datasources.yaml" <<'EOF'
    apiVersion: 1
    datasources:
      - name: Prometheus
        type: prometheus
        uid: prometheus
        url: http://localhost:9090
        isDefault: true
      - name: Loki
        type: loki
        uid: loki
        url: http://localhost:3100
      - name: Tempo
        type: tempo
        uid: tempo
        url: http://localhost:3200
        jsonData:
          tracesToLogsV2:
            datasourceUid: loki
          serviceMap:
            datasourceUid: prometheus
    EOF

    echo "Starting Prometheus on :9090..."
    ${pkgs.prometheus}/bin/prometheus \
      --config.file="$TMPDIR/prometheus/prometheus.yml" \
      --storage.tsdb.path="$TMPDIR/prometheus/data" \
      --web.listen-address=":9090" \
      > "$TMPDIR/prometheus.log" 2>&1 &

    echo "Starting Loki on :3100..."
    ${pkgs.grafana-loki}/bin/loki \
      -config.file="$TMPDIR/loki/loki.yaml" \
      > "$TMPDIR/loki.log" 2>&1 &

    echo "Starting Tempo on :3200 (OTLP HTTP :4318, OTLP gRPC :4317)..."
    ${pkgs.tempo}/bin/tempo \
      -config.file="$TMPDIR/tempo/tempo.yaml" \
      > "$TMPDIR/tempo.log" 2>&1 &

    echo "Starting Grafana on :3000 (admin/admin)..."
    GF_PATHS_DATA="$TMPDIR/grafana/data" \
    GF_PATHS_PLUGINS="$TMPDIR/grafana/plugins" \
    GF_PATHS_LOGS="$TMPDIR/grafana/logs" \
    GF_PATHS_PROVISIONING="$TMPDIR/grafana/provisioning" \
    GF_SERVER_HTTP_PORT=3000 \
    GF_AUTH_ANONYMOUS_ENABLED=true \
    GF_AUTH_ANONYMOUS_ORG_ROLE=Admin \
      ${pkgs.grafana}/bin/grafana server \
      --homepath=${pkgs.grafana}/share/grafana \
      > "$TMPDIR/grafana.log" 2>&1 &

    echo ""
    echo "Monitoring stack running:"
    echo "  Grafana:    http://localhost:3000"
    echo "  Prometheus: http://localhost:9090"
    echo "  Loki:       http://localhost:3100"
    echo "  Tempo:      http://localhost:3200"
    echo ""
    echo "Arabica should send traces to OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318"
    echo "Arabica metrics are scraped from :9101/metrics"
    echo ""
    echo "Press Ctrl+C to stop."
    wait
  '');
}
