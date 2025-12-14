class Blockblox < Formula
  desc "CLI tool for managing Roblox screen time limits"
  homepage "https://github.com/astrostl/blockblox"
  version "0.1.0"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/astrostl/blockblox/releases/download/v0.1.0/blockblox-v0.1.0-darwin-arm64.tar.gz"
    sha256 "62128faf417217dd094f8898b1dfbc3f0c9b8c14469c8655d6e65539dcd008e9"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/astrostl/blockblox/releases/download/v0.1.0/blockblox-v0.1.0-darwin-amd64.tar.gz"
    sha256 "5aa4df41c06b1fcc01ae3b5963bf24cb8700e6fccb8af8a77e6309c12e83e9b0"
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
