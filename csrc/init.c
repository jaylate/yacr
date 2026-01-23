#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>

int main(int argc, char **argv) {
	if (argc < 2) {
		fprintf(stderr, "Usage: %s <command> [<args>...]\n", argv[0]);
		exit(EXIT_FAILURE);
	}
	char *command = argv[1];
	char **args = argv+1;
	char *envp[] = { NULL };
	char *hostname = "container";
	if (sethostname(hostname, strlen(hostname)) != 0) {
		perror("Failed to set hostname");
		exit(EXIT_FAILURE);
	}

	chroot("rootfs");
	chdir("/");

    if (mount("none", "/proc", "proc", 0, "") != 0) {
        perror("mount");
        exit(EXIT_FAILURE);
    }

	execve(command, args, envp);

	perror("execve");
	exit(EXIT_FAILURE);
}
