{
  description = "Arabica & Oolong — AT Protocol brew/tea tracking apps";
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
        arabica = pkgs.callPackage ./nix/default.nix { appName = "arabica"; };
        oolong = pkgs.callPackage ./nix/default.nix { appName = "oolong"; };
        default = arabica;
      });

      apps = forAllSystems (pkgs: system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.arabica}/bin/arabica";
        };
        arabica = {
          type = "app";
          program = "${self.packages.${system}.arabica}/bin/arabica";
        };
        oolong = {
          type = "app";
          program = "${self.packages.${system}.oolong}/bin/oolong";
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

      nixosModules = {
        arabica = import ./nix/module.nix;
        oolong = import ./nix/oolong-module.nix;
        default = self.nixosModules.arabica;
      };
    };
}
