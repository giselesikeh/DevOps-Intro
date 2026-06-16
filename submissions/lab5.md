# Lab 5 — Virtualization: QuickNotes in a Vagrant VM

## Task 1 — Vagrant Up + Run QuickNotes Inside

### Vagrantfile

```ruby
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
```

### First 10 lines of `vagrant up` output

```text
Bringing machine 'default' up with 'virtualbox' provider...
==> default: Box 'ubuntu/jammy64' could not be found. Attempting to find and install...
    default: Box Provider: virtualbox
    default: Box Version: >= 0
==> default: Loading metadata for box 'ubuntu/jammy64'
    default: URL: https://vagrantcloud.com/api/v2/vagrant/ubuntu/jammy64
==> default: Adding box 'ubuntu/jammy64' (v20241002.0.0) for provider: virtualbox
    default: Downloading: https://vagrantcloud.com/ubuntu/boxes/jammy64/versions/20241002.0.0/providers/virtualbox/unknown/vagrant.box
==> default: Successfully added box 'ubuntu/jammy64' (v20241002.0.0) for 'virtualbox'!
==> default: Importing base box 'ubuntu/jammy64'...
```

### Go version inside the VM

Command:

```bash
vagrant ssh -c "go version"
```

Output:

```text
go version go1.24.5 linux/amd64
```

### QuickNotes build and run inside the VM

Commands:

```bash
vagrant ssh -c "cd /home/vagrant/quicknotes/app && go build -o /tmp/qn"
vagrant ssh -c "pkill qn || true; cd /home/vagrant/quicknotes/app && nohup /tmp/qn > /tmp/qn.log 2>&1 < /dev/null & sleep 2; cat /tmp/qn.log || true; ps aux | grep qn | grep -v grep || true"
```

Output:

```text
2026/06/16 20:43:54 quicknotes listening on :8080 (notes loaded: 7)
vagrant     9982  0.5  0.6 1674872 6808 pts/0    Sl+  20:43   0:00 /tmp/qn
```

### QuickNotes test from inside the VM

Command:

```bash
vagrant ssh -c "curl -i http://127.0.0.1:8080/health || true"
```

Output:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 16 Jun 2026 20:44:21 GMT
Content-Length: 26

{"notes":7,"status":"ok"}
```

Command:

```bash
vagrant ssh -c "curl -i http://127.0.0.1:8080/notes || true"
```

Output:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 16 Jun 2026 20:44:26 GMT
Content-Length: 910

[{"id":6,"title":"trace me","body":"in flight","created_at":"2026-06-12T09:22:36.318372454Z"},{"id":7,"title":"trace me","body":"in flight","created_at":"2026-06-12T09:27:28.846334414Z"},{"id":1,"title":"Welcome to QuickNotes","body":"This is the project you'll containerize, deploy, monitor, and harden across all 10 labs.","created_at":"2026-01-15T10:00:00Z"},{"id":2,"title":"Read app/main.go first","body":"Start by understanding the entry point — env vars, signal handling, graceful shutdown.","created_at":"2026-01-15T10:05:00Z"},{"id":3,"title":"DevOps mantra","body":"If it hurts, do it more often.","created_at":"2026-01-15T10:10:00Z"},{"id":4,"title":"Endpoint cheat-sheet","body":"GET /notes  GET /notes/{id}  POST /notes  DELETE /notes/{id}  GET /health  GET /metrics","created_at":"2026-01-15T10:15:00Z"},{"id":5,"title":"hello","body":"first POST","created_at":"2026-06-05T08:21:48.7039563Z"}]
```

### QuickNotes test from host through port forwarding

Command:

```bash
curl -i http://localhost:18080/health || true
```

Output:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 16 Jun 2026 20:44:48 GMT
Content-Length: 26

{"notes":7,"status":"ok"}
```

Command:

```bash
curl -i http://localhost:18080/notes || true
```

Output:

```text
HTTP/1.1 200 OK
Content-Type: application/json
Date: Tue, 16 Jun 2026 20:44:48 GMT
Content-Length: 910

