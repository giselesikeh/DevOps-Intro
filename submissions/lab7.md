# Lab 7 — Configuration Management: Deploy QuickNotes via Ansible

## Environment

I used the Lab 5 VirtualBox VM as the managed node and WSL Ubuntu as the Ansible control node.

The Lab 5 VM was already running. Because Ansible was executed from WSL, I used a Windows port proxy so WSL could reach the VM SSH port. The VM itself is still the Lab 5 Vagrant/VirtualBox VM.

Ansible version:

```text
ansible [core 2.17.14]
ansible package installed: ansible-10.7.0
python version = 3.10.12
```

Ansible ping check:

```text
ansible -i ansible/inventory.ini quicknotes_vm -m ping

lab5vm | SUCCESS => {
    "changed": false,
    "ping": "pong"
}
```

---

## Task 1 — Idempotent Deploy to the Lab 5 VM

### File layout

```text
ansible/
├── inventory.ini
├── playbook.yaml
├── files/
│   └── quicknotes
└── templates/
    └── quicknotes.service.j2
```

The static QuickNotes binary was built from the `app/` directory using:

```bash
cd app
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../ansible/files/quicknotes .
cd ..
```

---

### `ansible/inventory.ini`

```ini
[quicknotes_vm]
lab5vm ansible_host=172.31.208.1 ansible_port=2224 ansible_user=vagrant ansible_ssh_private_key_file=~/.ssh/quicknotes_lab5_private_key ansible_python_interpreter=/usr/bin/python3 ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o IdentitiesOnly=yes'
```

Note: `vagrant ssh-config` exposed the VM through localhost. Since Ansible was run from WSL, I used a Windows port proxy so WSL could reach the same Vagrant SSH connection. The port proxy forwards WSL access on port `2224` to the VirtualBox SSH forwarding port.

---

### `ansible/playbook.yaml`

```yaml
---
- name: Deploy QuickNotes with Ansible
  hosts: quicknotes_vm
  become: true
  gather_facts: false

  vars:
    quicknotes_user: quicknotes
    data_dir: /var/lib/quicknotes
    binary_path: /usr/local/bin/quicknotes
    listen_addr: ":8080"
    data_path: /var/lib/quicknotes/notes.json
    seed_path: /var/lib/quicknotes/seed.json
    restart_backoff: 5s

  tasks:
    - name: Create QuickNotes system group
      ansible.builtin.group:
        name: "{{ quicknotes_user }}"
        system: true

    - name: Create QuickNotes system user
      ansible.builtin.user:
        name: "{{ quicknotes_user }}"
        group: "{{ quicknotes_user }}"
        system: true
        create_home: false
        home: "{{ data_dir }}"
        shell: /usr/sbin/nologin

    - name: Ensure QuickNotes data directory exists
      ansible.builtin.file:
        path: "{{ data_dir }}"
        state: directory
        owner: "{{ quicknotes_user }}"
        group: "{{ quicknotes_user }}"
        mode: "0750"

    - name: Copy QuickNotes binary
      ansible.builtin.copy:
        src: files/quicknotes
        dest: "{{ binary_path }}"
        owner: root
        group: root
        mode: "0755"
      notify: restart quicknotes

    - name: Install QuickNotes systemd unit
      ansible.builtin.template:
        src: templates/quicknotes.service.j2
        dest: /etc/systemd/system/quicknotes.service
        owner: root
        group: root
        mode: "0644"
      notify: restart quicknotes

    - name: Enable and start QuickNotes service
      ansible.builtin.systemd:
        name: quicknotes
        enabled: true
        state: started
        daemon_reload: true
      when: not ansible_check_mode

  handlers:
    - name: restart quicknotes
      ansible.builtin.systemd:
        name: quicknotes
        state: restarted
        daemon_reload: true
      when: not ansible_check_mode
```

---

### `ansible/templates/quicknotes.service.j2`

