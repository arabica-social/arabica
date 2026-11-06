let
  source = builtins.path {
    path = ../.;
    name = "arabica-container-source";
    filter = import ./source-filter.nix {
      root = ../.;
      includeNix = true;
    };
  };
  flake = builtins.getFlake (builtins.unsafeDiscardStringContext (toString source));
in
flake.packages.${builtins.currentSystem}.container
