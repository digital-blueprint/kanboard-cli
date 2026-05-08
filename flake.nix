{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    devenv.url = "github:cachix/devenv";
  };

  outputs = { self, nixpkgs, devenv, ... }@inputs:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in {
      # ── devShell ──────────────────────────────────────────────────────────
      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system}; in {
          default = devenv.lib.mkShell {
            inherit inputs pkgs;
            modules = [ ./devenv.nix ];
          };
        }
      );

      # ── package ───────────────────────────────────────────────────────────
      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system}; in {
          default =
            let
              # Read the version from git tags when building inside a git tree;
              # fall back to the flake rev when built from an archive.
              appVersion = if (self ? rev)
                then builtins.substring 0 8 self.rev
                else "dev";
              module = "github.com/tu-graz/kanboard-cli";
            in
            pkgs.buildGoModule {
              pname = "kanboard-cli";
              version = appVersion;
              src = ./.;

              # vendor/ directory is committed; Nix uses it directly.
              vendorHash = null;

              ldflags = [
                "-s" "-w"
                "-X ${module}/internal/version.Version=${appVersion}"
                "-X ${module}/internal/version.Commit=${if (self ? rev) then self.rev else "dirty"}"
                "-X ${module}/internal/version.Date=1970-01-01T00:00:00Z"
              ];

              # libsecret is needed at link time on Linux for go-keyring.
              buildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux
                [ pkgs.libsecret ];

              nativeBuildInputs = [
                pkgs.installShellFiles
              ] ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
                pkgs.pkg-config
              ];

              postInstall = ''
                installShellCompletion --cmd kanboard-cli \
                  --bash <($out/bin/kanboard-cli completion bash) \
                  --zsh  <($out/bin/kanboard-cli completion zsh) \
                  --fish <($out/bin/kanboard-cli completion fish)
              '';

              meta = {
                description = "CLI client for Kanboard";
                homepage    = "https://github.com/digital-blueprint/kanboard-cli";
                license     = pkgs.lib.licenses.gpl3Only;
                mainProgram = "kanboard-cli";
              };
            };
        }
      );

      defaultPackage = forAllSystems (system: self.packages.${system}.default);
    };
}
