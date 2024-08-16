{
  inputs.nixpkgs.url = "github:nixos/nixpkgs/080a4a27f206d07724b88da096e27ef63401a504";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { inherit system; }; in
      rec {
        packages.default = pkgs.buildGoModule rec {
          name = "pinlist";
          src = ./.;
          vendorHash = "sha256-IoLVAfAY+678IVp6yszj62NjBqPruomnHyDd8RqtFPs=";

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
