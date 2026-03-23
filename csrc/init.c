#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mount.h>

int main(int argc, char **argv) {
	const char *usage = "Usage: %s [--hostname <name>] [--rootfs <path>] <command> [<args>...]\n";

	if (argc < 2) {
		fprintf(stderr, usage, argv[0]);
		exit(EXIT_FAILURE);
	}

	char *hostname = "container";
	char *rootfs = "rootfs";

	int i = 1;
	for (; i < argc; i++) {
		if (strcmp(argv[i], "--") == 0) {
			i++;
			break;
		}

		if (strcmp(argv[i], "--hostname") == 0 && i + 1 >= argc) {
			fprintf(stderr, "Missing value for --hostname\n");
			fprintf(stderr, usage, argv[0]);
			exit(EXIT_FAILURE);
		}
		if (strcmp(argv[i], "--hostname") == 0 && i + 1 < argc) {
			hostname = argv[++i];
			continue;
		}
		if (strcmp(argv[i], "--rootfs") == 0 && i + 1 >= argc) {
			fprintf(stderr, "Missing value for --rootfs\n");
			fprintf(stderr, usage, argv[0]);
			exit(EXIT_FAILURE);
		}
		if (strcmp(argv[i], "--rootfs") == 0 && i + 1 < argc) {
			rootfs = argv[++i];
			continue;
		}
		break;
	}

	if (i >= argc) {
		fprintf(stderr, usage, argv[0]);
		exit(EXIT_FAILURE);
	}

	char *command = argv[i];
	char **args = argv + i;
	char *envp[] = { NULL };
	if (sethostname(hostname, strlen(hostname)) != 0) {
		perror("Failed to set hostname");
		exit(EXIT_FAILURE);
	}

	if (chroot(rootfs) != 0) {
		perror("chroot");
		exit(EXIT_FAILURE);
	}
	if (chdir("/") != 0) {
		perror("chdir");
		exit(EXIT_FAILURE);
	}

    if (mount("proc", "/proc", "proc", 0, "") != 0) {
        perror("mount");
        exit(EXIT_FAILURE);
    }

	execve(command, args, envp);

	perror("execve");
	exit(EXIT_FAILURE);
}
