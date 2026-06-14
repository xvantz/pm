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

            # After first build error with the fake hash, run:
            #   nix build .#pm 2>&1 | grep "got:" | head -1
            # and paste the hash here.
            vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

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
              defaultText = literalExpression "self.packages.\${pkgs.stdenv.hostPlatform.system}.pm";
              description = "pm package to use";
            };

            dataDir = mkOption {
              type = types.str;
              default = "/home/xvantz/Documents/pm";
              description = "Directory for PM project data (YAML store). Used as PM_DIR.";
            };
          };

          config = mkIf cfg.enable {
            environment.systemPackages = [ cfg.package ];
          };
        };
    };
}
