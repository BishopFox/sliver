# Vagrant file for FreeBSD
#
# Here's an example:
#   GOVERSION=$(go env GOVERSION) GOARCH=$(go env GOARCH) vagrant up
#   vagrant rsync
#   vagrant ssh -c "cd wazero; go test ./..."
#
# Notes on FreeBSD:
# * GitHub Actions doesn’t support FreeBSD, and may never.
# * We could use Travis to run FreeBSD, but it would split our CI config.
# * Using Vagrant directly is easier to debug than vmactions/freebsd-vm.
# * GitHub Actions only supports virtualization on MacOS.
# * GitHub Actions removed vagrant from the image starting with macos-11.
# * Since VirtualBox doesn't work on arm64, freebsd/arm64 is untestable.

Vagrant.configure("2") do |config|
  config.vm.box = "generic/freebsd13"
  config.vm.synced_folder ".", "/home/vagrant/wazero",
    type: "rsync",
    rsync__exclude: ".git/"

  config.vm.provider "virtualbox" do |v|
    v.memory = 1024
    v.cpus = 1
    # To prevent any DNS-related provisioning errors:
    # https://serverfault.com/questions/453185/vagrant-virtualbox-dns-10-0-2-3-not-working/506206
    v.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
  end

  # Similar to `GOVERSION=$(go env GOVERSION) GOARCH=$(go env GOARCH) vagrant provision`
  config.vm.provision "install-golang", type: "shell", run: "once" do |sh|
    sh.env = {
      'GOVERSION': ENV['GOVERSION'],
      'GOARCH': ENV['GOARCH'],
    }
    sh.inline = <<~GOINSTALL
      set -eux -o pipefail
      curl -fsSL "https://dl.google.com/go/${GOVERSION}.freebsd-${GOARCH}.tar.gz" | tar Cxz /usr/local
      cat >> /usr/local/etc/profile <<EOF
export GOROOT=/usr/local/go
export PATH=/usr/local/go/bin:$PATH
EOF
    GOINSTALL
  end
end


