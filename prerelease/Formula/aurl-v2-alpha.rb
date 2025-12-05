class AurlV2Alpha < Formula
  desc "URL Command Line Tool with Authentication Support (v2 alpha build, Linux only)"
  homepage "https://github.com/classmethod/aurl"
  version "2.0.0-alpha20251205"

  # This formula is for Linux only. macOS users should use the Cask instead.
  depends_on :linux

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

  def install
    bin.install "aurl"
  end

  test do
    system "#{bin}/aurl", "--version"
  end
end