```ini
[Unit]
Description=QuickNotes API
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User={{ quicknotes_user }}
Group={{ quicknotes_user }}
WorkingDirectory={{ data_dir }}
Environment=ADDR={{ listen_addr }}
Environment=DATA_PATH={{ data_path }}
Environment=SEED_PATH={{ seed_path }}
ExecStart={{ binary_path }}
Restart=on-failure
RestartSec={{ restart_backoff }}

[Install]
WantedBy=multi-user.target
```

---

### Syntax check

```text
ansible-playbook -i ansible/inventory.ini ansible/playbook.yaml --syntax-check

playbook: ansible/playbook.yaml
```

---

### Dry run

```text
ansible-playbook -i ansible/inventory.ini ansible/playbook.yaml --check

PLAY [Deploy QuickNotes with Ansible] **************************************************************************************************************************************

TASK [Create QuickNotes system group] **************************************************************************************************************************************
changed: [lab5vm]

TASK [Create QuickNotes system user] ***************************************************************************************************************************************
changed: [lab5vm]

TASK [Ensure QuickNotes data directory exists] *****************************************************************************************************************************
[WARNING]: failed to look up user quicknotes. Create user up to this point in real play
[WARNING]: failed to look up group quicknotes. Create group up to this point in real play
changed: [lab5vm]

TASK [Copy QuickNotes binary] **********************************************************************************************************************************************
changed: [lab5vm]

TASK [Install QuickNotes systemd unit] *************************************************************************************************************************************
changed: [lab5vm]

TASK [Enable and start QuickNotes service] *********************************************************************************************************************************
skipping: [lab5vm]

RUNNING HANDLER [restart quicknotes] ***************************************************************************************************************************************
skipping: [lab5vm]

PLAY RECAP *****************************************************************************************************************************************************************
lab5vm                     : ok=5    changed=5    unreachable=0    failed=0    skipped=2    rescued=0    ignored=0
```

---

### First real run

```text
ansible-playbook -i ansible/inventory.ini ansible/playbook.yaml

PLAY [Deploy QuickNotes with Ansible] **************************************************************************************************************************************

TASK [Create QuickNotes system group] **************************************************************************************************************************************
changed: [lab5vm]

TASK [Create QuickNotes system user] ***************************************************************************************************************************************
changed: [lab5vm]

TASK [Ensure QuickNotes data directory exists] *****************************************************************************************************************************
changed: [lab5vm]

TASK [Copy QuickNotes binary] **********************************************************************************************************************************************
changed: [lab5vm]

TASK [Install QuickNotes systemd unit] *************************************************************************************************************************************
changed: [lab5vm]

TASK [Enable and start QuickNotes service] *********************************************************************************************************************************
changed: [lab5vm]

RUNNING HANDLER [restart quicknotes] ***************************************************************************************************************************************
changed: [lab5vm]

PLAY RECAP *****************************************************************************************************************************************************************
lab5vm                     : ok=7    changed=7    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

---

### Service status

```text
ansible -i ansible/inventory.ini quicknotes_vm -b -m command -a "systemctl status quicknotes --no-pager"

lab5vm | CHANGED | rc=0 >>
● quicknotes.service - QuickNotes API
     Loaded: loaded (/etc/systemd/system/quicknotes.service; enabled; vendor preset: enabled)
     Active: active (running) since Tue 2026-06-23 22:14:50 UTC; 18s ago
   Main PID: 19295 (quicknotes)
      Tasks: 7 (limit: 1099)
     Memory: 1.1M
        CPU: 55ms
     CGroup: /system.slice/quicknotes.service
             └─19295 /usr/local/bin/quicknotes

Jun 23 22:14:50 quicknotes-vm systemd[1]: Started QuickNotes API.
Jun 23 22:14:50 quicknotes-vm quicknotes[19295]: 2026/06/23 22:14:50 quicknotes listening on :8080 (notes loaded: 0)
```

---

### Host verification

```text
curl -s http://localhost:18080/health

