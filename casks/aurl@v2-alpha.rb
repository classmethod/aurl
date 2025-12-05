cask "aurl@v2-alpha" do
  version :latest
  sha256 :no_check

  # Always points to the latest v2 alpha release
  # Update this URL manually when you publish a new v2 alpha
  url "https://github.com/classmethod/aurl/releases/download/v2.0.0-alpha20251205/aurl_Darwin_x86_64.tar.gz"
  name "aurl (v2 alpha)"
  desc "URL Command Line Tool with Authentication Support (v2 alpha build)"
  homepage "https://github.com/classmethod/aurl"

  binary "aurl"

  postflight do
    system_command "#{staged_path}/aurl",
                   args: ["--version"]
  end
end
