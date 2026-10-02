{
    description = "go-north development environment";

    inputs = {
        nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
        nixpkgs-go.url = "github:NixOS/nixpkgs/nixos-24.11";
        flake-utils.url = "github:numtide/flake-utils";
        treefmt-nix = {
            url = "github:numtide/treefmt-nix";
            inputs.nixpkgs.follows = "nixpkgs";
        };
    };

    outputs =
        {
            self,
            nixpkgs,
            nixpkgs-go,
            flake-utils,
            treefmt-nix,
        }:
        flake-utils.lib.eachDefaultSystem (
            system:
            let
                pkgs = nixpkgs.legacyPackages.${system};
                goPkgs = nixpkgs-go.legacyPackages.${system};
                treefmtEval = treefmt-nix.lib.evalModule pkgs ./treefmt.nix;
                goCheck =
                    name: go:
                    (pkgs.buildGoModule.override { inherit go; }) {
                        pname = name;
                        version = "0";
                        src = self;
                        vendorHash = "sha256-RKhBFvK/+EYmCEWiYxbZ2m5xNQGrtHXQofxlIr8RwiM=";
                        env.CGO_ENABLED = "1";
                        preCheck = ''
                            go vet ./...
                        '';
                        checkFlags = [ "-race" ];
                    };
            in
            {
                formatter = treefmtEval.config.build.wrapper;

                checks = {
                    formatting = treefmtEval.config.build.check self;
                    go = goCheck "go-north-check" pkgs.go;
                    minimum-go = goCheck "go-north-go-1.23-check" goPkgs.go_1_23;
                };

                devShells.default = pkgs.mkShell {
                    packages = [
                        goPkgs.go_1_23
                        pkgs.gopls
                        pkgs.go-tools
                        pkgs.delve
                        pkgs.gnumake
                        treefmtEval.config.build.wrapper
                    ];
                };
            }
        );
}
