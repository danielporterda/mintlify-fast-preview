{
  description = "Fast clean-room Mintlify-style preview server";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.buildGo123Module {
            pname = "mintlify-fast-preview";
            version = "0.0.0";
            src = self;
            vendorHash = null;
            subPackages = [ "cmd/mintfast" ];
          };
        });

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/mintfast";
        };
      });

      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_23
              pkgs.gotools
              pkgs.gopls
              pkgs.python312Packages.playwright
            ];
          };
        });
    };
}
