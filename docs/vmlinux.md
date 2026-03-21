# vmlinux, kernel BTF, and custom kernels

The Phantom agent uses kernel **BTF** (BPF Type Format) for **CO-RE** in eBPF that references kernel types (e.g. `hook add`). When available, `list <symbol>` also uses an **unstripped vmlinux ELF** for disassembly via `objdump`.

## Feature matrix

| Feature | Depends on | Notes |
|---------|------------|--------|
| **kprobe / break** (repo `minikprobe.o`) | Running kernel, kallsyms | BTF not strictly required. |
| **`hook add` CO-RE** (`BPF_CORE_READ`, etc.) | **Kernel BTF** or a **vmlinux ELF** with `.BTF` | Without BTF, compile/attach may fail. |
| **`list <kernel-symbol>` disassembly** | A **vmlinux** file whose addresses match **kallsyms**, plus `objdump` or `llvm-objdump` on the host | `list` still works with kallsyms only; there will be no Disassembly section. |

## Agent BTF load order

Implementation: [`lib/agent/server/btf_spec_linux.go`](../lib/agent/server/btf_spec_linux.go).

1. **`btf.LoadKernelSpec()`** — usually **`/sys/kernel/btf/vmlinux`** (requires `CONFIG_DEBUG_INFO_BTF=y` in the running kernel).
2. **`-vmlinux` / `PHANTOM_VMLINUX`** — path you supply (must be an ELF BTF blob `LoadSpec` accepts).
3. **Automatic candidates** for the running release (`uname -r`):
   - `/boot/vmlinux-$(uname -r)`
   - `/usr/lib/debug/boot/vmlinux-$(uname -r)` (common on Debian/Ubuntu **linux-image-*-dbgsym**)
   - `/lib/modules/$(uname -r)/build/vmlinux` (typical for self-built kernels: build tree `vmlinux`)

If BTF is loaded successfully from an **ELF file** and you did **not** pass `-vmlinux`, the agent sets that path as the vmlinux used for **`list` disassembly** (same ELF as for CO-RE).

If only **(1)** succeeds (sysfs BTF) and there is no on-disk vmlinux path, **`list` still runs without disassembly**; pass **`-vmlinux`** explicitly to a vmlinux with symbols when you need asm output.

## Distribution kernels (binary packages)

- Prefer **`/sys/kernel/btf/vmlinux`** to exist (common on recent distros with BTF enabled).
- For **`list` disassembly**, install a **debug** kernel image or copy the matching **vmlinux**, e.g. on Ubuntu after installing the debug package matching your running kernel:

  ```bash
  # Typical path after installing debug symbols for the running kernel:
  # /usr/lib/debug/boot/vmlinux-$(uname -r)
  ```

## Self-built kernels

1. **Enable BTF** (recommended, matches the running image):

   ```text
   CONFIG_DEBUG_INFO_BTF=y
   ```

   After install/boot you should see **`/sys/kernel/btf/vmlinux`**. If you did not install the full image on the target host, copy **`vmlinux`** from the build tree (`make` output) and start the agent with:

   ```bash
   ./phantom-agent -listen :9090 -kprobe /path/to/minikprobe.o \
     -bpf-include /path/to/src/agent/bpf/include \
     -vmlinux /home/you/linux/vmlinux
   ```

2. **No sysfs BTF at runtime** (stripped config, or only `bzImage` copied): you must supply a **vmlinux** with a **`.BTF`** section; the agent tries **explicit `-vmlinux`** and the **standard paths** above.

3. **Build tree link**: if **`/lib/modules/$(uname -r)/build`** points at the kernel build tree and contains **`vmlinux`**, automatic discovery usually finds it.

## Environment variables and systemd

- **`PHANTOM_VMLINUX`** — same as **`-vmlinux`**.
- Example in [deploy/systemd/phantom-agent.service](../deploy/systemd/phantom-agent.service):

  ```ini
  Environment=PHANTOM_VMLINUX=/usr/lib/debug/boot/vmlinux-6.6.0-amd64
  ```

## e2e / CI

CI on GitHub **ubuntu-latest** normally has BTF and does **not** need vmlinux. On a **custom kernel** host, set **`E2E_VMLINUX`** so tests pass **`-vmlinux`** when starting the agent (see [`test/e2e/helpers.go`](../test/e2e/helpers.go)).

## Troubleshooting

- Log line **`kernel BTF unavailable`**: enable `CONFIG_DEBUG_INFO_BTF`, or set **`-vmlinux`**, or install a debug vmlinux, or verify your self-built artifact path.
- **`hook add` compile errors** mentioning BTF/types: usually **missing or mismatched BTF** vs the running kernel; align **vmlinux** or sysfs BTF with the **same** kernel build.
- **`list` without Disassembly**: set **`-vmlinux`** and install **binutils** (`objdump`); addresses must match the running kernel’s **kallsyms** (same build).
