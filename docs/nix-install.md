# NixOS Installation

## Using the Module

Add to your configuration.nix:

```nix
{
  imports = [ ./arabica-site/module.nix ];
  
  services.arabica = {
    enable = true;
    port = 18910;
    dataDir = "/var/lib/arabica";
    logLevel = "info";
    secureCookies = false; # Set true if behind HTTPS proxy
  };
}
```

## Manual Installation

Build and run directly:

```bash
# Build
nix-build -E 'with import <nixpkgs> {}; callPackage ./default.nix {}'

# Run
result/bin/arabica
```

The data directory will be created at `~/.local/share/arabica/` by default.
