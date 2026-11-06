# Nix/NixOS Installation

## NixOS Module

This repo exposes a NixOS module at `nixosModules.default`.

### Via flake input

```nix
{
  inputs.arabica.url = "github:<you>/arabica";

  outputs = { self, nixpkgs, arabica, ... }: {
    nixosConfigurations.my-host = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        arabica.nixosModules.default
        ({ ... }: {
          services.arabica = {
            enable = true;
            dataDir = "/var/lib/arabica";

            settings = {
              port = 18910;
              logLevel = "info";
              secureCookies = true;
              # publicUrl = "https://arabica.example.com";
            };

            oauth = {
              clientId = "https://arabica.example.com/client-metadata.json";
              redirectUri = "https://arabica.example.com/oauth/callback";
            };
          };
        })
      ];
    };
  };
}
```

### Via local checkout

```nix
{
  imports = [ ./nix/module.nix ];

  services.arabica = {
    enable = true;
    dataDir = "/var/lib/arabica";

    settings = {
      port = 18910;
      logLevel = "info";
      secureCookies = false; # only for local/dev http
    };

    oauth = {
      clientId = "https://arabica.example.com/client-metadata.json";
      redirectUri = "https://arabica.example.com/oauth/callback";
    };
  };
}
```

## Moderation and AI with SOPS

The module can generate both configuration files from one NixOS configuration.
They remain separate JSON files: moderation uses its existing roles format;
AI uses [`ai.example.json`](../ai.example.json). No JSON loader migration is
required. Neither generated file should contain secrets.

With the Arabica and sops-nix NixOS modules imported and
`sops.defaultSopsFile` configured for your encrypted secrets:

```nix
{ config, ... }:
{
  sops.secrets.arabica-ai-api-key = { };
  sops.templates."arabica.env".content = ''
    ARABICA_AI_API_KEY=${config.sops.placeholder.arabica-ai-api-key}
  '';

  services.arabica = {
    enable = true;
    settings.publicUrl = "https://arabica.example.com";
    oauth = {
      clientId = "https://arabica.example.com/client-metadata.json";
      redirectUri = "https://arabica.example.com/oauth/callback";
    };
    moderation.moderators = [
      { did = "did:plc:replace-me"; role = "admin"; }
    ];
    ai.settings = {
      default_provider = "openai-compatible";
      providers.openai-compatible = {
        base_url = "https://api.openai.com/v1";
        model = "replace-with-your-model";
        api_key = "\${ARABICA_AI_API_KEY}";
        timeout_seconds = 30;
        max_request_bytes = 6291456;
        max_response_bytes = 1048576;
        max_tokens = 2048;
      };
    };
    environmentFiles = [ config.sops.templates."arabica.env".path ];
  };
}
```

The backslash in the Nix string preserves the literal `${ARABICA_AI_API_KEY}`
placeholder in generated JSON. The application expands it from the service's
environment once at startup. The API key is not evaluated into the Nix store.
The template is read by systemd, so it can remain root-readable. Store a
single-line API key without whitespace or control characters in SOPS.

For an entire AI JSON file managed by SOPS instead, use a runtime string path:

```nix
sops.secrets.arabica-ai-config = {
  owner = config.services.arabica.user;
  group = config.services.arabica.group;
};
services.arabica.ai.configFile = config.sops.secrets.arabica-ai-config.path;
```

That secret must contain the complete JSON document. `ai.configFile` takes
priority over `ai.settings`. Do not use a Nix path literal or `builtins.readFile`
for credential-bearing files: that can copy their contents into the Nix store.

Without either AI option, the existing application lookup is unchanged:
`ARABICA_AI_CONFIG` from an `environmentFiles` file overrides `./ai.json`.
A missing default disables AI; an explicitly configured missing or invalid file
fails startup. Environment-file values take precedence over module-generated
systemd environment values, so remove a stale `ARABICA_AI_CONFIG` entry when
switching to `ai.settings`. The daily allowance is separate from provider
configuration: set `ARABICA_AI_DAILY_CHARS` (default 10000, `0` disables) and
optional `ARABICA_AI_UNLIMITED_DIDS` in the same `environmentFiles` entry or the
systemd `Environment`; admins are exempt automatically. Restart the service after configuration, allowance or key changes;
configure sops-nix `restartUnits = [ "arabica.service" ];` on secrets/templates
if you want rotations to restart it automatically.

## Build/Run Manually (flake)

```bash
# Build package
nix build .#arabica

# Run built binary
./result/bin/arabica

# Or run directly
nix run .#arabica
```

The service stores its databases in `dataDir` (`/var/lib/arabica` by default),
with paths set via `ARABICA_DB_PATH` and `ARABICA_FEED_INDEX_PATH`.

### Updating dependency hashes

`nix/default.nix` pins the pnpm dependency store and Go vendor tree separately.
Refresh the pnpm hash when `pnpm-lock.yaml` changes. Refresh `vendorHash` when
Go dependencies or imported packages change, including new imports from an
already-required module.

Temporarily set the affected hash to `lib.fakeHash`, then build its dependency
output to obtain the replacement hash from Nix's `got:` diagnostic:

```bash
nix build --no-link .#arabica.pnpmDeps
nix build --no-link .#arabica.goModules
```

Replace the placeholder with the reported hash and run `nix build .#arabica`
to verify the complete package, including its tests. Keep the offline install
and frozen-lockfile checks enabled.
