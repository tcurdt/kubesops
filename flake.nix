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
        packages.default = pkgs.buildGoModule {
          pname = "kubesops";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # Will need to update this

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
            go
            gopls
            gotools
            sops
            kubernetes-helm
          ];
        };
      }
    );
}
