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
            ];

            environment.systemPackages = [ cfg.package ];
            environment.sessionVariables.PM_DIR = cfg.dataDir;
            environment.interactiveShellInit = ''
              export PM_DIR="${cfg.dataDir}"
            '';

            systemd.services.pm-serve = mkIf (cfg.environmentFile != null) {
              description = "PM Project Memory daemon (single writer API)";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];
              serviceConfig = {
                Type = "simple";
                DynamicUser = true;
                StateDirectory = "pm-serve";
                Restart = "always";
                RestartSec = "5";
                EnvironmentFile = cfg.environmentFile;
              };
              script = ''
                export PM_DIR="${cfg.dataDir}"
                exec ${cfg.package}/bin/pm serve --addr "${cfg.listenAddr}"
              '';
            };
          };
        };
    };
}
