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
    go.enable = true;
    nix.enable = true;
    shell.enable = true;
    javascript = {
      enable = true;
      pnpm.enable = true;
    };
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
