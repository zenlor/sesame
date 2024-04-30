{
  description = "";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.11";
    flakelight.url = "github:nix-community/flakelight";
    flakelight.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = {flakelight, ...}@inputs: flakelight ./. {
    inherit inputs;
    systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" ];

    devShell.packages = pkgs: [
      pkgs.go
      pkgs.gopls
    ];
  };
}
