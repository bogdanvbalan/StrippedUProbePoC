package main

import (
	"debug/elf"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -Wall" bpf ../bpf_program.c -- -I/usr/include/x86_64-linux-gnu -I../headers -I/usr/include

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: %s <path_to_stripped_binary> <hex_relative_offset_read> <hex_relative_offset_write>", os.Args[0])
	}
	binPath := os.Args[1]
	relativeOffsetReadHex := os.Args[2]
	relativeOffsetWriteHex := os.Args[3]

	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal("Removing memlock:", err)
	}

	// Parse hex relative offsets (from unstripped binary)
	relativeOffsetRead, err := parseHex(relativeOffsetReadHex)
	if err != nil {
		log.Fatalf("Invalid read relative offset %q: %v", relativeOffsetReadHex, err)
	}
	relativeOffsetWrite, err := parseHex(relativeOffsetWriteHex)
	if err != nil {
		log.Fatalf("Invalid write relative offset %q: %v", relativeOffsetWriteHex, err)
	}

	log.Printf("Target binary: %s", binPath)
	log.Printf("Read Relative Offset:   0x%x (%d)", relativeOffsetRead, relativeOffsetRead)
	log.Printf("Write Relative Offset:  0x%x (%d)", relativeOffsetWrite, relativeOffsetWrite)

	// Determine base file offset of the .text segment in the stripped binary
	strippedBaseOffset, err := findStrippedExecutableSegmentOffset(binPath)
	if err != nil {
		log.Fatalf("Could not find base file offset in stripped binary '%s': %v", binPath, err)
	}
	log.Printf("Base file offset (stripped): 0x%x (%d)", strippedBaseOffset, strippedBaseOffset)

	// Compute absolute file offsets by adding base + relative symbol offsets
	fileOffsetRead := strippedBaseOffset + relativeOffsetRead
	fileOffsetWrite := strippedBaseOffset + relativeOffsetWrite
	log.Printf("Computed READ probe file offset:  0x%x", fileOffsetRead)
	log.Printf("Computed WRITE probe file offset: 0x%x", fileOffsetWrite)

	// Load compiled BPF objects
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		log.Fatalf("Loading BPF objects: %v", err)
	}
	defer objs.Close()

	exe, err := link.OpenExecutable(binPath)
	if err != nil {
		log.Fatalf("Opening executable '%s': %v", binPath, err)
	}

	// READ probe
	log.Printf("Attaching uprobe READ at file offset=0x%x", fileOffsetRead)
	upRead, err := exe.Uprobe("", objs.UprobeReadEntry, &link.UprobeOptions{
		Address: fileOffsetRead,
	})
	if err != nil {
		log.Fatalf("Attaching uprobe READ failed: %v", err)
	}
	defer upRead.Close()
	log.Println("Attached uprobe READ")

	// WRITE entry probe
	log.Printf("Attaching uprobe WRITE at file offset=0x%x", fileOffsetWrite)
	upWrite, err := exe.Uprobe("", objs.UprobeWriteEntry, &link.UprobeOptions{
		Address: fileOffsetWrite,
	})
	if err != nil {
		log.Fatalf("Attaching uprobe WRITE failed: %v", err)
	}
	defer upWrite.Close()
	log.Println("Attached uprobe WRITE")

	// WRITE exit probe (uretprobe)
	log.Printf("Attaching uretprobe WRITE at file offset=0x%x", fileOffsetWrite)
	upRet, err := exe.Uretprobe("", objs.UretprobeWriteExit, &link.UprobeOptions{
		Address: fileOffsetWrite,
	})
	if err != nil {
		log.Fatalf("Attaching uretprobe WRITE failed: %v", err)
	}
	defer upRet.Close()
	log.Println("Attached uretprobe WRITE")

	log.Println("Probes attached — waiting for trace events (trace_pipe)...")

	// Wait for termination signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Detaching probes and exiting.")
}

func parseHex(hexStr string) (uint64, error) {
	clean := strings.TrimPrefix(strings.ToLower(hexStr), "0x")
	if clean == "" {
		return 0, errors.New("empty hex string")
	}
	v, err := strconv.ParseUint(clean, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing hex '%s': %w", hexStr, err)
	}
	return v, nil
}

func findStrippedExecutableSegmentOffset(path string) (uint64, error) {
	f, err := elf.Open(path)
	if err != nil {
		return 0, fmt.Errorf("elf.Open %s: %w", path, err)
	}
	defer f.Close()

	// Inspect PT_LOAD segments for debugging
	log.Println("Inspecting program headers:")
	for _, prog := range f.Progs {
		log.Printf("  Type=%v Flags=%v Off=0x%x Vaddr=0x%x", prog.Type, prog.Flags, prog.Off, prog.Vaddr)
	}

	// Prefer .text section offset when available
	if sec := f.Section(".text"); sec != nil && sec.Offset != 0 {
		log.Printf("Using .text section offset: Off=0x%x, Addr=0x%x", sec.Offset, sec.Addr)
		return sec.Offset, nil
	}

	// Otherwise, pick the first executable PT_LOAD
	for _, prog := range f.Progs {
		if prog.Type == elf.PT_LOAD && (prog.Flags&elf.PF_X) != 0 {
			log.Printf("Selected PT_LOAD exec segment: Off=0x%x, Vaddr=0x%x", prog.Off, prog.Vaddr)
			return prog.Off, nil
		}
	}

	return 0, errors.New("no executable segment or .text section found")
}
