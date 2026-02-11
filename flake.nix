{
  description = "addt development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go (nixos-unstable typically has recent versions)
            go
            gnumake
            git

            # Container runtimes for testing
            podman
            docker
          ];

          shellHook = ''
            echo "addt development environment"
            echo "Go version: $(go version)"
            echo ""
            echo "Run 'make help' to see available targets"
          '';
        };
      }
    );
}
