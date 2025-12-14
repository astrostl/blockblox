class Blockblox < Formula
  desc "CLI tool for managing Roblox screen time limits"
  homepage "https://github.com/astrostl/blockblox"
  version "0.1.1"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/astrostl/blockblox/releases/download/v0.1.1/blockblox-v0.1.1-darwin-arm64.tar.gz"
    sha256 "1dbba94d48f97281de74b6bdd9dd9d87093d208aba0fe249e3c7dce371353469"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/astrostl/blockblox/releases/download/v0.1.1/blockblox-v0.1.1-darwin-amd64.tar.gz"
    sha256 "fe9712cb93403ee818bcdd6ee5717d7945f16432de16c882a742c63936b90db7"
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
