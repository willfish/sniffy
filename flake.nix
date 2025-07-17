{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
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
        pkgs = import nixpkgs { system = system; };

        lint = pkgs.writeScriptBin "lint" ''
          pre-commit run --all-files --show-diff-on-failure
        '';
        sniffy = pkgs.buildGoModule {
          pname = "sniffy";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-5HhG2GAvf6COM4qN0YZU6kQDiXTcrkjj7Itfee2vK6E=";

          meta = with pkgs.lib; {
            description = "A tool for finding unused secrets";
            homepage = "https://github.com/willfish/sniffy";
            license = licenses.mit;
            maintainers = [ maintainers.willfish ];
          };
        };
      in
      {
        packages.default = sniffy;
        packages.sniffy = sniffy;
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            golangci-lint
            gopls
            lint
          ];
        };
      }
    );
}
