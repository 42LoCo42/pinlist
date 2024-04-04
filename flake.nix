{
  inputs.nixpkgs.url = "github:nixos/nixpkgs/080a4a27f206d07724b88da096e27ef63401a504";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
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
        };
      in
      rec {
        packages.default = pkgs.buildGoModule {
          name = "pin";
          src = ./.;
          vendorHash = "sha256-J1IwR/3vXXjxPIKPVOG9hKIUiv8b3sgv/J382eh/bHQ=";

          preBuild = ''
            ${jade}/bin/jade -d jade -writer .
          '';
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
