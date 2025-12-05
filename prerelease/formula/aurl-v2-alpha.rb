class AurlV2Alpha < Formula
  desc "URL Command Line Tool with Authentication Support (v2 alpha build)"
  homepage "https://github.com/classmethod/aurl"
  version "2.0.0-alpha20251205"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/classmethod/aurl/releases/download/v#{version}/aurl_Darwin_arm64.tar.gz"
      sha256 "24ea04015fce314299450900bb4ff72a45ce18f02892be7780213853212f37f1"
    else
      url "https://github.com/classmethod/aurl/releases/download/v#{version}/aurl_Darwin_x86_64.tar.gz"
      sha256 "a77ac9dd0c6c057475b3acbf1245d9f4a681b4a2e9458b95987e5d3fac908d68"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/classmethod/aurl/releases/download/v#{version}/aurl_Linux_arm64.tar.gz"
      sha256 "de1023e52412ca7486d2aa1a465464e31f9b0d5f56cde194c0f16878256f4cec"
    elsif Hardware::CPU.intel?
      if Hardware::CPU.is_64_bit?
        url "https://github.com/classmethod/aurl/releases/download/v#{version}/aurl_Linux_x86_64.tar.gz"
        sha256 "c083efe1922e0ebde9f9e02e17dac4b92f0d8efeacffc75353d7300c77304379"
      else
        url "https://github.com/classmethod/aurl/releases/download/v#{version}/aurl_Linux_i386.tar.gz"
        sha256 "c6147577f79bf49fc8e6981148cd6936c5c329146154e49672cd65c4236d730a"
      end
    end
  end

  def install
    bin.install "aurl"
  end

  test do
    system "#{bin}/aurl", "--version"
  end
end