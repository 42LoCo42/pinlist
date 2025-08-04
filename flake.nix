{
  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        inherit (pkgs.lib.fileset) toSource unions;
      in
      rec {
        packages.default = pkgs.buildGoModule rec {
          pname = "pinlist";
          version = "1.1.0";
          src = toSource {
            root = ./.;
            fileset = unions [
              ./go.mod
              ./go.sum
              ./main.go
              ./static
            ];
          };

          ldflags = [ "-s" "-w" ];
          vendorHash = "sha256-BPQk2IFemxrElWkPyv1Y+RYNIFQeF+0ofgu97Buo0L4=";

          meta = {
            description = "Super simple text/link pinlist tool";
            homepage = "https://github.com/42LoCo42/pinlist";
            mainProgram = pname;
          };
        };

        devShells.default = pkgs.mkShell {
          inputsFrom = [ packages.default ];
          packages = with pkgs; [
            air
            sqlite-interactive
          ];
        };
      });
}
