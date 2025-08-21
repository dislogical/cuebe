# Copyright © 2025 Colden Cullen
# SPDX-License-Identifier: MIT

{ pkgs, lib, config, inputs, ... }:

{
  packages = with pkgs; [
    # Dev tools
    git
    pre-commit
    commitizen
    hawkeye
    cobra-cli
    mkdocs
    mike
    python313Packages.mkdocs-material
  ];

  languages = {
    go.enable = true;
    go.enableHardeningWorkaround = true;
    cue.enable = true;
  };

  outputs = {
    cuebe = pkgs.buildGoModule rec {
      name = "cuebe";
      src = ./.;

      vendorHash = lib.fakeHash;

      GOFLAGS = [
        "-o=${name}"
      ];
    };
  };
}
