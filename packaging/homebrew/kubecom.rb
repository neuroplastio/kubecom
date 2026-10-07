# Rendered by kubecom's release workflow from packaging/homebrew/kubecom.rb in
# github.com/neuroplastio/kubecom; change it there.
#
# A launcher plus a seed build (D295/D299): the formula installs the thin
# launcher and a complete kubecom binary. On first run the wrapper seeds the
# launcher's home (~/.local/kubecom) from that binary, so kubecom works with no
# network and survives a channel outage; `kubecom update` then updates the copy
# in the home, outside the package manager.
class Kubecom < Formula
  desc "A fast, keyboard-driven, zero-deploy Kubernetes TUI"
  homepage "https://github.com/neuroplastio/kubecom"
  version "@VERSION@"
  license "Apache-2.0"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom-launcher_darwin_arm64"
    sha256 "@LAUNCHER_DARWIN_ARM64@"
    resource "seed" do
      url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom_darwin_arm64"
      sha256 "@SEED_DARWIN_ARM64@"
    end
  elsif OS.mac?
    url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom-launcher_darwin_amd64"
    sha256 "@LAUNCHER_DARWIN_AMD64@"
    resource "seed" do
      url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom_darwin_amd64"
      sha256 "@SEED_DARWIN_AMD64@"
    end
  elsif Hardware::CPU.arm?
    url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom-launcher_linux_arm64"
    sha256 "@LAUNCHER_LINUX_ARM64@"
    resource "seed" do
      url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom_linux_arm64"
      sha256 "@SEED_LINUX_ARM64@"
    end
  else
    url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom-launcher_linux_amd64"
    sha256 "@LAUNCHER_LINUX_AMD64@"
    resource "seed" do
      url "https://github.com/neuroplastio/kubecom/releases/download/@VERSION@/kubecom_linux_amd64"
      sha256 "@SEED_LINUX_AMD64@"
    end
  end

  def install
    libexec.install Dir["kubecom-launcher_*"].first => "kubecom-launcher"
    resource("seed").stage do
      (libexec/"kubecom-seed").install Dir["kubecom_*"].first => "kubecom"
    end
    # The release assets are bare binaries, which brew downloads without the
    # executable bit.
    chmod 0755, [libexec/"kubecom-launcher", libexec/"kubecom-seed/kubecom"]

    # bin/kubecom seeds the launcher's home from the packaged build on first
    # run, then hands over. The home is enlaunch's default (~/.local/kubecom);
    # an already-updated install is left alone.
    (bin/"kubecom").write <<~SH
      #!/bin/sh
      home="$HOME/.local/kubecom"
      if [ ! -e "$home/bin/kubecom" ]; then
        mkdir -p "$home/builds/@COMMIT@"
        cp "#{opt_libexec}/kubecom-seed/kubecom" "$home/builds/@COMMIT@/kubecom"
        chmod +x "$home/builds/@COMMIT@/kubecom"
        ln -sfn "builds/@COMMIT@" "$home/bin"
      fi
      exec "#{opt_libexec}/kubecom-launcher" "$@"
    SH
    chmod 0755, bin/"kubecom"
  end

  test do
    ENV["HOME"] = testpath.to_s
    assert_match "kubecom", shell_output("#{bin}/kubecom version")
  end
end
