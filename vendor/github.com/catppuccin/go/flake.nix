{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
    whiskers.url = "github:catppuccin/whiskers";
  };

  outputs =
    { self
    , flake-utils
    , nixpkgs
    , whiskers
    ,
    } @ inputs:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        whiskers = inputs.whiskers.packages.${system}.default;
      in
      {
        devShells.default = pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            go
            gopls
            whiskers
            (writeScriptBin "update" ''
              whiskers go.tera
              go fmt
            '')
          ];
        };
      }
    );
}
