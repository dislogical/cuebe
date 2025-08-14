{ pkgs, lib, config, inputs, ... }:

{
  packages = with pkgs; [
    # Dev tools
    git
    pre-commit
    commitizen
  ];

  languages = {
    go.enable = true;
    cue.enable = true;
  };
}
