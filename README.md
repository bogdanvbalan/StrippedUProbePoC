# eBPF Offset Probing POC

## Goal

This project is a Proof of Concept (POC) demonstrating how to attach eBPF uprobes/uretprobes to functions within a **stripped** native binary (specifically, a non-PIE executable). Since stripped binaries lack symbol tables, probes cannot be attached using function names directly. Instead, this POC uses function **file offsets** obtained from an **unstripped** version of the same binary.

This technique is relevant for scenarios where you need to instrument production binaries (like Envoy compiled against BoringSSL) which are often stripped for size and security reasons.

## Concept

1.  **Target Simulation:** A simple C program (`dummy_ssl.c`) simulates target functions (`dummy_SSL_read`, `dummy_SSL_write`) we might want to trace in a real application.
2.  **Compilation:** The C program is compiled into two versions using `compile.sh`:
    * `dummy_ssl_unstripped`: Compiled with debug symbols and as non-PIE (`-no-pie`).
    * `dummy_ssl_stripped`: A copy of the unstripped version, subsequently processed with `strip` to remove symbols.
3.  **Offset Discovery:** The `find_offsets.sh` script analyzes the `dummy_ssl_unstripped` binary using `nm` to find the file offsets (addresses relative to the start of the file) for the `dummy_SSL_read` and `dummy_SSL_write` functions.
4.  **eBPF Program:** A basic eBPF C program (`bpf_program.c`) is created to simply print messages to the kernel trace pipe when the probes are triggered (`bpf_printk`).
5.  **Go Loader:** A Go program (`goloader/main.go`) uses the `cilium/ebpf` library to:
    * Load the compiled eBPF program.
    * Take the target binary path (`./dummy_ssl_stripped`) and the calculated function offsets as command-line arguments.
    * Attempt to attach the eBPF probes to the **stripped** binary using the provided offsets.
6.  **Verification:** The `run_poc.sh` script orchestrates the build and offset discovery, then provides instructions for manually running the Go loader, the trace pipe monitor, and the stripped C program to observe if the hooks trigger.

## Components

* `dummy_ssl.c`: Simple C program simulating target functions.
* `bpf_program.c`: Simple eBPF C program using `bpf_printk`.
* `goloader/main.go`: Go program using `cilium/ebpf` to load and attach eBPF probes by offset.
* `goloader/go.mod`, `goloader/go.sum`: Go module files.
* `compile.sh`: Builds C binaries, eBPF object, and Go loader. Compiles eBPF without `-g` (debug info) as it led to clearer errors.
* `find_offsets.sh`: Finds function offsets from the unstripped binary using `nm`.
* `run_poc.sh`: Main script to orchestrate build, offset discovery, and provide run instructions.
* `README.md`: This file.
* `.gitignore`: Specifies intentionally untracked files.

## Prerequisites

* Linux Kernel with eBPF support (most modern kernels).
* `go` compiler (tested with Go 1.18+, requires module support).
* `clang` and `llvm` (version 10+ recommended for BPF compilation).
* `gcc` and `binutils` (`strip`, `nm`).
* `make` (often needed by dependencies).
* Potentially `libbpf-dev` (or equivalent) for BPF headers, although `cilium/ebpf` often bundles needed headers via `bpf2go`. Ensure include paths in `compile.sh` and `goloader/main.go` (`//go:generate`) match your system (e.g., `/usr/include/x86_64-linux-gnu`).
* `sudo` privileges for loading eBPF programs and accessing the trace pipe.
* (Optional but recommended for validation) `bpftrace`.

## Build

1.  Ensure all prerequisites are installed.
2.  Ensure correct include paths (for your architecture, e.g., `x86_64-linux-gnu`) are set in `compile.sh` (clang command) and `goloader/main.go` (`//go:generate` line).
3.  Make scripts executable: `chmod +x *.sh`
4.  Run the main build and setup script:
    ```bash
    ./run_poc.sh
    ```
    This script will:
    * Compile `dummy_ssl.c` (stripped and unstripped, non-PIE).
    * Compile `bpf_program.c` (into `bpf_program.o`, without debug info).
    * Generate Go code from the BPF object file using `bpf2go` (invoked via `go generate` within `compile.sh`).
    * Compile the Go loader (`goloader/main.go`) into `./ebpf_loader`.
    * Run `find_offsets.sh` to find and export offsets.
    * Print the final instructions for running the POC.

## Run (Manual Steps)

The `run_poc.sh` script will print these instructions after a successful build:

1.  **Terminal 1 (eBPF Loader):** Run the compiled Go program with `sudo`, passing the path to the **stripped** binary and the discovered offsets:
    ```bash
    # Example command printed by run_poc.sh:
    sudo ./ebpf_loader ./dummy_ssl_stripped 0x<offset_read> 0x<offset_write>
    ```
    *(Observe the output. Based on current findings, this command is expected to **fail** with an error like "symbol _start: not found".)*

2.  **Terminal 2 (Trace Monitor):** Monitor the kernel trace pipe:
    ```bash
    sudo cat /sys/kernel/debug/tracing/trace_pipe
    ```

3.  **Terminal 3 (Target Program):** Run the **stripped** C program:
    ```bash
    ./dummy_ssl_stripped
    ```
    Press Enter when prompted.
