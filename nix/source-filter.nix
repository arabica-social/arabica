{
  root,
  includeNix ? false,
}:
path: type:
let
  rootString = toString root;
  pathString = toString path;
  relative = builtins.substring (builtins.stringLength rootString + 1) (-1) pathString;
  parts = builtins.match "([^/]+)(/.*)?" relative;
  topLevel = if parts == null then "" else builtins.head parts;
  basename = baseNameOf path;
  hasPrefix = prefix: value:
    builtins.substring 0 (builtins.stringLength prefix) value == prefix;
  hasSuffix = suffix: value:
    let
      offset = builtins.stringLength value - builtins.stringLength suffix;
    in
    offset >= 0 && builtins.substring offset (-1) value == suffix;
in
pathString == rootString
|| (
  builtins.elem topLevel ([
    "cmd"
    "internal"
    "lexicons"
    "static"
    "tests"
    "web"
    "go.mod"
    "go.sum"
    "package.json"
    "pnpm-lock.yaml"
    "pnpm-workspace.yaml"
    "ai.example.json"
  ] ++ (if includeNix then [ "flake.nix" "flake.lock" "nix" ] else [ ]))
  && !(builtins.elem basename [
    "node_modules"
    ".svelte-kit"
    "test-results"
    "playwright-report"
    "storybook-static"
    "ai.json"
    "roles.json"
    "known-dids.txt"
    ".env"
    ".server-url"
    ".server-did"
    ".control-url"
    ".git"
    ".jj"
  ])
  && !(hasPrefix ".env." basename)
  && !(relative == "web/build" || hasPrefix "web/build/" relative)
  && !(hasPrefix "internal/web/spa/build/" relative && basename != ".gitkeep")
  && !(builtins.any (suffix: hasSuffix suffix relative) [
    ".secret"
    ".pem"
    ".key"
    ".log"
    ".out"
    ".bak"
    ".db"
    ".db-shm"
    ".db-wal"
    ".swp"
    ".swo"
    "~"
  ])
)
