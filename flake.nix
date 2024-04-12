{
  inputs.nixpkgs.url = "github:nixos/nixpkgs/080a4a27f206d07724b88da096e27ef63401a504";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        inherit (pkgs.lib) getExe;

        jade = pkgs.buildGoModule rec {
          pname = "jade";
          version = "2a11849";
          src = pkgs.fetchFromGitHub {
            owner = "joker";
            repo = pname;
            rev = version;
            hash = "sha256-spydTTkaIyQKfiGCP2lzUFN16DQElQnPGuJVzcvq5FY=";
          };
          vendorHash = "sha256-n4WeygbreZQwp6immjWzTT/IjAedZv27joSVU4wBPWI=";
          meta.mainProgram = "jade";
        };
      in
      rec {
        packages.default = pkgs.buildGoModule rec {
          name = "pinlist";
          src = ./.;
          vendorHash = "sha256-IoLVAfAY+678IVp6yszj62NjBqPruomnHyDd8RqtFPs=";

          preBuild = ''
            ${getExe jade} -d jade -writer .
          '';

          CGO_ENABLED = "0";
          stripAllList = [ "bin" ];
          meta.mainProgram = "pinlist";
        };

        packages.image = pkgs.dockerTools.buildImage {
          name = "pinlist";
          tag = "latest";
          config.Cmd = [ (getExe packages.default) ];
        };

        devShells.default = pkgs.mkShell {
          inputsFrom = [ packages.default ];
          packages = with pkgs; [
            gopls
            jade
            nodePackages.prettier
          ];
        };
      });
}
