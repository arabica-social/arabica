{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.oolong;

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
        type = lib.types.enum [
          "admin"
          "moderator"
        ];
        description = "The moderation role assigned to this user.";
      };
      note = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Optional note about this moderator.";
      };
    };
  };

  moderatorsConfigFile = pkgs.writeText "oolong-moderators.json" (
    builtins.toJSON {
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
          permissions = [
            "hide_record"
            "unhide_record"
            "view_reports"
            "dismiss_report"
          ];
        };
      };
      users = map (
        u:
        {
          inherit (u) did role;
        }
        // lib.optionalAttrs (u.handle != "") { inherit (u) handle; }
        // lib.optionalAttrs (u.note != "") { inherit (u) note; }
      ) cfg.moderation.moderators;
    }
  );

  effectiveConfigPath =
    if cfg.moderation.configFile != null then
      cfg.moderation.configFile
    else if cfg.moderation.moderators != [ ] then
      moderatorsConfigFile
    else
      null;
in
{
  options.services.oolong = {
    enable = lib.mkEnableOption "Oolong tea brew tracking service";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.callPackage ./default.nix { appName = "oolong"; };
      defaultText = lib.literalExpression ''pkgs.callPackage ./default.nix { appName = "oolong"; }'';
      description = "The oolong package to use.";
    };

    settings = {
      port = lib.mkOption {
        type = lib.types.port;
        default = 18920;
        description = "Port on which the oolong server listens.";
      };

      logLevel = lib.mkOption {
        type = lib.types.enum [
          "debug"
          "info"
          "warn"
          "error"
        ];
        default = "info";
        description = "Log level for the oolong server.";
      };

      logFormat = lib.mkOption {
        type = lib.types.enum [
          "pretty"
          "json"
        ];
        default = "json";
        description = "Log format. Use 'json' for production, 'pretty' for development.";
      };

      secureCookies = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Whether to set the Secure flag on cookies. Should be true when using HTTPS.";
      };

      publicUrl = lib.mkOption {
        type = lib.types.nullOr lib.types.str;
        default = null;
        description = ''
          Public-facing URL of the server (e.g. https://oolong.example.com).
          Used for absolute URLs in OpenGraph metadata. If not set, the
          server derives it from the Host header at request time.
        '';
        example = "https://oolong.example.com";
      };
    };

    moderation = {
      configFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = ''
          Path to a moderators JSON config file. If set, this takes priority
          over the `moderators` list option.
        '';
        example = "/etc/oolong/moderators.json";
      };

      moderators = lib.mkOption {
        type = lib.types.listOf moderatorUserType;
        default = [ ];
        description = ''
          List of moderator users. When set, a config file is generated
          automatically with the standard admin and moderator roles.
          Ignored if `configFile` is set.
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
        description = "SMTP server hostname. Can also be set via SMTP_HOST in an environment file.";
        example = "smtp.example.com";
      };

      port = lib.mkOption {
        type = lib.types.nullOr lib.types.port;
        default = null;
        description = "SMTP server port. Can also be set via SMTP_PORT in an environment file.";
      };

      from = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Sender address for outgoing email. Can also be set via SMTP_FROM in an environment file.";
        example = "noreply@oolong.example.com";
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
      example = lib.literalExpression ''[ "/run/secrets/oolong.env" ]'';
    };

    oauth = {
      clientId = lib.mkOption {
        type = lib.types.str;
        description = ''
          OAuth client ID. This should be the URL to your client metadata endpoint.
        '';
        example = "https://oolong.example.com/.well-known/oauth-client-metadata.json";
      };

      redirectUri = lib.mkOption {
        type = lib.types.str;
        description = ''
          OAuth redirect URI. This is where users are redirected after authentication.
        '';
        example = "https://oolong.example.com/oauth/callback";
      };
    };

    dataDir = lib.mkOption {
      type = lib.types.path;
      default = "/var/lib/oolong";
      description = "Directory where oolong stores its data (OAuth sessions, etc.).";
    };

    user = lib.mkOption {
      type = lib.types.str;
      default = "oolong";
      description = "User account under which oolong runs.";
    };

    group = lib.mkOption {
      type = lib.types.str;
      default = "oolong";
      description = "Group under which oolong runs.";
    };

    otelEndpoint = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      description = "OTLP HTTP endpoint for OpenTelemetry traces (e.g. localhost:4318).";
      example = "localhost:4318";
    };

    openFirewall = lib.mkOption {
      type = lib.types.bool;
      default = false;
      description = "Whether to open the firewall for the oolong port.";
    };
  };

  config = lib.mkIf cfg.enable {
    users.users.${cfg.user} = lib.mkIf (cfg.user == "oolong") {
      isSystemUser = true;
      group = cfg.group;
      description = "Oolong service user";
      home = cfg.dataDir;
      createHome = true;
    };

    users.groups.${cfg.group} = lib.mkIf (cfg.group == "oolong") { };

    systemd.services.oolong = {
      description = "Oolong Tea Brew Tracking Service";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];

      serviceConfig = {
        Type = "simple";
        User = cfg.user;
        Group = cfg.group;
        ExecStart = "${cfg.package}/bin/oolong";
        Restart = "on-failure";
        RestartSec = "10s";

        EnvironmentFile = cfg.environmentFiles;

        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadWritePaths = [ cfg.dataDir ];
        ProtectKernelTunables = true;
        ProtectKernelModules = true;
        ProtectControlGroups = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = true;
        LockPersonality = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        MemoryDenyWriteExecute = true;
        SystemCallArchitectures = "native";
        CapabilityBoundingSet = "";
      };

      environment = {
        LOG_LEVEL = cfg.settings.logLevel;
        LOG_FORMAT = cfg.settings.logFormat;
        SECURE_COOKIES = lib.boolToString cfg.settings.secureCookies;
        # Oolong per-app env (uppercase prefix matches app.Name).
        OOLONG_PORT = toString cfg.settings.port;
        OOLONG_OAUTH_CLIENT_ID = cfg.oauth.clientId;
        OOLONG_OAUTH_REDIRECT_URI = cfg.oauth.redirectUri;
        OOLONG_DATA_DIR = cfg.dataDir;
      }
      // lib.optionalAttrs (cfg.settings.publicUrl != null) {
        OOLONG_PUBLIC_URL = cfg.settings.publicUrl;
      }
      // lib.optionalAttrs (effectiveConfigPath != null) {
        OOLONG_MODERATORS_CONFIG = toString effectiveConfigPath;
      }
      // lib.optionalAttrs (cfg.smtp.enable && cfg.smtp.host != "") {
        SMTP_HOST = cfg.smtp.host;
      }
      // lib.optionalAttrs (cfg.smtp.enable && cfg.smtp.port != null) {
        SMTP_PORT = toString cfg.smtp.port;
      }
      // lib.optionalAttrs (cfg.smtp.enable && cfg.smtp.from != "") {
        SMTP_FROM = cfg.smtp.from;
      }
      // lib.optionalAttrs (cfg.otelEndpoint != null) {
        OTEL_EXPORTER_OTLP_ENDPOINT = cfg.otelEndpoint;
      };
    };

    networking.firewall.allowedTCPPorts = lib.optional cfg.openFirewall cfg.settings.port;
  };
}
