{
  lib,
  buildGoModule,
  templ,
  tailwindcss_4,
  appName ? "arabica",
}:

buildGoModule {
  pname = appName;
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
    go build -o ${appName} ./cmd/${appName}
    runHook postBuild
  '';

  installPhase =
    let
      wrapperScript = ''
        #!/bin/sh
        SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
        SHARE_DIR="$SCRIPT_DIR/../share/${appName}"
        cd "$SHARE_DIR"
        exec "$SCRIPT_DIR/${appName}-unwrapped" "$@"
      '';
    in
    ''
          mkdir -p $out/bin
          mkdir -p $out/share/${appName}

          # Copy static files
          cp -r static $out/share/${appName}/
          cp ${appName} $out/bin/${appName}-unwrapped
          cat > $out/bin/${appName} <<'WRAPPER'
      ${wrapperScript}
      WRAPPER
          chmod +x $out/bin/${appName}
    '';

  meta = with lib; {
    description = "${appName} — AT Protocol app";
    license = licenses.mit;
    platforms = platforms.linux;
    mainProgram = appName;
  };
}
