class Blockblox < Formula
  desc "CLI tool for managing Roblox screen time limits"
  homepage "https://github.com/astrostl/blockblox"
  version "0.2.0"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/astrostl/blockblox/releases/download/v0.2.0/blockblox-v0.2.0-darwin-arm64.tar.gz"
    sha256 "3d0778a54e621d84360283ae4412ddecea8211f06eab56ac833d8a2912796f4a"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/astrostl/blockblox/releases/download/v0.2.0/blockblox-v0.2.0-darwin-amd64.tar.gz"
    sha256 "23552c4b2ec46b01e26ba1c207bdf991580d9df71bf00e8a79a41aff9585c802"
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
