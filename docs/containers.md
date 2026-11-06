# Containers

Arabica can be built and run on Linux without Nix. `deploy/Dockerfile` builds the
SvelteKit SPA and Go executable in separate stages; the runtime contains neither
Node.js nor the Go toolchain. Docker/BuildKit is the tested portable build path.
The commands also accept Podman through the `engine` argument, but Podman has not
been verified here.

## Build and test without Nix

Only a container engine is needed to build; `just` is optional. Internet access
is required to fetch base images and locked Go/pnpm dependencies.

```sh
docker build --file deploy/Dockerfile --tag arabica:local .
docker build --file deploy/Dockerfile --target test .
python3 deploy/test-container.py arabica:local
```

Equivalent helpers:

```sh
just container-build
just container-check
just container-test
# Alternative engine, when installed:
just container-build arabica:local podman
just container-test arabica:local podman
```

The `test` target runs Svelte checking, frontend unit/component tests, and
`CGO_ENABLED=0 go test ./... -count=1` inside the build environment. It does not
run Playwright or the separate integration suite. Ordinary image builds do not
run these suites automatically.

The smoke test requires Python 3 on the host and a local container engine. It
creates disposable containers and a named volume, then checks:

- nonroot startup with a read-only root filesystem and all capabilities dropped;
- SQLite health, SPA direct loads, referenced JavaScript, CSS, fonts and static files;
- OAuth client metadata for a configured HTTPS origin and the anonymous session API;
- presence of the HTTPS CA bundle;
- clean SIGTERM exit and SQLite integrity;
- persisted database contents after replacing the container using the same volume;
- startup rejection of a missing explicitly configured AI configuration file.

Its containers and volume are removed on completion, including failure. It does
not authenticate to a real PDS, mutate real accounts, or call an AI provider.
Firehose connectivity is not required for these checks. The engine must be local:
its loopback-published ports must be reachable from the Python process.

Base-image release tags are configurable via `NODE_IMAGE`, `GO_IMAGE` and
`RUNTIME_IMAGE` build arguments. The Go image must satisfy `go.mod`; Node must
support the pinned pnpm and frontend dependencies. For release builds, pin base
images to reviewed digests, test the resulting application image, and deploy its
immutable digest. The build does not publish images automatically.

## Run locally

```sh
docker compose --file deploy/compose.yaml up --build --detach
# Open http://127.0.0.1:18910
docker compose --file deploy/compose.yaml logs --follow
docker compose --file deploy/compose.yaml down
```

Compose stores its own database in the Docker-managed `arabica-data` volume
(normally named `arabica_arabica-data`). It does not mount or modify your normal
local database. The application uses uid/gid 10001 inside this separate volume.

The named volume survives `compose down`. **`docker compose --file deploy/compose.yaml down --volumes`
deletes the local database**, including OAuth sessions, moderation and AI access
state. Do not use it as a routine stop command.

To run the image directly:

```sh
docker volume create arabica-data
docker run --detach --name arabica \
  --publish 127.0.0.1:18910:18910 \
  --mount type=volume,source=arabica-data,target=/var/lib/arabica \
  --read-only --tmpfs /tmp:rw,nosuid,nodev,size=64m,mode=1777 \
  --cap-drop=ALL --security-opt=no-new-privileges:true \
  arabica:local
```

Compose publishes only on loopback. For a reverse proxy serving HTTPS:

```sh
ARABICA_PUBLIC_URL=https://preview.example.com SECURE_COOKIES=true \
  docker compose --file deploy/compose.yaml up --detach
```

Public URL configuration derives the OAuth client metadata URL and callback for
that origin unless explicit OAuth settings override them. The OAuth server must
be able to fetch the metadata over public HTTPS. Local Compose uses insecure
cookies only for loopback HTTP; use secure cookies for public HTTPS deployments.

## Runtime contract

Both the portable and Nix-built images use:

