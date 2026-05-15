{ pkgs, lib, ... }:

{
  # Needed at link-time for go-keyring on Linux (libsecret / D-Bus)
  packages = lib.optionals pkgs.stdenv.isLinux (
    with pkgs;
    [
      libsecret
      pkg-config
      dbus
    ]
  );

  # ── Environment variables ─────────────────────────────────────────────────
  env = {
    # CGO is required for go-keyring on Linux (links against libsecret).
    CGO_ENABLED = "1";
  };

  # ── Shell welcome message ─────────────────────────────────────────────────
  enterShell = ''
    echo ""
    echo "kanboard-cli dev environment  (go $(go version | awk '{print $3}'))"
    echo ""
    echo "just recipes:"
    echo "  just build            build binary with version ldflags"
    echo "  just run <args>       build + run (e.g. just run task list -p 1)"
    echo "  just test             go test ./..."
    echo "  just test-race        go test -race ./..."
    echo "  just vendor           go mod tidy + go mod vendor"
    echo "  just clean            remove build artefacts"
    echo "  just nix-build        nix build .#default"
    echo "  just nix-run <args>   nix run .#default -- <args>"
    echo "  just version          print version from built binary"
    echo ""
    echo "Run 'just' or 'just --list' for the full list."
    echo ""
  '';
}
