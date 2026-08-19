{
  description = "atlas - Agent first CLI for Atlassian products";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
      gomod2nix,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ gomod2nix.overlays.default ];
        };

        releasePleaseManifest = builtins.fromJSON (builtins.readFile ./.github/.release-please-manifest.json);
        version = releasePleaseManifest.".";

        # Add platform-specific build inputs here (e.g., CGO deps)
        buildInputs = [ ];

        # macOS-specific build inputs for CGO
        darwinBuildInputs = pkgs.lib.optionals pkgs.stdenv.hostPlatform.isDarwin [
          pkgs.apple-sdk_15
        ];
      in
      {
        packages.default = pkgs.buildGoApplication {
          pname = "atlas";
          inherit version;
          src = ./.;
          modules = ./gomod2nix.toml;
          go = pkgs.go_1_25;

          buildInputs = buildInputs ++ darwinBuildInputs;

          # Set CGO_ENABLED=1 if you need CGO
          CGO_ENABLED = 0;

          ldflags = [
            "-s"
            "-w"
            "-X github.com/mholtzscher/atlas/cmd.Version=${version}"
          ];

          meta = with pkgs.lib; {
            description = "Agent first CLI for Atlassian products";
            homepage = "https://github.com/mholtzscher/atlas";
            license = licenses.mit;
            mainProgram = "atlas";
            platforms = platforms.all;
          };
        };

        formatter = pkgs.nixfmt-rfc-style;

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go_1_25
            pkgs.gopls
            pkgs.gotools
            pkgs.gomod2nix
            pkgs.just
            pkgs.cruft
          ]
          ++ buildInputs
          ++ darwinBuildInputs;

          # Set CGO_ENABLED="1" if you need CGO
          CGO_ENABLED = "0";
        };

        devShells.ci = pkgs.mkShell {
          buildInputs = [
            pkgs.go_1_25
            pkgs.just
          ]
          ++ buildInputs
          ++ darwinBuildInputs;

          CGO_ENABLED = "0";
        };
      }
    );
}