{"notes":0,"status":"ok"}
```

---

## Design Questions for Task 1

### a) What is the difference between `command:` and dedicated modules such as `apt`, `file`, `copy`, and `systemd`? Which is idempotent, and why does it matter?

The `command:` module runs a command directly on the target machine. It does not automatically understand the desired final state, so it is not idempotent by default. For example, a command can run every time even if nothing needs to change.

Dedicated modules such as `file`, `copy`, `template`, and `systemd` understand state. The `file` module checks whether a path exists and whether ownership and permissions match. The `copy` module checks whether the destination content differs from the source. The `template` module renders the template and only changes the target file if the rendered output is different. The `systemd` module checks whether the service is enabled or running.

This matters because an idempotent playbook can be safely re-run after a partial failure or during normal maintenance without causing unnecessary changes.

---

### b) `notify:` and handlers: when does a handler fire? When does it not fire? Why is that the right default?

A handler fires only when a task that notifies it reports `changed`. In this lab, the `restart quicknotes` handler is notified by the binary copy task and the systemd unit template task.

The handler does not fire when those tasks report `ok`, meaning the binary and unit file are already correct. This is the right default because services should only restart when something relevant changed. Restarting a service unnecessarily can cause avoidable downtime or disruption.

---

### c) Variable hierarchy: list the top 3 places you would put variables for this lab and why.

For this lab, I would use these three places:

1. **Playbook vars**: This is suitable for a small lab because the variables are easy to see in the same file as the tasks.
2. **Group variables** such as `group_vars/quicknotes_vm.yml`: This would be better if I had different environments, such as development and production, using the same playbook with different values.
3. **Role defaults** such as `roles/quicknotes/defaults/main.yaml`: This would be best if the QuickNotes deployment was converted into a reusable role. Defaults are easy to override from inventory, group variables, host variables, or extra variables.

---

### d) `gather_facts: true` is the default. Do you need it for this playbook? What does turning it off save you per run?

I do not need `gather_facts: true` for this playbook because the tasks do not depend on OS facts, CPU information, memory information, network interface facts, or distribution-specific variables.

Turning it off saves the SSH round trips and time Ansible would spend collecting facts at the beginning of every run. On a small VM this saves a few seconds, but on many hosts it can save much more time.

---

## Task 2 — Prove Idempotency and Selective Re-run

### 2.1 Re-run = zero changes

I re-ran the same playbook without changing anything.

```text
ansible-playbook -i ansible/inventory.ini ansible/playbook.yaml

PLAY [Deploy QuickNotes with Ansible] **************************************************************************************************************************************

TASK [Create QuickNotes system group] **************************************************************************************************************************************
ok: [lab5vm]

TASK [Create QuickNotes system user] ***************************************************************************************************************************************
ok: [lab5vm]

TASK [Ensure QuickNotes data directory exists] *****************************************************************************************************************************
ok: [lab5vm]

TASK [Copy QuickNotes binary] **********************************************************************************************************************************************
ok: [lab5vm]

TASK [Install QuickNotes systemd unit] *************************************************************************************************************************************
ok: [lab5vm]

TASK [Enable and start QuickNotes service] *********************************************************************************************************************************
ok: [lab5vm]

PLAY RECAP *****************************************************************************************************************************************************************
lab5vm                     : ok=6    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

This proves idempotency because the second run found that the system group, system user, data directory, binary, systemd unit, and service state were already correct.

---

### 2.2 Variable tweak = selective change

For the selective-change test, I changed one variable in `ansible/playbook.yaml`:

```yaml
restart_backoff: 3s
```

to:

```yaml
restart_backoff: 4s
```

This variable is used inside the Jinja2 systemd unit template:

```ini
RestartSec={{ restart_backoff }}
```

After changing the variable, I re-ran the playbook.

