package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/cilium/ebpf/rlimit" // Needed again

	"github.com/cilium/ebpf/link"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -Wall" bpf ../bpf_program.c -- -I/usr/include/x86_64-linux-gnu -I../headers -I/usr/include

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: %s <path_to_stripped_binary> <hex_offset_read> <hex_offset_write>", os.Args[0])
	}
	binPath := os.Args[1]
	offsetReadHex := os.Args[2]
	offsetWriteHex := os.Args[3]

	// Add rlimit removal back
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal("Removing memlock:", err)
	}

	// Parse hex offsets
	offsetRead, err := parseHex(offsetReadHex)
	if err != nil {
		log.Fatalf("Invalid read offset %q: %v", offsetReadHex, err)
	}
	offsetWrite, err := parseHex(offsetWriteHex)
	if err != nil {
		log.Fatalf("Invalid write offset %q: %v", offsetWriteHex, err)
	}

	log.Printf("Target binary: %s", binPath)
	log.Printf("Read Offset:   0x%x (%d)", offsetRead, offsetRead)
	log.Printf("Write Offset:  0x%x (%d)", offsetWrite, offsetWrite)

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err = loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("Loading BPF objects: %v", err)
	}
	defer objs.Close()

	ex, err := link.OpenExecutable(binPath)
	if err != nil {
		log.Fatalf("Opening executable '%s': %v", binPath, err)
	}

	log.Printf("Attaching uprobe READ using offset + dummy symbol...")
	upRead, err := ex.Uprobe("_start", objs.UprobeReadEntry, &link.UprobeOptions{
		Offset: offsetRead,
	})
	if err != nil {
		log.Fatalf("Attaching uprobe to read offset 0x%x (symbol _start): %v", offsetRead, err)
	}
	defer upRead.Close()
	log.Printf("Attached uprobe to dummy_SSL_read at offset 0x%x", offsetRead)

	log.Printf("Attaching uprobe WRITE using offset + dummy symbol...")
	upWrite, err := ex.Uprobe("_start", objs.UprobeWriteEntry, &link.UprobeOptions{
		Offset: offsetWrite,
	})
	if err != nil {
		log.Fatalf("Attaching uprobe to write offset 0x%x (symbol _start): %v", offsetWrite, err)
	}
	defer upWrite.Close()
	log.Printf("Attached uprobe to dummy_SSL_write entry at offset 0x%x", offsetWrite)

	log.Printf("Attaching uretprobe WRITE using offset + dummy symbol...")
	urpWrite, err := ex.Uretprobe("_start", objs.UretprobeWriteExit, &link.UprobeOptions{
		Offset: offsetWrite,
	})
	if err != nil {
		log.Fatalf("Attaching uretprobe to write offset 0x%x (symbol _start): %v", offsetWrite, err)
	}
	defer urpWrite.Close()
	log.Printf("Attached uretprobe to dummy_SSL_write exit at offset 0x%x", offsetWrite)

	log.Println("Successfully attached eBPF probes (using offset + dummy symbol). Waiting for events...")
	log.Println("Run the target program now, and watch for output in:")
	log.Println("sudo cat /sys/kernel/debug/tracing/trace_pipe")
	log.Println("Press Ctrl+C to stop.")

	// Wait for termination signal
	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)
	<-stopper

	log.Println("Received signal, detaching probes and exiting.")
}

func parseHex(hexStr string) (uint64, error) {
	cleaned := strings.TrimPrefix(strings.ToLower(hexStr), "0x")
	if cleaned == "" {
		return 0, errors.New("empty hex string")
	}
	val, err := strconv.ParseUint(cleaned, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing hex '%s': %w", hexStr, err)
	}
	return val, nil
}
