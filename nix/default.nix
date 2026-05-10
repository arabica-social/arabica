{
  lib,
  buildGoModule,
  templ,
  tailwindcss_4,
}:

buildGoModule {
  pname = "arabica";
  version = "0.1.0";
  src = ../.;
  vendorHash = "sha256-ZDRE/PuCOkRpencco8FpN8wxv25N7SlgiAUse0jJM/E=";

  nativeBuildInputs = [
    templ
    tailwindcss_4
  ];

  preBuild = ''
    tailwindcss -i static/css/app.css -o static/css/output.css --minify
    templ generate
  '';

  buildPhase = ''
    runHook preBuild
    go build -o arabica ./cmd/server
    runHook postBuild
  '';

  installPhase =
    let
      wrapperScript = ''
        #!/bin/sh
        SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
        SHARE_DIR="$SCRIPT_DIR/../share/arabica"
        cd "$SHARE_DIR"
        exec "$SCRIPT_DIR/arabica-unwrapped" "$@"
      '';
    in
    ''
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
