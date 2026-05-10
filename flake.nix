{
  description = "Arabica - Coffee brew tracking application";
  inputs = { nixpkgs.url = "nixpkgs/nixpkgs-unstable"; };
  outputs = { nixpkgs, self, ... }:
    let
      forAllSystems = function:
        nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-linux" ]
        (system: function nixpkgs.legacyPackages.${system} system);
    in {
      devShells = forAllSystems (pkgs: system: {
        default =
          pkgs.mkShell { packages = with pkgs; [ go templ ]; };
      });

      packages = forAllSystems (pkgs: system: rec {
        arabica = pkgs.callPackage ./nix/default.nix { };
        default = arabica;
      });

      apps = forAllSystems (pkgs: system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.arabica}/bin/arabica";
        };
        tailwind = {
          type = "app";
          program = toString (pkgs.writeShellScript "tailwind-build" ''
            cd ${./.}
            cat static/css/tokens.css static/css/reset.css static/css/utilities.css static/css/components.css > static/css/output.css
          '');
        };
        monitoring = import ./nix/monitoring.nix { inherit pkgs; };
      });

      nixosModules.default = import ./nix/module.nix;
    };
}
