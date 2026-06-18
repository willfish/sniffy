{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks = {
      url = "github:cachix/git-hooks.nix";
    };
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      pre-commit-hooks,
      ...
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

        preCommitCheck = pre-commit-hooks.lib.${system}.run {
          src = ./.;
          configPath = ".pre-commit-config-nix.yaml";
          default_stages = [ "pre-commit" ];
          hooks = {
            actionlint = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-added-large-files = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-case-conflicts = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-executables-have-shebangs = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-json = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-merge-conflicts = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-shebang-scripts-are-executable = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-toml = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            check-yaml = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            deadnix = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            end-of-file-fixer = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            mixed-line-endings = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            nixfmt-rfc-style = {
              package = pre-commit-hooks.inputs.nixpkgs.legacyPackages.${system}.nixfmt;
              enable = true;
              stages = [ "pre-commit" ];
            };
            shellcheck = {
              enable = true;
              stages = [ "pre-commit" ];
            };
            trim-trailing-whitespace = {
              enable = true;
              stages = [ "pre-commit" ];
            };
          };
        };
      in
      {
        packages.default = sniffy;
        packages.sniffy = sniffy;
        devShells.default = pkgs.mkShell {
          shellHook = ''
            ${preCommitCheck.shellHook}
            export PATH=${pkgs.writeShellScriptBin "pre-commit" ''
              set -euo pipefail

              has_config=false
              for arg in "$@"; do
                case "$arg" in
                  -c|--config|--config=*)
                    has_config=true
                    ;;
                esac
              done

              if [ "$has_config" = true ]; then
                exec ${preCommitCheck.config.package}/bin/pre-commit "$@"
              fi

              if [ "''${1:-}" = "run" ]; then
                shift
                exec ${preCommitCheck.config.package}/bin/pre-commit run --config .pre-commit-config-nix.yaml "$@"
              fi

              exec ${preCommitCheck.config.package}/bin/pre-commit "$@"
            ''}/bin:$PATH
          '';

          buildInputs =
            preCommitCheck.enabledPackages
            ++ (with pkgs; [
              go
              golangci-lint
              gopls
              lint
            ]);
        };
      }
    );
}