[{"id":1,"title":"Welcome to QuickNotes","body":"This is the project you'll containerize, deploy, monitor, and harden across all 10 labs.","created_at":"2026-01-15T10:00:00Z"},{"id":2,"title":"Read app/main.go first","body":"Start by understanding the entry point — env vars, signal handling, graceful shutdown.","created_at":"2026-01-15T10:05:00Z"},{"id":3,"title":"DevOps mantra","body":"If it hurts, do it more often.","created_at":"2026-01-15T10:10:00Z"},{"id":4,"title":"Endpoint cheat-sheet","body":"GET /notes  GET /notes/{id}  POST /notes  DELETE /notes/{id}  GET /health  GET /metrics","created_at":"2026-01-15T10:15:00Z"},{"id":5,"title":"hello","body":"first POST","created_at":"2026-06-05T08:21:48.7039563Z"},{"id":6,"title":"trace me","body":"in flight","created_at":"2026-06-12T09:22:36.318372454Z"},{"id":7,"title":"trace me","body":"in flight","created_at":"2026-06-12T09:27:28.846334414Z"}]
```

### Design questions

#### a) Synced folders

I used the `virtualbox` synced folder type to mount the host `./app` directory into `/home/vagrant/quicknotes/app` inside the VM. I chose this because it works naturally with VirtualBox on Windows and lets the VM see the application files without manually copying them. The trade-off is that VirtualBox shared folders can be slower and can have different permission behavior compared with a native Linux filesystem or `rsync`.

#### b) NAT vs Bridged vs Host-only

The VM uses NAT networking with a forwarded port. NAT is safer for this course exercise because the VM is not directly placed on the local network. Binding the forwarded port to `127.0.0.1` means only my host machine can access QuickNotes on port `18080`, while a bridged interface would expose the VM more directly to other machines on the same LAN.

#### c) Provisioning options

I used the Vagrant `shell` provisioner. It is enough for this lab because the setup is simple: update apt metadata, install basic packages, install Go, and prepare the synced application directory. Tools like Ansible, Puppet, or Chef are better for larger infrastructure, but shell provisioning is simple, readable, and appropriate for this single VM lab.

#### d) Why pin Go to `1.24.5` instead of `1.24`?

I pinned Go to `1.24.5` to make the environment reproducible. A broad version like `1.24` can change over time as new patch releases are published, so two students running the lab at different times could get slightly different Go toolchains. A specific point release makes the VM setup deterministic and easier to debug.

---

## Task 2 — Snapshots: Save, Break, Restore

### Snapshot saved

Command:

```bash
vagrant snapshot save clean-quicknotes
vagrant snapshot list
```

Output:

```text
==> default: Snapshotting the machine as 'clean-quicknotes'...
==> default: Snapshot saved! You can restore the snapshot at any time by
==> default: using `vagrant snapshot restore`. You can delete it using
==> default: `vagrant snapshot delete`.
==> default:
clean-quicknotes
```

### Break the VM deliberately

I deliberately broke the VM by deleting the Go installation and the Go symlinks. This is destructive because the `go` command is no longer available inside the VM after this step.

Command:

```bash
vagrant ssh -c "sudo rm -rf /usr/local/go /usr/local/bin/go /usr/local/bin/gofmt"
```

### Verify the VM is broken

Command:

```bash
vagrant ssh -c "go version"
```

Output:

```text
bash: line 1: go: command not found
```

### Restore the snapshot and time the restore

Command:

```bash
time vagrant snapshot restore clean-quicknotes
```

Output:

```text
==> default: Forcing shutdown of VM...
==> default: Restoring the snapshot 'clean-quicknotes'...
==> default: Checking if box 'ubuntu/jammy64' version '20241002.0.0' is up to date...
==> default: Resuming suspended VM...
==> default: Booting VM...
==> default: Waiting for machine to boot. This may take a few minutes...
    default: SSH address: 127.0.0.1:2222
    default: SSH username: vagrant
    default: SSH auth method: private key
==> default: Machine booted and ready!
==> default: Machine already provisioned. Run `vagrant provision` or use the `--provision`
==> default: flag to force provisioning. Provisioners marked to run always will still run.

real    0m40.315s
user    0m0.000s
sys     0m0.031s
```

### Verify recovery after restore

Commands:

```bash
vagrant ssh -c "go version"
vagrant ssh -c "curl -s http://127.0.0.1:8080/health || true"
curl -s http://localhost:18080/health || true
```

Output:

```text
go version go1.24.5 linux/amd64
{"notes":7,"status":"ok"}
{"notes":7,"status":"ok"}
```

### Design questions

#### e) Why are snapshots not backups?

Snapshots are not backups because they usually live on the same physical disk or VM storage as the original VM. If the host disk fails, the VM is deleted, or the VM storage is corrupted, both the VM and its snapshots can be lost together. A real backup should be stored separately from the VM and should survive failure of the original machine or disk.

#### f) What does copy-on-write mean for disk usage?

Copy-on-write means the snapshot does not immediately duplicate the whole VM disk. Instead, VirtualBox keeps the original disk state and stores only the changed blocks after the snapshot is taken. This makes one snapshot cheap at first, but 10 snapshots can gradually consume much more disk space as more blocks change over time.

#### g) When is snapshotting an antipattern?

Snapshotting becomes an antipattern when a VM is kept running on a long chain of snapshots. Long snapshot chains can slow down disk operations, make recovery more complicated, and increase the chance of storage problems. Snapshots should be temporary checkpoints, not a long-term backup or versioning strategy.


---

## Bonus Task — VM vs Container Resource Baseline

### B.1: Vagrant VM measurements

Commands used:

```bash id="l20vtt"
time vagrant halt
time vagrant up
vagrant ssh -c "free -h"
vagrant ssh -c "ps -A --no-headers | wc -l"
du -sh "/c/Users/smartech/VirtualBox VMs/quicknotes-lab5"
```

VM cold boot output:

```text id="sfu9z3"
real    0m52.203s
user    0m0.015s
sys     0m0.031s
```

VM idle RAM output:

```text id="j2mvk9"
               total        used        free      shared  buff/cache   available