```text
ansible-playbook -i ansible/inventory.ini ansible/playbook.yaml

PLAY [Deploy QuickNotes with Ansible] **************************************************************************************************************************************

TASK [Create QuickNotes system group] **************************************************************************************************************************************
ok: [lab5vm]

TASK [Create QuickNotes system user] ***************************************************************************************************************************************
ok: [lab5vm]

TASK [Ensure QuickNotes data directory exists] *****************************************************************************************************************************
ok: [lab5vm]

TASK [Copy QuickNotes binary] **********************************************************************************************************************************************
ok: [lab5vm]

TASK [Install QuickNotes systemd unit] *************************************************************************************************************************************
changed: [lab5vm]

TASK [Enable and start QuickNotes service] *********************************************************************************************************************************
ok: [lab5vm]

RUNNING HANDLER [restart quicknotes] ***************************************************************************************************************************************
changed: [lab5vm]

PLAY RECAP *****************************************************************************************************************************************************************
lab5vm                     : ok=7    changed=2    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

Only the systemd unit template changed, and the `restart quicknotes` handler fired because the template task notified it. The other tasks remained `ok`, which shows that the playbook changed only the affected part of the VM.

---

### 2.3 `--check --diff` preview

For the third variable change, I changed:

```yaml
seed_path: /var/lib/quicknotes/seed.json
```

to:

```yaml
seed_path: /var/lib/quicknotes/seed-preview.json
```

Then I ran the playbook in dry-run diff mode.

```text
ansible-playbook -i ansible/inventory.ini ansible/playbook.yaml --check --diff

PLAY [Deploy QuickNotes with Ansible] **************************************************************************************************************************************

TASK [Create QuickNotes system group] **************************************************************************************************************************************
ok: [lab5vm]

TASK [Create QuickNotes system user] ***************************************************************************************************************************************
ok: [lab5vm]

TASK [Ensure QuickNotes data directory exists] *****************************************************************************************************************************
ok: [lab5vm]

TASK [Copy QuickNotes binary] **********************************************************************************************************************************************
ok: [lab5vm]

TASK [Install QuickNotes systemd unit] *************************************************************************************************************************************
--- before: /etc/systemd/system/quicknotes.service
+++ after: /home/student/.ansible/tmp/ansible-local-298045_5t3_u_g/tmpzziaboh6/quicknotes.service.j2
@@ -10,7 +10,7 @@
 WorkingDirectory=/var/lib/quicknotes
 Environment=ADDR=:8080
 Environment=DATA_PATH=/var/lib/quicknotes/notes.json
-Environment=SEED_PATH=/var/lib/quicknotes/seed.json
+Environment=SEED_PATH=/var/lib/quicknotes/seed-preview.json
 ExecStart=/usr/local/bin/quicknotes
 Restart=on-failure
 RestartSec=4s

changed: [lab5vm]

TASK [Enable and start QuickNotes service] *********************************************************************************************************************************
skipping: [lab5vm]

RUNNING HANDLER [restart quicknotes] ***************************************************************************************************************************************
skipping: [lab5vm]

