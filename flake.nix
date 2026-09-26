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
              description = "Host directory for PM project data (YAML store). Used as PM_DIR.";
            };

            containerDataDir = mkOption {
              type = types.str;
              default = "/data/pm";
              description = "PM data directory path INSIDE the Hermes container (bind-mount target of dataDir).";
            };

            listenAddr = mkOption {
              type = types.str;
              default = "127.0.0.1:8472";
              description = "Address for `pm serve` daemon (single writer HTTP API). Keep localhost unless behind Tailscale. Never 0.0.0.0 to the world.";
            };

            tokenFile = mkOption {
              type = types.nullOr types.path;
              default = null;
              example = "/run/secrets/pm_token";
              description = "File containing the Bearer token for `pm serve`. Manage via sops-nix. When null, the pm-serve systemd service is not created.";
            };
          };

          config = mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];
            environment.sessionVariables.PM_DIR = cfg.dataDir;
            environment.interactiveShellInit = ''
              export PM_DIR="${cfg.dataDir}"
            '';

            systemd.services.pm-serve = mkIf (cfg.tokenFile != null) {
              description = "PM Project Memory daemon (single writer API)";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
              restart = "always";
              restartSec = "5";
              serviceConfig = {
                Type = "simple";
                DynamicUser = true;
                StateDirectory = "pm-serve";
                # Token via file (sops-managed), never in nix store or unit text.
                LoadCredential = "pm-token:${cfg.tokenFile}";
              };
              script = ''
                export PM_TOKEN="$(cat "$CREDENTIALS_DIRECTORY/pm-token")"
                export PM_DIR="${cfg.dataDir}"
                exec ${cfg.package}/bin/pm serve --addr "${cfg.listenAddr}"
              '';
            };
          };
        };
    };
}
