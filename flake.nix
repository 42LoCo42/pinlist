{
  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { inherit system; }; in
      rec {
        packages.default = pkgs.buildGoModule rec {
          name = "pinlist";
          src = ./.;
          vendorHash = "sha256-6LEpelMU1eGbjYHQ7LjZqZU/lUGc3tlRen8NgT5vStg=";

          CGO_ENABLED = "0";
          stripAllList = [ "bin" ];
          meta.mainProgram = "pinlist";
        };

        devShells.default = pkgs.mkShell {
          inputsFrom = [ packages.default ];
          packages = with pkgs; [
            air
            nodePackages.prettier
            sqlite-interactive
          ];
        };
      });
}
