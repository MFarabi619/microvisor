# This is your nixos configuration.
# For home configuration, see /modules/home/*
{ flake, ... }:
{
  imports = [
    # flake.inputs.self.nixosModules.default
    ./myusers.nix
    ./nix.nix
  ];
}
