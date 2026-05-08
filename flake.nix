{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in {
      # ── package ───────────────────────────────────────────────────────────
      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system}; in {
          default =
            let
              appVersion = "0.3.0";
              module = "github.com/tu-graz/kanboard-cli";
            in
            pkgs.buildGoModule {
              pname = "kanboard-cli";
              version = appVersion;
              src = ./.;

              vendorHash = "sha256-rUmZHeZLEpksCA7bhtP+8uby3NdCQ4PzUhBQzVlrgD0=";

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
