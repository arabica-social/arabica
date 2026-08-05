{
  lib,
  buildGoModule,
  nodejs,
  pnpm,
  fetchPnpmDeps,
  zstd,
  appName ? "arabica",
}:

buildGoModule rec {
  pname = appName;
  version = "0.1.0";
  src = ../.;
  vendorHash = "sha256-3VCfmyT4MHdNrU5BomDE0U5xlmuVWVQLzDFLsGKaLf4=";

  pnpmDeps = fetchPnpmDeps {
    inherit pname version src;
    fetcherVersion = 3;
    hash = "sha256-+AeoImiQZNJ5GXOocS4DNK7MgUCsosdPT4z/s43kexA=";
  };

  nativeBuildInputs = [
    nodejs
    pnpm
    zstd
  ];

  preBuild = ''
    if [[ "$name" == *-go-modules ]]; then
      :
    else
      export HOME=$TMPDIR
      export npm_config_manage_package_manager_versions=false
      STORE_PATH=$(mktemp -d)
      tar --zstd -xf ${pnpmDeps}/pnpm-store.tar.zst -C "$STORE_PATH"
      chmod -R +w "$STORE_PATH"
      pnpm config set store-dir "$STORE_PATH"
      pnpm config set package-import-method clone-or-copy
      pnpm install --offline --ignore-scripts --frozen-lockfile
      cd web && pnpm install --offline --ignore-scripts --frozen-lockfile && pnpm run build && cd ..
      mkdir -p internal/web/spa/build
      cp -r web/build/* internal/web/spa/build/
    fi
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