PLAY RECAP *****************************************************************************************************************************************************************
lab5vm                     : ok=5    changed=1    unreachable=0    failed=0    skipped=2    rescued=0    ignored=0
```

The diff clearly shows that only the rendered `SEED_PATH` line would change. Because this was run with `--check`, the change was previewed but not applied.

After recording the diff, I reverted the playbook variable back to:

```yaml
seed_path: /var/lib/quicknotes/seed.json
```

---

## Design Questions for Task 2

### e) Why does the second run report `changed=0`? What specifically does the `file` / `template` module check to decide?

The second run reports `changed=0` because the VM already matches the desired state described in the playbook.

The `file` module checks whether `/var/lib/quicknotes` already exists and whether its owner, group, and permissions match the requested state. The `copy` module checks whether the destination binary already has the same content and requested permissions. The `template` module renders the Jinja2 template, compares the rendered output with the existing remote file, and reports `ok` if the content, owner, group, and mode already match.

Because nothing differed, Ansible did not need to make any changes.

---

### f) What would happen if you used `shell: 'echo "ADDR=..." > /etc/systemd/system/quicknotes.service'` instead of the `template:` module? Trace the failure modes.

Using `shell` to write the systemd unit would be weaker and less reliable. The shell command would run as a side effect and would not naturally compare the desired file content with the existing file. It could report `changed` every time, even when the unit file is already correct.

It would also be easy to accidentally overwrite the file with incomplete content, lose permissions or ownership handling, break quoting of environment variables, or forget to trigger a handler only when the rendered file actually changed. This could cause unnecessary service restarts on every run or deploy a broken systemd unit.

The `template` module avoids these problems because it renders the full file, compares it to the remote file, manages ownership and mode, and only reports `changed` when the rendered output differs.

---

### g) `ansible-playbook --check` is dry-run. `--diff` shows changes. What is the bug you would catch by running `--check --diff` before a production deploy that you would miss with plain `--check`?

Plain `--check` can tell me that a task would change something, but it does not show exactly what content would change. With `--check --diff`, I can inspect the actual file difference before applying it.

For example, in this lab, `--check --diff` showed the exact rendered change:

```diff
-Environment=SEED_PATH=/var/lib/quicknotes/seed.json
+Environment=SEED_PATH=/var/lib/quicknotes/seed-preview.json
```

This would help catch bugs such as a wrong environment variable value, wrong port, wrong path, missing line, or accidentally broken systemd unit before deploying to production.


---

## Bonus Task — `ansible-pull` GitOps Loop

### Bonus setup

For the bonus task, I configured the VM to pull configuration from my GitHub fork using `ansible-pull`.

Inside the VM, I installed Ansible and Git, created a local inventory, created a systemd service for `ansible-pull`, and created a systemd timer that runs every 5 minutes.

The local inventory on the VM targets itself:

```ini
[quicknotes_vm]
localhost ansible_connection=local ansible_python_interpreter=/usr/bin/python3
```

The `ansible-pull` service runs:

```text
/usr/bin/ansible-pull -U https://github.com/giselesikeh/DevOps-Intro.git -C feature/lab7 -d /var/lib/ansible-pull/DevOps-Intro -i /etc/ansible/quicknotes-local.ini ansible/playbook.yaml
```

The timer uses:

```ini
[Timer]
OnBootSec=1min
OnUnitActiveSec=5min
Unit=ansible-pull-quicknotes.service
```

---

### Timer status

```text
ansible -i ansible/inventory.ini quicknotes_vm -b -m shell -a "systemctl list-timers --all | grep ansible-pull || true"

lab5vm | CHANGED | rc=0 >>
Tue 2026-06-23 23:19:57 UTC 2min 32s left      Tue 2026-06-23 23:14:57 UTC 2min 27s ago  ansible-pull-quicknotes.timer  ansible-pull-quicknotes.service
```

The timer was also active and waiting:

```text
ansible -i ansible/inventory.ini quicknotes_vm -b -m command -a "systemctl status ansible-pull-quicknotes.timer --no-pager"

lab5vm | CHANGED | rc=0 >>
● ansible-pull-quicknotes.timer - Run QuickNotes ansible-pull every 5 minutes
     Loaded: loaded (/etc/systemd/system/ansible-pull-quicknotes.timer; enabled; vendor preset: enabled)
     Active: active (waiting) since Tue 2026-06-23 23:14:57 UTC; 2min 44s ago
    Trigger: Tue 2026-06-23 23:19:57 UTC; 2min 15s left
   Triggers: ● ansible-pull-quicknotes.service

