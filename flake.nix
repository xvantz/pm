{
  description = "PM — Project Memory: долговременная память проектов";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        rec {
          pm = pkgs.buildGoModule {
            pname = "pm";
            version = "0.1.0";

            src = ./.;

            subPackages = [
              "cmd/pm"
              "cmd/pm-mcp"
            ];

            ldflags = [
              "-s"
              "-w"
              "-X main.Version=0.1.0"
            ];

            vendorHash = "sha256-4HpSNp8hmMkcMeO9SGuq8hqQq6tvU1mJO2Nfvt50amQ=";

            meta = {
              description = "Project Memory — долговременная память проектов";
              homepage = "https://git.827482.xyz/xvantz/pm";
              license = pkgs.lib.licenses.mit;
              maintainers = [ "Ivan R. <ivan@xvantz.dev>" ];
              platforms = [ "x86_64-linux" ];
            };
          };

          default = pm;
        }
      );

      nixosModules.default =
        { config, lib, pkgs, ... }:
        with lib;
        let
          system = pkgs.stdenv.hostPlatform.system;
          cfg = config.services.pm;
        in
        {
          options.services.pm = {
            enable = mkEnableOption "PM Project Memory";

            package = mkOption {
              type = types.package;
              default = self.packages.${system}.pm;
              defaultText = literalExpression "self.packages.${pkgs.stdenv.hostPlatform.system}.pm";
              description = "pm package to use";
            };

            dataDir = mkOption {
              type = types.str;
              default = "/home/xvantz/Documents/pm";
              description = "Host directory for PM project data (YAML store). Only the pm-serve daemon reads it.";
            };

            user = mkOption {
              type = types.str;
              example = "xvantz";
              description = ''
                User the pm-serve daemon runs as (REQUIRED, no default:
                a system module cannot guess whose data it serves).
                Must read dataDir: keep it as the owner of that directory.
                The same user gets PM_TOKEN in login shells (see below).
              '';
            };

            listenAddr = mkOption {
              type = types.str;
              default = "127.0.0.1:8472";
              description = "Address for `pm serve` daemon (single writer HTTP API). Keep localhost unless behind Tailscale. Never 0.0.0.0 to the world.";
            };

            environmentFile = mkOption {
              type = types.nullOr types.path;
              default = null;
              example = literalExpression "config.sops.secrets.pm_env.path";
              description = ''
                Env file with PM_TOKEN for `pm serve` (same convention as
                hermes_env/forgejo_env: sops secret, KEY=value lines).
                The daemon is useless without a token, so an enabled service
                without this file fails the build (see assertions).
              '';
            };

            backup = {
              enable = mkEnableOption "git backup of the PM data directory (one commit per action, async push off-host)";

              repoUrl = mkOption {
                type = types.nullOr types.str;
                default = null;
                example = "git@git.827482.xyz:xvantz/pm-data.git";
                description = ''
                  Git remote for the data-dir backup (passed as PM_BACKUP_REPO).
                  Required when backup is enabled. Unset repo means no backup:
                  the daemon serves the same API without committing.
                '';
              };

              keyFile = mkOption {
                type = types.nullOr types.path;
                default = null;
                example = literalExpression "config.sops.secrets.pm_backup_key.path";
                description = ''
                  SSH private key for the backup remote (passed as PM_BACKUP_KEY,
                  used via GIT_SSH_COMMAND with IdentitiesOnly). Same secrecy
                  as environmentFile: sops secret, mode 600, service user only.
                  Optional: without it SSH falls back to the default agent/keys.
                '';
              };

              tokenFile = mkOption {
                type = types.nullOr types.path;
                default = null;
                example = literalExpression "config.sops.secrets.pm_backup_token.path";
                description = ''
                  File holding a Bearer token for an HTTPS backup remote
                  (passed as PM_BACKUP_TOKEN, sent as an Authorization header
                  via git env config, never on the command line or baked into
                  the URL). Same secrecy as keyFile. Use this for HTTPS remotes,
                  keyFile for SSH ones.
                '';
              };
            };
          };

          config = mkIf cfg.enable {
            assertions = [
              {
                assertion = cfg.environmentFile != null;
                message = ''
                  services.pm.environmentFile is not set.
                  Add a sops secret with PM_TOKEN, e.g.:
                    sops.secrets.pm_env = { owner = "xvantz"; restartUnits = [ "pm-serve.service" ]; };
                  and point services.pm.environmentFile at config.sops.secrets.pm_env.path,
                  then put PM_TOKEN=... into secrets.yaml (sops).
                '';
              }
              {
                assertion = !cfg.backup.enable || cfg.backup.repoUrl != null;
                message = ''
                  services.pm.backup is enabled without services.pm.backup.repoUrl.
                  Set the backup remote, e.g.:
                    services.pm.backup.repoUrl = "git@git.827482.xyz:xvantz/pm-data.git";
                  or disable backup (the default): services.pm.backup.enable = false.
                '';
              }
            ];

            environment.systemPackages = [ cfg.package ];
            environment.sessionVariables.PM_DIR = cfg.dataDir;
            # PM_API is not a secret (plain http address), so unlike PM_TOKEN
            # it can live in the store. Derived from listenAddr: changing the
            # address never orphans shells on the default.
            environment.sessionVariables.PM_API = "http://${cfg.listenAddr}";
            environment.interactiveShellInit = ''
              export PM_DIR="${cfg.dataDir}"
              export PM_API="http://${cfg.listenAddr}"
            '';

            # The service provisions PM_TOKEN into the user's login shells:
            # sourced at runtime from the same sops env file the daemon
            # uses. Nothing secret touches the nix store; no hand edits
            # to shell configs needed.
            programs.zsh.interactiveShellInit = mkIf (cfg.environmentFile != null) ''
              if [[ -r "${cfg.environmentFile}" ]]; then
                set -a; source "${cfg.environmentFile}"; set +a
              fi
            '';
            programs.bash.interactiveShellInit = mkIf (cfg.environmentFile != null) ''
              if [[ -r "${cfg.environmentFile}" ]]; then
                set -a; source "${cfg.environmentFile}"; set +a
              fi
            '';

            systemd.services.pm-serve = mkIf (cfg.environmentFile != null) {
              description = "PM Project Memory daemon (single writer API)";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
              serviceConfig = {
                Type = "simple";
                # The data lives wherever dataDir points (a $HOME by default),
                # so the daemon runs as its owner, not as a DynamicUser.
                User = cfg.user;
                Restart = "always";
                RestartSec = "5";
                EnvironmentFile = cfg.environmentFile;
              };
              script = ''
                export PM_DIR="${cfg.dataDir}"
                ${optionalString (cfg.backup.enable && cfg.backup.repoUrl != null) ''
                  export PM_BACKUP_REPO="${cfg.backup.repoUrl}"
                ''}
                ${optionalString (cfg.backup.enable && cfg.backup.keyFile != null) ''
                  export PM_BACKUP_KEY="${cfg.backup.keyFile}"
                ''}
                ${optionalString (cfg.backup.enable && cfg.backup.tokenFile != null) ''
                  export PM_BACKUP_TOKEN="$(cat "${cfg.backup.tokenFile}")"
                ''}
                exec ${cfg.package}/bin/pm serve --addr "${cfg.listenAddr}"
              '';
            };
          };
        };
    };
}
