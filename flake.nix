{
  description = "PM — Project Memory: долговременная память проектов";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };

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

        # Set to empty after first build to auto-detect, or use:
        # nix build .#pm 2>&1 | grep "got:" to get the real hash
        vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";

        meta = {
          description = "Project Memory — долговременная память проектов";
          homepage = "https://git.827482.xyz/xvantz/pm";
          license = pkgs.lib.licenses.mit;
          maintainers = [ "Ivan R. <ivan@xvantz.dev>" ];
          platforms = [ "x86_64-linux" ];
        };
      };
    in
    {
      packages.${system} = {
        inherit pm;
        default = pm;
      };

      apps.${system} = {
        pm = {
          type = "app";
          program = "${pm}/bin/pm";
        };
        pm-mcp = {
          type = "app";
          program = "${pm}/bin/pm-mcp";
        };
      };
    };
}
