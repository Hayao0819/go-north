{
    description = "go-north development environment";

    inputs = {
        nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
        nixpkgs-go.url = "github:NixOS/nixpkgs/nixos-24.05";
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
                    name: go: commands:
                    pkgs.stdenv.mkDerivation {
                        inherit name;
                        src = self;
                        nativeBuildInputs = [ go ];
                        dontConfigure = true;
                        buildPhase = ''
                            runHook preBuild
                            export HOME="$TMPDIR/home"
                            export GOCACHE="$TMPDIR/go-cache"
                            export GOPATH="$TMPDIR/go"
                            mkdir -p "$HOME" "$GOCACHE" "$GOPATH"
                            ${commands}
                            runHook postBuild
                        '';
                        installPhase = ''
                            mkdir -p "$out"
                        '';
                    };
            in
            {
                formatter = treefmtEval.config.build.wrapper;

                checks = {
                    formatting = treefmtEval.config.build.check self;
                    go = goCheck "go-north-check" pkgs.go ''
                        go vet ./...
                        CGO_ENABLED=1 go test -race ./...
                    '';
                    minimum-go = goCheck "go-north-go-1.21-check" goPkgs.go_1_21 ''
                        go vet ./...
                        CGO_ENABLED=1 go test -race ./...
                    '';
                };

                devShells.default = pkgs.mkShell {
                    packages = [
                        goPkgs.go_1_21
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
