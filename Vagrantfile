Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"
  config.vm.hostname = "quicknotes-vm"
  config.vm.boot_timeout = 600

  config.vm.network "forwarded_port",
    guest: 8080,
    host: 18080,
    host_ip: "127.0.0.1",
    id: "quicknotes"

  config.vm.synced_folder "./app", "/home/vagrant/quicknotes/app",
    type: "virtualbox",
    SharedFoldersEnableSymlinksCreate: false

  config.vm.provider "virtualbox" do |vb|
    vb.name = "quicknotes-lab5"
    vb.memory = 1024
    vb.cpus = 2
  end

  config.vm.provision "shell", inline: <<-SHELL
    set -eux

    GO_VERSION="1.24.5"
    GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"

    mkdir -p /home/vagrant/quicknotes

    timedatectl set-ntp true || true
    systemctl restart systemd-timesyncd || true
    sleep 5

    apt-get -o Acquire::Check-Valid-Until=false update
    apt-get install -y curl ca-certificates build-essential

    if ! /usr/local/go/bin/go version 2>/dev/null | grep -q "go${GO_VERSION}"; then
      rm -rf /usr/local/go

      if [ -f "/vagrant/${GO_TARBALL}" ]; then
        cp "/vagrant/${GO_TARBALL}" "/tmp/${GO_TARBALL}"
      else
        curl -fL "https://go.dev/dl/${GO_TARBALL}" -o "/tmp/${GO_TARBALL}"
      fi

      tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
      rm -f "/tmp/${GO_TARBALL}"
    fi

    cat >/etc/profile.d/go.sh <<'EOS'
export PATH=/usr/local/go/bin:$PATH
EOS

    ln -sf /usr/local/go/bin/go /usr/local/bin/go
    ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

    chown -R vagrant:vagrant /home/vagrant/quicknotes || true
  SHELL
end
