# eBPF Offset Probing POC

## Goal

This project is a Proof of Concept (POC) demonstrating how to attach eBPF uprobes/uretprobes to functions within a **stripped** native binary (specifically, a non-PIE executable). Since stripped binaries lack symbol tables, probes cannot be attached using function names directly. Instead, this POC uses function **file offsets** obtained from an **unstripped** version of the same binary.

This technique is relevant for scenarios where you need to instrument production binaries (like Envoy compiled against BoringSSL) which are often stripped for size and security reasons.

## Concept

1.  **Target Simulation:** A simple C program (`dummy_ssl.c`) simulates target functions (`dummy_SSL_read`, `dummy_SSL_write`) we might want to trace in a real application.
2.  **Compilation:** The C program is compiled into two versions using `compile.sh`:
    *   `dummy_ssl_unstripped`: Compiled with debug symbols and as non-PIE (`-no-pie`).
    *   `dummy_ssl_stripped`: A copy of the unstripped version, subsequently processed with `strip` to remove symbols.
3.  **Relative Offset Discovery:** The `find_offsets.sh` script analyzes the `dummy_ssl_unstripped` binary using `nm` and `readelf`. It calculates the offset of the target function *relative* to the start of its code section (`.text`), adding a 4-byte adjustment for the `endbr64` instruction. This `relative_offset` is exported.
4.  **eBPF Program:** A basic eBPF C program (`bpf_program.c`) is created to simply print messages to the kernel trace pipe when the probes are triggered (`bpf_printk`).
5.  **Go Loader & Final Offset Calculation:** A Go program (`goloader/main.go`) uses the `cilium/ebpf` library. It:
    *   Takes the target binary path (`./dummy_ssl_stripped`) and the *relative offsets* (from `find_offsets.sh`) as command-line arguments.
    *   Finds the *base file offset* (`stripped_base_offset`) of the executable code segment in the *stripped* binary using the `debug/elf` package.
    *   Calculates the *final absolute file offset* for the probe: `final_probe_offset = stripped_base_offset + relative_offset`.
    *   Attaches the eBPF probes to the **stripped** binary using this `final_probe_offset` in the `Address` field of `link.UprobeOptions` (with `Offset: 0`).
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
    *(Wait for "Probes attached — waiting for trace events..." message)*

2.  **Terminal 2 (Trace Monitor):** Monitor the kernel trace pipe:
    ```bash
    sudo cat /sys/kernel/debug/tracing/trace_pipe
    ```

3.  **Terminal 3 (Target Program):** Run the **stripped** C program:
    ```bash
    ./dummy_ssl_stripped
    ```
    Press Enter when prompted.

## Expected Output

When the `dummy_ssl_stripped` program calls `dummy_SSL_read` and `dummy_SSL_write`, you should see messages like the following appearing in **Terminal 2** (the trace pipe):

<...>- (...) [00N] .... : bpf_printk: eBPF: dummy_SSL_read ENTERED! <...>- (...) [00N] .... : bpf_printk: eBPF: dummy_SSL_write ENTERED! <...>- (...) [00N] .... : bpf_printk: eBPF: dummy_SSL_write EXITED!

*(Repeat for each iteration in the C program)*

Press `Ctrl+C` in Terminal 1 to stop the eBPF loader and detach the probes.

## File Structure

*   `dummy_ssl.c`: Target C application.
*   `bpf_program.c`: eBPF probe code.
*   `goloader/`: Directory containing the Go eBPF loader source (`main.go`, `go.mod`, `go.sum`).
*   `compile.sh`: Builds all components.
*   `find_offsets.sh`: Calculates *relative* offsets from the unstripped binary.
*   `run_poc.sh`: Orchestrates the execution and provides instructions.
*   `README.md`: This file.
*   `LICENSE`: License information (assuming MIT + component licenses).
*   `.gitignore`: Specifies files to be ignored by Git.
*   *(Build Outputs)*: `dummy_ssl_unstripped`, `dummy_ssl_stripped`, `bpf_program.o`, `ebpf_loader`.

## License

This project uses the MIT License for the user-provided code. Please see the `LICENSE` file (if created) for details and information on component licenses (GPL for eBPF, Apache 2.0 for cilium/ebpf).

## Notes

*   The success of this specific offset calculation method (`stripped_base_file_offset + relative_offset` passed to `Address`) has been confirmed in the environment where this POC was developed. However, eBPF interactions can sometimes vary subtly between kernel versions or library updates.
*   This POC assumes that the relative order and offsets of instructions *within* the `.text` section are preserved during the `strip` process, which is generally true for standard `strip` usage.
*   The `endbr64` adjustment (+4 bytes) is specific to binaries compiled with CET/IBT enabled. This might need adjustment if different compiler flags are used or if the target function doesn't have this instruction.
*   The binary is compiled as non-PIE (`-no-pie`). Probing Position Independent Executables (PIE) might require different offset calculations or handling, as addresses are relative to a base that changes with ASLR even before considering stripping.