| Setting | Value |
|---|---|
| User/group | `10001:10001` |
| HTTP | `0.0.0.0:18910` |
| Image working directory | `/app` |
| Persistent data | `/var/lib/arabica` (`ARABICA_DATA_DIR`) |
| Logging | JSON to stdout |
| Shutdown | SIGTERM; allow at least 30 seconds for draining |
| Writable locations | Data volume and `/tmp` |

The Nix entrypoint uses the existing package wrapper, which changes its effective
working directory to the package's immutable asset directory. Use absolute paths
for mounted configuration in either image.

An empty Docker named volume inherits the image directory ownership. A bind
mount or Kubernetes PVC must be writable by uid/gid 10001; for Kubernetes, arrange
volume permissions through the storage driver/security context (`fsGroup: 10001`
where supported). Do not run the application as root to bypass permissions.

Neither image includes operator `ai.json`, `roles.json`, `known-dids.txt`, or
`.env` files. Mount configuration read-only at explicit absolute paths and set
`ARABICA_AI_CONFIG` / `ARABICA_MODERATORS_CONFIG`. Supply AI keys and SMTP
credentials at runtime, not with build arguments or Dockerfile `ENV`. Config files
must be readable by uid 10001. See [AI configuration](api/ai-chat.md).

`deploy/Dockerfile.dockerignore` allowlists build inputs and excludes common secret filenames and
generated artifacts. Keep arbitrary credentials outside the source tree; an
innocuously named file beneath an included source directory is still a build input.

`/healthz` returns 503 on a SQLite ping failure, but disconnected firehose status
is degraded with HTTP 200. Its JSON includes index readiness, which is not an
HTTP readiness gate. Choose Kubernetes probe semantics accordingly; do not make
an upstream firehose outage trigger continuous application restarts.

Metrics remain on container loopback `127.0.0.1:9101`; publishing port 9101 does
not make them accessible. No public metrics endpoint is added by this packaging.

## Build the Nix image

The flake exposes `packages.<system>.container` for `x86_64-linux` and
`aarch64-linux`. Each is a native-architecture image archive built with the
existing Nix application package; Nix builds also run that package's Go tests.
The archive uses the Docker image format accepted by Docker and containerd.
Building the image requires Nix; running it on Debian or another Linux host does
not.

For a local working tree, including uncommitted Jujutsu changes:

```sh
nix build --impure --file nix/build-container.nix --out-link result-container
docker load --input result-container
python3 deploy/test-container.py arabica:local
```

The working-tree entrypoint filters source inputs **before** passing the flake to
Nix. Do not substitute `path:$PWD#container`: a raw path flake can copy ignored
operator configuration into the Nix store before package-level filtering occurs.
For a published, secret-free source revision, the ordinary flake output also works:

```sh
nix build github:arabica-social/arabica/<revision>#container
```

Use separate tags if keeping portable and Nix images loaded simultaneously: both
examples default to `arabica:local`. Transfer the archive to another machine and
load it there, or tag/push the loaded image to your own registry.

For local k3s image testing on a matching CPU architecture:

```sh
sudo k3s ctr --namespace k8s.io images import result-container
```

Reference the imported image by its full name (`docker.io/library/arabica:local`)
and use `imagePullPolicy: Never` for that local test. Import on every node that
might run the pod, or use a registry for normal deployments. Image imports are not
a Kubernetes deployment: ingress/TLS, configuration, a PVC, probes and backups
still need to be supplied. Use one application replica and avoid overlapping
versions against the current SQLite volume.

## Independent preview instances

Give each instance its own volume, hostname, configuration and OAuth metadata.
Separate Compose project names likewise create separate named volumes; choose a
different `ARABICA_HOST_PORT` when running several on one host.

A separate local database does **not** sandbox PDS mutations. Logging into a
preview with a real account can change that account's real records. Use dedicated
test accounts for destructive testing, and do not copy production OAuth sessions
into preview databases.

Application backups currently land beneath the data directory. Copy consistent
backups off the host/volume and test restoration; the PVC is not a backup. An
image rollback does not undo database migrations or PDS writes.
