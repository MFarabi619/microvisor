{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:

{
  # packages = with pkgs; [ ];

  languages = {
    nix.enable = true;
    go.enable = true;
  };

  scripts = {
    hello.exec = ''
      echo hello
    '';
  };

  enterShell = ''
    hello
  '';

  # tasks = {
  #   "myproj:setup".exec = "mytool build";
  #   "devenv:enterShell".after = [ "myproj:setup" ];
  # };

  enterTest = ''
    echo "Running tests"
    git --version | grep --color=auto "${pkgs.git.version}"
  '';

  # git-hooks.hooks.shellcheck.enable = true;
}
