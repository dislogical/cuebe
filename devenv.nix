# Copyright © 2025 Colden Cullen

{ pkgs, lib, config, inputs, ... }:

{
  packages = with pkgs; [
    # Dev tools
    git
    pre-commit
    commitizen
    hawkeye
  ];

  languages = {
    go.enable = true;
    cue.enable = true;
  };
}
