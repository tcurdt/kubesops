{
  description = "Manage Kubernetes secrets as encrypted dotenv files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule.override { go = pkgs.go_1_24; } {
          pname = "kubesops";
          version = "0.2.0";

          src = ./.;

          vendorHash = "sha256-8osrjy8Paz+a/JF+CJjnzM7N4ftayOXtBwN3Ootu4Ko="; # Will need to update this

          ldflags = [
            "-s"
            "-w"
          ];

          meta = with pkgs.lib; {
            description = "Manage Kubernetes secrets as encrypted dotenv files";
            homepage = "https://github.com/tcurdt/kubesops";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

        apps.default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/kubesops";
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_24
            gopls
            gotools
            sops
            kubernetes-helm
          ];
        };
      }
    );
}
