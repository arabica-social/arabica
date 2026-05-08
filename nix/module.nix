{ config, lib, pkgs, ... }:

let
  cfg = config.services.arabica;

  # teaName is the single source of truth on the nix side for the
  # tea-tracking sister app. It must match cmd/server/apps.go's
  # teaAppName constant — the binary derives its env-var prefix and
  # data-dir name from that constant, and this module sets envs based
  # on this value. Renaming the tea app means bumping both.
  teaName = "matcha";
  teaPrefix = lib.toUpper teaName;

  moderatorUserType = lib.types.submodule {
    options = {
      did = lib.mkOption {
        type = lib.types.str;
        description = "AT Protocol DID of the moderator.";
        example = "did:plc:abc123xyz";
      };
      handle = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Optional handle for the moderator (for readability).";
        example = "alice.bsky.social";
      };
      role = lib.mkOption {
        type = lib.types.enum [ "admin" "moderator" ];
        description = "The moderation role assigned to this user.";
      };
      note = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Optional note about this moderator.";
      };
    };
  };

  # Build the moderators JSON config file from Nix settings
  moderatorsConfigFile = pkgs.writeText "moderators.json" (builtins.toJSON {
    roles = {
      admin = {
        description = "Full platform control";
        permissions = [
          "hide_record"
          "unhide_record"
          "blacklist_user"
          "unblacklist_user"
          "view_reports"
          "dismiss_report"
          "view_audit_log"
          "reset_autohide"
        ];
      };
      moderator = {
        description = "Content moderation";
        permissions =
          [ "hide_record" "unhide_record" "view_reports" "dismiss_report" ];
      };
    };
    users = map (u:
      {
        inherit (u) did role;
      } // lib.optionalAttrs (u.handle != "") { inherit (u) handle; }
      // lib.optionalAttrs (u.note != "") { inherit (u) note; })
      cfg.moderation.moderators;
  });

  # Resolve the config path: explicit file takes priority, then generated from moderators list
  effectiveConfigPath = if cfg.moderation.configFile != null then
    cfg.moderation.configFile
  else if cfg.moderation.moderators != [ ] then
    moderatorsConfigFile
  else
    null;
in {
  options.services.arabica = {
    enable = lib.mkEnableOption "Arabica coffee brew tracking service";

    mode = lib.mkOption {
      type = lib.types.enum [ "all" "arabica" "matcha" ];
      default = "all";
      description = ''
        Which apps the unified server binary should boot. "all" runs
        both arabica (coffee) and the tea sister app in one process on
        distinct ports. "arabica" or "matcha" runs just one. Maps to
        the APPS environment variable.
      '';
    };

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
        description =
          "Log format. Use 'json' for production, 'pretty' for development.";
      };

      secureCookies = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description =
          "Whether to set the Secure flag on cookies. Should be true when using HTTPS.";
      };

      publicUrl = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        description = ''
          Public-facing URL of the server (e.g. https://arabica.social).
          Used for absolute URLs in OpenGraph metadata. If not set, the
          server derives it from the Host header at request time.
        '';
        example = "https://arabica.social";
      };
    };

    moderation = {
      configFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = ''
          Path to a moderators JSON config file. If set, this takes priority
          over the `moderators` list option. See the project README for the
          expected format.
        '';
        example = "/etc/arabica/moderators.json";
      };

      moderators = lib.mkOption {
        type = lib.types.listOf moderatorUserType;
        default = [ ];
        description = ''
          List of moderator users. When set, a config file is generated
          automatically with the standard admin and moderator roles.
          Ignored if `configFile` is set.
        '';
        example = lib.literalExpression ''
          [
            { did = "did:plc:abc123"; role = "admin"; handle = "alice.bsky.social"; note = "Platform owner"; }
            { did = "did:plc:def456"; role = "moderator"; handle = "bob.bsky.social"; }
          ]
        '';
      };
    };

    smtp = {
      enable = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description = ''
          Enable SMTP email notifications for join requests.
          SMTP credentials (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, SMTP_FROM)
          can be provided via environmentFiles.
        '';
      };

      host = lib.mkOption {
        type = lib.types.str;
        default = "";
        description =
          "SMTP server hostname. Can also be set via SMTP_HOST in an environment file.";
        example = "smtp.example.com";
      };

      port = lib.mkOption {
        type = lib.types.nullOr lib.types.port;
        default = null;
        description =
          "SMTP server port. Can also be set via SMTP_PORT in an environment file.";
      };

      from = lib.mkOption {
        type = lib.types.str;
        default = "";
        description =
          "Sender address for outgoing email. Can also be set via SMTP_FROM in an environment file.";
        example = "noreply@arabica.example.com";
      };
    };

    environmentFiles = lib.mkOption {
      type = lib.types.listOf lib.types.path;
      default = [ ];
      description = ''
        List of environment files to load into the systemd service.
        Useful for secrets like SMTP_USER and SMTP_PASS that should
        not be stored in the Nix store.
      '';
      example = lib.literalExpression ''[ "/run/secrets/arabica.env" ]'';
    };

    oauth = {
      clientId = lib.mkOption {
        type = lib.types.str;
        description = ''
          OAuth client ID. This should be the URL to your client metadata endpoint.
          For example: https://arabica.example.com/.well-known/oauth-client-metadata.json
        '';
        example = "https://arabica.example.com/.well-known/oauth-client-metadata.json";
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
      description =
        "Directory where arabica stores its data (OAuth sessions, etc.).";
    };

    # Tea-app (matcha) settings. Mirrors the top-level arabica options
    # but scoped under `matcha` so a host running both apps from the
    # unified binary can configure each independently. The binary
    # reads <APP>_PORT, <APP>_PUBLIC_URL, <APP>_OAUTH_*, <APP>_DATA_DIR,
    # <APP>_METRICS_PORT, <APP>_BIND_ADDR — where <APP> is the
    # uppercase teaName ("MATCHA" today).
    matcha = {
      port = lib.mkOption {
        type = lib.types.port;
        default = 18920;
        description = "Port on which the tea (matcha) server listens.";
      };

      bindAddr = lib.mkOption {
        type = lib.types.str;
        default = "0.0.0.0";
        description = "Bind address for the tea (matcha) HTTP listener.";
      };

      metricsPort = lib.mkOption {
        type = lib.types.port;
        default = 9102;
        description =
          "Localhost-only Prometheus metrics port for the tea app.";
      };

      publicUrl = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        description = ''
          Public-facing URL of the tea server (e.g. https://matcha.social).
          Used for absolute URLs in OpenGraph metadata and OAuth callbacks
          when the corresponding oauth.* options are unset.
        '';
        example = "https://matcha.social";
      };

      dataDir = lib.mkOption {
        type = lib.types.path;
        default = "/var/lib/matcha";
        description = "Data directory for the tea app.";
      };

      oauth = {
        clientId = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = null;
          description = ''
            OAuth client ID for the tea app. If null, the binary falls
            back to localhost development mode for the tea listener.
          '';
        };

        redirectUri = lib.mkOption {
          type = lib.types.nullOr lib.types.str;
          default = null;
          description = "OAuth redirect URI for the tea app.";
        };
      };

      openFirewall = lib.mkOption {
        type = lib.types.bool;
        default = false;
        description =
          "Whether to open the firewall for the tea app's HTTP port.";
      };
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

    otelEndpoint = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      description =
        "OTLP HTTP endpoint for OpenTelemetry traces (e.g. localhost:4318).";
      example = "localhost:4318";
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

        EnvironmentFile = cfg.environmentFiles;

        # Security hardening
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [ cfg.dataDir ]
          ++ lib.optional (cfg.mode != "arabica") cfg.matcha.dataDir;
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
        APPS = cfg.mode;
        LOG_LEVEL = cfg.settings.logLevel;
        LOG_FORMAT = cfg.settings.logFormat;
        SECURE_COOKIES = lib.boolToString cfg.settings.secureCookies;
        # Arabica per-app env (uppercase prefix matches app.Name).
        ARABICA_PORT = toString cfg.settings.port;
        ARABICA_OAUTH_CLIENT_ID = cfg.oauth.clientId;
        ARABICA_OAUTH_REDIRECT_URI = cfg.oauth.redirectUri;
        ARABICA_DATA_DIR = cfg.dataDir;
        # Tea (matcha) per-app env. Always exported so combined-mode
        # boots find them; ignored when mode = "arabica".
        "${teaPrefix}_PORT" = toString cfg.matcha.port;
        "${teaPrefix}_BIND_ADDR" = cfg.matcha.bindAddr;
        "${teaPrefix}_METRICS_PORT" = toString cfg.matcha.metricsPort;
        "${teaPrefix}_DATA_DIR" = cfg.matcha.dataDir;
      } // lib.optionalAttrs (cfg.settings.publicUrl != null) {
        ARABICA_PUBLIC_URL = cfg.settings.publicUrl;
      } // lib.optionalAttrs (cfg.matcha.publicUrl != null) {
        "${teaPrefix}_PUBLIC_URL" = cfg.matcha.publicUrl;
      } // lib.optionalAttrs (cfg.matcha.oauth.clientId != null) {
        "${teaPrefix}_OAUTH_CLIENT_ID" = cfg.matcha.oauth.clientId;
      } // lib.optionalAttrs (cfg.matcha.oauth.redirectUri != null) {
        "${teaPrefix}_OAUTH_REDIRECT_URI" = cfg.matcha.oauth.redirectUri;
      } // lib.optionalAttrs (effectiveConfigPath != null) {
        ARABICA_MODERATORS_CONFIG = toString effectiveConfigPath;
      } // lib.optionalAttrs (cfg.smtp.enable && cfg.smtp.host != "") {
        SMTP_HOST = cfg.smtp.host;
      } // lib.optionalAttrs (cfg.smtp.enable && cfg.smtp.port != null) {
        SMTP_PORT = toString cfg.smtp.port;
      } // lib.optionalAttrs (cfg.smtp.enable && cfg.smtp.from != "") {
        SMTP_FROM = cfg.smtp.from;
      } // lib.optionalAttrs (cfg.otelEndpoint != null) {
        OTEL_EXPORTER_OTLP_ENDPOINT = cfg.otelEndpoint;
      };
    };

    networking.firewall.allowedTCPPorts =
      lib.optional (cfg.openFirewall && cfg.mode != "matcha") cfg.settings.port
      ++ lib.optional (cfg.matcha.openFirewall && cfg.mode != "arabica") cfg.matcha.port;
  };
}
