#!/bin/bash

set -e

echo "### Step 1: Compiling C, eBPF, and Go programs ###"
./compile.sh

echo -e "\n### Step 2: Finding function offsets in unstripped binary ###"
source ./find_offsets.sh

# Check if offsets were found and exported
if [ -z "$DUMMY_READ_OFFSET" ] || [ -z "$DUMMY_WRITE_OFFSET" ]; then
    echo "Error: Offsets not found or not set after sourcing find_offsets.sh. Exiting."
    exit 1
fi

echo -e "\n### Step 3: Running the POC ###"
echo "Binary Path:   ./dummy_ssl_stripped"
echo "Read Offset:   ${DUMMY_READ_OFFSET}"
echo "Write Offset:  ${DUMMY_WRITE_OFFSET}"
echo ""
echo "In the FIRST terminal, run the eBPF loader (requires sudo):"
echo "sudo ./ebpf_loader ./dummy_ssl_stripped ${DUMMY_READ_OFFSET} ${DUMMY_WRITE_OFFSET}"
echo ""
echo "In the SECOND terminal, monitor the kernel trace pipe (requires sudo):"
echo "sudo cat /sys/kernel/debug/tracing/trace_pipe"
echo ""
echo "In the THIRD terminal, run the stripped C program AFTER starting the loader:"
echo "Press Enter in the C program's window when prompted to start function calls."
echo "./dummy_ssl_stripped"
echo ""
echo "You should see 'eBPF: ...' messages in the second terminal when the C program calls the functions."
echo "Press Ctrl+C in the loader terminal (first one) to stop it."
