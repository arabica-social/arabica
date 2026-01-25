{ lib, buildGoModule, buildNpmPackage }:

let
  frontend = buildNpmPackage {
    pname = "arabica-frontend";
    version = "0.1.0";
    src = ./frontend;
    npmDepsHash = "sha256-zCQiB+NV3iIxZtZ/hHKZ23FbLzBDJmmngBJ4s3QPhyk=";

    preBuild = ''
      mkdir -p ../static
    '';

    buildPhase = ''
      npm run build
    '';

    installPhase = ''
      mkdir -p $out
      cp -r ../static/app $out/
    '';
  };
in buildGoModule {
  pname = "arabica";
  version = "0.1.0";
  src = ./.;
  vendorHash = "sha256-xgxoI2tmT4tVjgy+dv96ptI2YSU8T+Yq+rzApAiJ3yw=";

  preBuild = ''
    echo "Copying pre-built frontend..."
    mkdir -p static/app
    cp -r ${frontend}/app/* static/app/ || true
    cp -r ${frontend}/* static/app/ || true
    ls -la static/app/ || true
  '';

  buildPhase = ''
    runHook preBuild
    go build -o arabica cmd/arabica-server/main.go
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

        # Copy static files 
        cp -r static $out/share/arabica/
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
