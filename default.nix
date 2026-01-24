{ lib, buildGoModule, tailwindcss }:

buildGoModule rec {
  pname = "arabica";
  version = "0.1.0";
  src = ./.;
  vendorHash = "sha256-mrIFu5c2EuGvYHyjJVqC8WzlsmUJYCm/6yUpJ0IGPlA=";

  nativeBuildInputs = [ tailwindcss ];

  preBuild = ''
    tailwindcss -i web/static/css/style.css -o web/static/css/output.css --minify
  '';

  buildPhase = ''
    runHook preBuild
    go build -o arabica cmd/server/main.go
    runHook postBuild
  '';

  installPhase = let
    wrapperScript = ''
      #!/bin/sh
      SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
      SHARE_DIR="$SCRIPT_DIR/../share/arabica"

      # Set default database path if not specified
      # Uses XDG_DATA_HOME or falls back to ~/.local/share
      if [ -z "$ARABICA_DB_PATH" ]; then
          DATA_DIR="''${XDG_DATA_HOME:-$HOME/.local/share}/arabica"
          mkdir -p "$DATA_DIR"
          export ARABICA_DB_PATH="$DATA_DIR/arabica.db"
      fi

      cd "$SHARE_DIR"
      exec "$SCRIPT_DIR/arabica-unwrapped" "$@"
    '';
  in ''
        mkdir -p $out/bin
        mkdir -p $out/share/arabica

        # Copy static files and templates
        cp -r web $out/share/arabica/
        cp -r templates $out/share/arabica/
        cp arabica $out/bin/arabica-unwrapped
        cat > $out/bin/arabica <<'WRAPPER'
    ${wrapperScript}
    WRAPPER
        chmod +x $out/bin/arabica
  '';

  meta = with lib; {
    description = "Arabica - Coffee brew tracker";
    license = licenses.mit;
    platforms = platforms.linux;
    mainProgram = "arabica";
  };
}
