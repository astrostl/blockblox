class Blockblox < Formula
  desc "CLI tool for managing Roblox screen time limits"
  homepage "https://github.com/astrostl/blockblox"
  version "0.2.1"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/astrostl/blockblox/releases/download/v0.2.1/blockblox-v0.2.1-darwin-arm64.tar.gz"
    sha256 "e271f7f63e553b114799482c79839385ad079c7887b74193126f5d1bfdc2db89"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/astrostl/blockblox/releases/download/v0.2.1/blockblox-v0.2.1-darwin-amd64.tar.gz"
    sha256 "66bab3a8352a6e6962bcfc2301916afa75791e6b90d58ace800fcf8bfc0a49ff"
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
