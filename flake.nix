{
  description = "Arabica development and packaging flake";
  inputs = {
    nixpkgs.url = "nixpkgs/nixpkgs-unstable";
  };
  outputs =
    { nixpkgs, self, ... }:
    let
      forAllSystems =
        function:
        nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-linux" ] (
          system: function nixpkgs.legacyPackages.${system} system
        );
    in
    {
      devShells = forAllSystems (
        pkgs: system: {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              templ
              just
              nodejs
              pnpm
              prettier
            ];
          };
        }
      );
      packages = forAllSystems (
        pkgs: system: rec {
          arabica = pkgs.callPackage ./nix/default.nix { appName = "arabica"; };
          # Alternate package that boots the embedded SvelteKit SPA (SPA=1).
          # Non-default: keep the legacy templ/HTMX package as `default`
          # until the SPA migration is the primary surface.
          arabica-spa = pkgs.callPackage ./nix/spa-wrapper.nix {
            inherit (pkgs) lib stdenvNoCC;
            base = arabica;
            appName = "arabica";
          };
          default = arabica;
        }
      );
      apps = forAllSystems (
        pkgs: system: {
          default = {
            type = "app";
            program = "${self.packages.${system}.arabica}/bin/arabica";
          };
          arabica = {
            type = "app";
            program = "${self.packages.${system}.arabica}/bin/arabica";
          };
          arabica-spa = {
            type = "app";
            program = "${self.packages.${system}.arabica-spa}/bin/arabica-spa";
          };
          monitoring = import ./nix/monitoring.nix { inherit pkgs; };
        }
      );
      nixosModules = {
        arabica = import ./nix/module.nix;
        default = self.nixosModules.arabica;
      };
    };
}
