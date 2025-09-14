# This is your nix-darwin configuration.
# For home configuration, see /modules/home/*
{
  imports = [
    ./myusers.nix
    ./documentation.nix
    ./environment.nix
    ./homebrew.nix
    ./networking.nix
    ./power.nix
  ];
}