Mem:           957Mi       179Mi       492Mi        0.0Ki       285Mi       629Mi
Swap:             0B          0B          0B
```

VM process count output:

```text id="vs6der"
104
```

VM disk size output:

```text id="osb08t"
3.4G    /c/Users/smartech/VirtualBox VMs/quicknotes-lab5
```

### B.2: Docker container measurements

Because Docker Hub timed out while pulling `golang:1.24`, I used an existing local `ubuntu:latest` image and mounted the same Go 1.24.5 tarball used for the VM. The container still runs the same QuickNotes application and exposes it on host port `28080`.

Command used to run the container:

```bash id="p892nc"
docker rm -f quicknotes-lab5-docker 2>/dev/null || true

MSYS_NO_PATHCONV=1 docker run -d --name quicknotes-lab5-docker \
  -p 28080:8080 \
  -v "$(pwd -W)/app:/src" \
  -v "$(pwd -W)/go1.24.5.linux-amd64.tar.gz:/tmp/go1.24.5.linux-amd64.tar.gz" \
  -w /src \
  ubuntu:latest \
  sh -c 'rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go1.24.5.linux-amd64.tar.gz && export PATH=/usr/local/go/bin:$PATH && go build -o /tmp/qn && /tmp/qn'
```

Container health check:

```bash id="jbg30c"
curl -s http://localhost:28080/health
```

Output:

```text id="l16dec"
{"notes":7,"status":"ok"}
```

Docker cold start commands:

```bash id="ezxrvc"
time docker stop quicknotes-lab5-docker
time docker start quicknotes-lab5-docker
```

Docker stop output:

```text id="a6a0qo"
real    0m1.632s
user    0m0.046s
sys     0m0.122s
```

Docker start output:

```text id="berfyd"
real    0m0.610s
user    0m0.061s
sys     0m0.137s
```

Docker idle RAM output:

```text id="gclxex"
CONTAINER ID   NAME                     CPU %     MEM USAGE / LIMIT     MEM %     NET I/O        BLOCK I/O    PIDS
3b6f77671517   quicknotes-lab5-docker   0.00%     61.02MiB / 7.696GiB   0.77%     1.7kB / 574B   0B / 298MB   9
```

Docker process count output:

```text id="j2u4xp"
2
```

Docker image size output:

```text id="p32zey"
ubuntu:latest 119MB
```

Docker container size output:

```text id="i7vk2s"
CONTAINER ID   IMAGE           COMMAND                  CREATED         STATUS         PORTS                                           NAMES                    SIZE
3b6f77671517   ubuntu:latest   "sh -c 'rm -rf /usr/…"   8 minutes ago   Up 6 minutes   0.0.0.0:28080->8080/tcp, [::]:28080->8080/tcp   quicknotes-lab5-docker   400MB (virtual 488MB)
```

### B.3: VM vs Docker comparison

| Dimension     | Vagrant VM | Docker container |
| ------------- | ---------: | ---------------: |
| Cold start    |   52.203 s |          0.610 s |
| Idle RAM      |    179 MiB |        61.02 MiB |
| On-disk size  |     3.4 GB |           400 MB |
| Process count |        104 |                2 |

### Reflection

The biggest difference was the cold start time: the Docker container restarted in less than a second, while the VM needed about 52 seconds to boot. The RAM and process count also show that the VM runs a full guest operating system, while the container only runs the application process and its small wrapper process. A VM is the right tool when stronger isolation, a full OS boundary, or a different kernel/userspace environment is needed. A container is a better fit for stateless microservices because it starts quickly, uses less memory, and has a much smaller disk footprint. These numbers explain why containers became popular for stateless services between 2014 and 2020: they allowed faster deployment and higher density than full VMs.

