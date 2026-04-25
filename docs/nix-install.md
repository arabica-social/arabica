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

## Build/Run Manually (flake)

```bash
# Build package
nix build .#arabica

# Run built binary
./result/bin/arabica

# Or run directly
nix run .#arabica
```

By default the wrapper stores data at `~/.local/share/arabica/arabica.db` when
`ARABICA_DB_PATH` is not set.
