{
  lib,
  stdenvNoCC,
  base,
  appName,
}:

# Thin wrapper around the base arabica/oolong package that boots the server
# with the embedded SvelteKit SPA enabled (SPA=1). The base package already
# embeds the SPA build output (see nix/default.nix preBuild); this derivation
# only flips the runtime env var so `nix run .#arabica-spa` serves the SPA
# without rebuilding the Go/pnpm stack.
#
# The base binary is identical whether or not SPA is on — SPA activation is a
# runtime decision (internal/atplatform/server reads <PREFIX>_SPA or SPA).
# Keeping this a separate, non-default package lets the SPA version coexist
# with the legacy templ/HTMX default until the migration is complete.
stdenvNoCC.mkDerivation {
  pname = "${appName}-spa";
  inherit (base) version;

  dontUnpack = true;
  dontBuild = true;

  installPhase = ''
    runHook preInstall

    mkdir -p $out/bin $out/share

    # Reuse the base package's embedded static assets and Go binary verbatim
    # via symlinks so nothing is duplicated in the store.
    ln -s ${base}/share/${appName} $out/share/${appName}
    ln -s ${base}/bin/${appName}-unwrapped $out/bin/${appName}-unwrapped

    # SPA wrapper: same layout as the base wrapper, but exports SPA=1 before
    # exec so the server enables the SvelteKit shell handler.
    #
    # Installed under the canonical name (<appName>) so this package is a
    # drop-in for the NixOS module: setting services.arabica.package to
    # arabica-spa works unchanged, because the module ExecStart points at
    # bin/<appName> which this wrapper provides.
    # A same-name symlink <appName>-spa is also provided so `nix run
    # .#arabica-spa` and the flake `apps` entry resolve to an explicit name.
    cat > $out/bin/${appName} <<'WRAPPER'
    #!/bin/sh
    SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
    SHARE_DIR="$SCRIPT_DIR/../share/${appName}"
    cd "$SHARE_DIR"
    export SPA=1
    exec "$SCRIPT_DIR/${appName}-unwrapped" "$@"
    WRAPPER
    chmod +x $out/bin/${appName}
    ln -s ${appName} $out/bin/${appName}-spa

    runHook postInstall
  '';

  meta = with lib; {
    description = "${appName} — AT Protocol app (SvelteKit SPA enabled)";
    license = licenses.mit;
    platforms = platforms.linux;
    mainProgram = "${appName}-spa";
  };
}
