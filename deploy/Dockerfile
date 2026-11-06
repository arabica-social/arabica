# syntax=docker/dockerfile:1
ARG NODE_IMAGE=node:24-bookworm-slim
ARG GO_IMAGE=golang:1.27-bookworm
ARG RUNTIME_IMAGE=debian:bookworm-slim

FROM ${NODE_IMAGE} AS frontend
WORKDIR /src
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY web/package.json ./web/package.json
RUN npm install --global "pnpm@$(node -p "require('./package.json').packageManager.split('@')[1]")"
RUN pnpm install --frozen-lockfile
COPY web/ ./web/
RUN pnpm --dir web run build

FROM frontend AS frontend-test
RUN pnpm --dir web run check && pnpm --dir web run test
RUN touch /frontend-tests-passed

FROM ${GO_IMAGE} AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=frontend /src/web/build/ ./internal/web/spa/build/
RUN CGO_ENABLED=0 go build -trimpath -o /out/arabica ./cmd/arabica

FROM backend AS test
COPY ai.example.json ./ai.example.json
COPY --from=frontend-test /frontend-tests-passed /frontend-tests-passed
RUN CGO_ENABLED=0 go test ./... -count=1

FROM ${RUNTIME_IMAGE} AS runtime
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 arabica \
    && useradd --uid 10001 --gid 10001 --home-dir /var/lib/arabica --no-create-home arabica \
    && mkdir -p /app /var/lib/arabica \
    && chown 10001:10001 /var/lib/arabica
WORKDIR /app
COPY --from=backend /out/arabica /usr/local/bin/arabica
COPY static/ ./static/
ENV ARABICA_DATA_DIR=/var/lib/arabica LOG_FORMAT=json
USER 10001:10001
EXPOSE 18910
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/arabica"]
