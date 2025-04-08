#!/bin/bash

set -e # Exit on error

C_SOURCE="dummy_ssl.c"
UNSTRIPPED_BINARY="dummy_ssl_unstripped"
STRIPPED_BINARY="dummy_ssl_stripped"
BPF_C_SOURCE="bpf_program.c"
BPF_OBJECT="bpf_program.o"
GO_SUBDIR="goloader" # Subdir for Go code
GO_BINARY="ebpf_loader" # Output binary in parent dir

echo "Compiling C program..."
gcc -O0 -g -no-pie ${C_SOURCE} -o ${UNSTRIPPED_BINARY}
cp ${UNSTRIPPED_BINARY} ${STRIPPED_BINARY}
strip ${STRIPPED_BINARY}
echo "Unstripped binary: ${UNSTRIPPED_BINARY}"
echo "Stripped binary:   ${STRIPPED_BINARY}"

echo "Compiling eBPF program..."
clang -O2 -target bpf -c ${BPF_C_SOURCE} -o ${BPF_OBJECT} \
  -I/usr/include/x86_64-linux-gnu
echo "eBPF object file: ${BPF_OBJECT}"

cd goloader && go generate ./...
cd -

echo "Compiling Go program (cd ./${GO_SUBDIR} && go build -o ../${GO_BINARY} .)..."
(cd ${GO_SUBDIR} && go build -v -o ../${GO_BINARY} .)

echo "Go loader binary: ./${GO_BINARY}"

echo "Compilation finished."
chmod +x ${UNSTRIPPED_BINARY} ${STRIPPED_BINARY} ${GO_BINARY}