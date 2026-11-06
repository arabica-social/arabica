{
  dockerTools,
  cacert,
  coreutils,
  arabica,
  tag ? "local",
}:

dockerTools.buildLayeredImage {
  name = "arabica";
  inherit tag;
  contents = [
    arabica
    cacert
    coreutils
    dockerTools.binSh
  ];

  extraCommands = ''
    mkdir -p app etc tmp var/lib/arabica
    ln -s ${arabica}/share/arabica/static app/static
    ln -s ca-bundle.crt etc/ssl/certs/ca-certificates.crt
    printf 'arabica:x:10001:10001:Arabica:/var/lib/arabica:/bin/sh\n' > etc/passwd
    printf 'arabica:x:10001:\n' > etc/group
    chmod 1777 tmp
  '';

  fakeRootCommands = ''
    chown 10001:10001 var/lib/arabica
  '';

  config = {
    Entrypoint = [ "${arabica}/bin/arabica" ];
    User = "10001:10001";
    WorkingDir = "/app";
    ExposedPorts = { "18910/tcp" = { }; };
    Env = [
      "PATH=/bin"
      "ARABICA_PORT=18910"
      "ARABICA_DATA_DIR=/var/lib/arabica"
      "LOG_FORMAT=json"
      "SSL_CERT_FILE=/etc/ssl/certs/ca-bundle.crt"
      "HOME=/var/lib/arabica"
    ];
  };
}
