# This file is a reference template for a Homebrew formula.
# It is NOT used by GoReleaser unless you configure Homebrew publishing.

class Pachim < Formula
  desc "Deploy your projects to Pachim servers"
  homepage "https://pachim.sh"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/pachimsh/pachim-cli/releases/download/v#{version}/pachim_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_arm do
      url "https://github.com/pachimsh/pachim-cli/releases/download/v#{version}/pachim_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/pachimsh/pachim-cli/releases/download/v#{version}/pachim_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_arm do
      url "https://github.com/pachimsh/pachim-cli/releases/download/v#{version}/pachim_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "pachim"
  end

  test do
    assert_match "pachim", shell_output("#{bin}/pachim --version")
  end
end

