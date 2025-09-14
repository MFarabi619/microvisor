# This is your nixos configuration.
# For home configuration, see /modules/home/*
{ flake, ... }:
{
  imports = [
    # flake.inputs.self.nixosModules.default
    ./services
    ./console.nix
    ./environment.nix
    ./fonts.nix
    ./hardware.nix
    ./myusers.nix
    ./networking.nix
    ./nix.nix
    ./security.nix
    ./systemd.nix
    ./time.nix
    ./virtualisation.nix
  ];
}
