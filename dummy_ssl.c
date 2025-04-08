#include <stdio.h>
#include <unistd.h>
#include <time.h>

__attribute__((noinline))
int dummy_SSL_read(int sock, char *buf, int len) {
    printf("dummy_SSL_read called (PID: %d)\n", getpid());
    usleep(10000); 
    if (buf && len > 0) {
        buf[0] = 'R'; 
    }
    return 1; 
}

// Simulate SSL_write
__attribute__((noinline))
int dummy_SSL_write(int sock, const char *buf, int len) {
    printf("dummy_SSL_write called (PID: %d)\n", getpid());
    usleep(15000); 
    return len; 
}

int main() {
    char buffer[10];
    int i = 0;
    printf("Starting dummy SSL client (PID: %d). Will call functions periodically.\n", getpid());
    printf("Run the eBPF loader in another terminal now.\n");
    printf("Press Enter to start calling functions...\n");
    getchar(); 

    while (i < 5) { 
        printf("\n--- Iteration %d ---\n", i + 1);
        dummy_SSL_read(1, buffer, sizeof(buffer));
        dummy_SSL_write(1, "hello", 5);
        sleep(2); 
        i++;
    }

    printf("\nDummy SSL client finished.\n");
    return 0;
}
