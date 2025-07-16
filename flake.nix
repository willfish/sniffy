{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { system = system; };

        lint = pkgs.writeScriptBin "lint" ''
          pre-commit run --all-files --show-diff-on-failure
        '';
        update-modules = pkgs.writeScriptBin "update-modules" ''
          cd collector
          go get -u ./...
        '';
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            circleci-cli
            go
            golangci-lint
            gopls
            lint
            serverless
            update-modules
          ];
        };
      });
}
