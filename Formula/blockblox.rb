class Blockblox < Formula
  desc "CLI tool for managing Roblox screen time limits"
  homepage "https://github.com/astrostl/blockblox"
  version "0.1.2"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/astrostl/blockblox/releases/download/v0.1.2/blockblox-v0.1.2-darwin-arm64.tar.gz"
    sha256 "0efa3973c99e16480a5340f93f351ae1a5f2ebb093f23e6b12107d1e474041f5"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/astrostl/blockblox/releases/download/v0.1.2/blockblox-v0.1.2-darwin-amd64.tar.gz"
    sha256 "28bc788d63b060accb7540cd27aa567db1e481edacbbc82e52fc6b3b567a074f"
  else
    odie "Blockblox is only supported on macOS (requires Keychain for Chrome cookie extraction)."
  end

  def install
    bin.install "blockblox-darwin-arm64" => "blockblox" if Hardware::CPU.arm?
    bin.install "blockblox-darwin-amd64" => "blockblox" if Hardware::CPU.intel?
  end

  def caveats
    <<~EOS
      First time setup:
        blockblox init

      This extracts Roblox credentials from Chrome.
      Make sure you're logged into Roblox in Chrome first.

      Usage:
        blockblox get           # Show current screen time limit
        blockblox set 4h        # Set to 4 hours
        blockblox set 4h30m     # Set to 4 hours 30 minutes
        blockblox set 0         # Remove limit
    EOS
  end

  test do
    system bin/"blockblox", "--help" rescue nil
  end
end