Jun 23 23:14:57 quicknotes-vm systemd[1]: Started Run QuickNotes ansible-pull every 5 minutes.
```

---

### Initial manual `ansible-pull` check

I manually started the pull service once to confirm it worked:

```text
ansible -i ansible/inventory.ini quicknotes_vm -b -m command -a "systemctl start ansible-pull-quicknotes.service"
```

The service pulled the repository and applied the playbook locally:

```text
Jun 23 23:15:13 quicknotes-vm ansible-pull[21525]: localhost | CHANGED => {
Jun 23 23:15:13 quicknotes-vm ansible-pull[21525]:     "after": "19ac6f200a19f34d99afa9785d04a94d33043d38",
Jun 23 23:15:13 quicknotes-vm ansible-pull[21525]:     "before": null,
Jun 23 23:15:13 quicknotes-vm ansible-pull[21525]:     "changed": true
Jun 23 23:15:13 quicknotes-vm ansible-pull[21525]: }
```

The playbook then ran locally on the VM:

```text
Jun 23 23:15:13 quicknotes-vm ansible-pull[21525]: PLAY RECAP *********************************************************************
Jun 23 23:15:13 quicknotes-vm ansible-pull[21525]: localhost                  : ok=6    changed=0    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
Jun 23 23:15:13 quicknotes-vm systemd[1]: Finished Ansible Pull QuickNotes GitOps sync.
```

---

### GitOps convergence demonstration

To prove convergence, I changed one variable in `ansible/playbook.yaml`:

```yaml
restart_backoff: 4s
```

to:

```yaml
restart_backoff: 5s
```

Then I committed the change:

```text
git commit -m "Test ansible pull convergence"

[feature/lab7 42d3a5c] Test ansible pull convergence
 1 file changed, 1 insertion(+), 1 deletion(-)
```

Commit details:

```text
COMMIT 42d3a5c65e534c27d5397d2a762271f833083757
TIME 2026-06-24 02:21:57 +0300
SUBJECT Test ansible pull convergence
```

After the timer fired, the VM pulled the new commit:

```text
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: localhost | CHANGED => {
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]:     "after": "42d3a5c65e534c27d5397d2a762271f833083757",
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]:     "before": "19ac6f200a19f34d99afa9785d04a94d33043d38",
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]:     "changed": true,
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]:     "remote_url_changed": false
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: }
```

The playbook then changed only the systemd template and restarted QuickNotes through the handler:

```text
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: TASK [Install QuickNotes systemd unit] *****************************************
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: changed: [localhost]
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: TASK [Enable and start QuickNotes service] *************************************
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: ok: [localhost]
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: RUNNING HANDLER [restart quicknotes] *******************************************
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: changed: [localhost]
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: PLAY RECAP *********************************************************************
Jun 23 23:23:47 quicknotes-vm ansible-pull[22339]: localhost                  : ok=7    changed=2    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
```

I verified that the VM reconciled to the new state:

```text
ansible -i ansible/inventory.ini quicknotes_vm -b -m command -a "grep RestartSec /etc/systemd/system/quicknotes.service"

lab5vm | CHANGED | rc=0 >>
RestartSec=5s
```

This proves that after pushing the Git change, the VM automatically pulled the updated branch and reconciled its local service configuration through the systemd timer.

---

## Bonus Design Questions

### h) `ansible-pull` is pull mode. What is the security benefit compared with push mode?

In the normal push model, the control node needs SSH access into the VM. That means credentials or keys must be managed on the control side, and the VM must expose SSH access to the control node.

With `ansible-pull`, the VM pulls its configuration from Git itself. The control machine does not need to SSH into the VM for every deployment. This reduces the need to expose inbound access and makes the VM responsible for reconciling itself from a trusted Git source.

### i) What is the same pattern called at the Kubernetes layer, and why is `ansible-pull` a fair simulator at the VM layer?

At the Kubernetes layer, this pattern is usually called GitOps. Tools such as Argo CD and Flux follow this model by watching a Git repository and reconciling the cluster state to match the desired state stored in Git.

`ansible-pull` is a fair simulator at the VM layer because it uses the same idea: Git contains the desired state, the managed machine periodically pulls from Git, and the machine reconciles itself automatically without a manual push from the control node.


Not attempted yet.
