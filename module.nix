{ config, lib, pkgs, ... }:

let cfg = config.services.arabica;
in {
  options.services.arabica = {
    enable = lib.mkEnableOption "Arabica coffee brew tracking service";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ./default.nix { };
      defaultText = lib.literalExpression "pkgs.callPackage ./default.nix { }";
      description = "The arabica package to use.";
    };

    settings = {
      port = lib.mkOption {
        type = lib.types.port;
        default = 18910;
        description = "Port on which the arabica server listens.";
      };

      logLevel = lib.mkOption {
        type = lib.types.enum [ "debug" "info" "warn" "error" ];
        default = "info";
        description = "Log level for the arabica server.";
      };

      logFormat = lib.mkOption {
        type = lib.types.enum [ "pretty" "json" ];
        default = "json";
        description = "Log format. Use 'json' for production, 'pretty' for development.";
      };

      secureCookies = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Whether to set the Secure flag on cookies. Should be true when using HTTPS.";
      };
    };

    oauth = {
      clientId = lib.mkOption {
        type = lib.types.str;
        description = ''
          OAuth client ID. This should be the URL to your client-metadata.json endpoint.
          For example: https://arabica.example.com/client-metadata.json
        '';
        example = "https://arabica.example.com/client-metadata.json";
      };

      redirectUri = lib.mkOption {
        type = lib.types.str;
        description = ''
          OAuth redirect URI. This is where users are redirected after authentication.
          For example: https://arabica.example.com/oauth/callback
        '';
        example = "https://arabica.example.com/oauth/callback";
      };
    };

    dataDir = lib.mkOption {
      type = lib.types.path;
      default = "/var/lib/arabica";
      description = "Directory where arabica stores its data (OAuth sessions, etc.).";
    };

    user = lib.mkOption {
      type = lib.types.str;
      default = "arabica";
      description = "User account under which arabica runs.";
    };

    group = lib.mkOption {
      type = lib.types.str;
      default = "arabica";
      description = "Group under which arabica runs.";
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Whether to open the firewall for the arabica port.";
    };
  };

  config = lib.mkIf cfg.enable {
    users.users.${cfg.user} = lib.mkIf (cfg.user == "arabica") {
      isSystemUser = true;
      group = cfg.group;
      description = "Arabica service user";
      home = cfg.dataDir;
      createHome = true;
    };

    users.groups.${cfg.group} = lib.mkIf (cfg.group == "arabica") { };

    systemd.services.arabica = {
      description = "Arabica Coffee Brew Tracking Service";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        Type = "simple";
        User = cfg.user;
        Group = cfg.group;
        ExecStart = "${cfg.package}/bin/arabica";
        Restart = "on-failure";
        RestartSec = "10s";

        # Security hardening
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [ cfg.dataDir ];
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [ "AF_INET" "AF_INET6" "AF_UNIX" ];
        RestrictNamespaces = true;
        LockPersonality = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        MemoryDenyWriteExecute = true;
        SystemCallArchitectures = "native";
        CapabilityBoundingSet = "";
      };

      environment = {
        PORT = toString cfg.settings.port;
        LOG_LEVEL = cfg.settings.logLevel;
        LOG_FORMAT = cfg.settings.logFormat;
        SECURE_COOKIES = lib.boolToString cfg.settings.secureCookies;
        OAUTH_CLIENT_ID = cfg.oauth.clientId;
        OAUTH_REDIRECT_URI = cfg.oauth.redirectUri;
        ARABICA_DB_PATH = "${cfg.dataDir}/arabica.db";
      };
    };

    networking.firewall =
      lib.mkIf cfg.openFirewall { allowedTCPPorts = [ cfg.settings.port ]; };
  };
}
