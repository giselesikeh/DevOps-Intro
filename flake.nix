{
  description = "Reproducible QuickNotes build with Nix";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in {
      packages = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
          go = pkgs.go_1_24;
          buildGoModule = pkgs.buildGoModule.override { inherit go; };
        in rec {
          quicknotes = buildGoModule {
            pname = "quicknotes";
            version = "lab11";

            src = ./app;

            env.CGO_ENABLED = "0";

            # QuickNotes currently has no external Go module dependencies.
            # buildGoModule requires vendorHash = null when the vendored module tree is empty.
            vendorHash = null;

            ldflags = [ "-s" "-w" ];

            subPackages = [ "." ];

            meta = {
              description = "QuickNotes Go application";
              mainProgram = "quicknotes";
            };
          };

          docker = pkgs.dockerTools.buildImage {
            name = "quicknotes-nix";
            tag = "lab11";

            # Fixed timestamp for deterministic image metadata.
            created = "1970-01-01T00:00:01Z";

            copyToRoot = pkgs.buildEnv {
              name = "quicknotes-image-root";
              paths = [ quicknotes ];
              pathsToLink = [ "/bin" ];
            };

            # QuickNotes writes runtime data relative to the working directory.
            # /tmp is writable for the numeric nonroot user.
            extraCommands = ''
              mkdir -p tmp
              chmod 1777 tmp
            '';

            config = {
              Entrypoint = [ "/bin/quicknotes" ];
              ExposedPorts = {
                "8080/tcp" = {};
              };
              User = "65532:65532";
              WorkingDir = "/tmp";
            };
          };

          default = quicknotes;
        });

      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_24
              pkgs.gopls
              pkgs.golangci-lint
            ];
          };
        });
    };
}
