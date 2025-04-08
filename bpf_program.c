// bpf_program.c
#include <linux/bpf.h>
#include <linux/ptrace.h>      
#include <bpf/bpf_helpers.h>   

#ifndef SEC
#define SEC(NAME) __attribute__((section(NAME), used))
#endif

char LICENSE[] SEC("license") = "GPL";

// Uprobe for read entry
SEC("uprobe/read_entry") 
int uprobe_read_entry(struct pt_regs *ctx) {
    bpf_printk("eBPF: dummy_SSL_read ENTERED!\n");
    return 0; 
}

// Uprobe for write entry
SEC("uprobe/write_entry") 
int uprobe_write_entry(struct pt_regs *ctx) {
    bpf_printk("eBPF: dummy_SSL_write ENTERED!\n");
    return 0; 
}

// Uretprobe for write exit
SEC("uretprobe/write_exit") 
int uretprobe_write_exit(struct pt_regs *ctx) {
    bpf_printk("eBPF: dummy_SSL_write EXITED!\n");
    return 0; 
}
