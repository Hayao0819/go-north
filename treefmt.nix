{ ... }:
{
    projectRootFile = "flake.nix";

    programs = {
        gofmt.enable = true;
        nixfmt = {
            enable = true;
            indent = 4;
        };
        yamlfmt = {
            enable = true;
            settings.formatter = {
                type = "basic";
                indent = 4;
            };
        };
    };

    settings.global.excludes = [
        ".direnv/**"
    ];
}
