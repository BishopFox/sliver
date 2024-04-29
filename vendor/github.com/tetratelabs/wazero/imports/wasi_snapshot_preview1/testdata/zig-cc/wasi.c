#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <stdbool.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <stdlib.h>
#include <time.h>

#define formatBool(b) ((b) ? "true" : "false")

void main_ls(char *dir_name, bool repeat) {
  DIR *d;
  struct dirent *dir;
  d = opendir(dir_name);
  if (d) {
    while ((dir = readdir(d)) != NULL) {
      printf("./%s\n", dir->d_name);
    }
    if (repeat) {
      rewinddir(d);
      while ((dir = readdir(d)) != NULL) {
        printf("./%s\n", dir->d_name);
      }
    }
    closedir(d);
  } else if (errno == ENOTDIR) {
    printf("ENOTDIR\n");
  } else {
    printf("%s\n", strerror(errno));
  }
}

void main_stat() {
  printf("stdin isatty: %s\n", formatBool(isatty(0)));
  printf("stdout isatty: %s\n", formatBool(isatty(1)));
  printf("stderr isatty: %s\n", formatBool(isatty(2)));
  printf("/ isatty: %s\n", formatBool(isatty(3)));
}

void main_poll(int timeout, int millis) {
  int ret = 0;
  fd_set rfds;
  struct timeval tv;

  FD_ZERO(&rfds);
  FD_SET(0, &rfds);

  tv.tv_sec = timeout;
  tv.tv_usec = millis*1000;
  ret = select(1, &rfds, NULL, NULL, &tv);
  if ((ret > 0) && FD_ISSET(0, &rfds)) {
    printf("STDIN\n");
  } else {
    printf("NOINPUT\n");
  }
}

void main_sleepmillis(int millis) {
   struct timespec tim, tim2;
   tim.tv_sec = 0;
   tim.tv_nsec = millis * 1000000;

   if(nanosleep(&tim , &tim2) < 0 ) {
      printf("ERR\n");
      return;
   }

   printf("OK\n");
}

void main_open_rdonly() {
  const char *path = "zig-cc.rdonly.test";
  int fd;
  char buf[32];

  fd = open(path, O_CREAT|O_TRUNC|O_RDONLY, 0644);
  if (fd < 0) {
    perror("ERR: open");
    goto cleanup;
  }
  if (write(fd, "hello world\n", 12) >= 0) {
    perror("ERR: write");
    goto cleanup;
  }
  if (read(fd, buf, sizeof(buf)) != 0) {
    perror("ERR: read");
    goto cleanup;
  }
  puts("OK");
 cleanup:
  close(fd);
  unlink(path);
}

void main_open_wronly() {
  const char *path = "zig-cc.wronly.test";
  int fd;
  char buf[32];

  fd = open(path, O_CREAT|O_TRUNC|O_WRONLY, 0644);
  if (fd < 0) {
    perror("ERR: open");
    goto cleanup;
  }
  if (write(fd, "hello world\n", 12) != 12) {
    perror("ERR: write");
    goto cleanup;
  }
  if (read(fd, buf, sizeof(buf)) >= 0) {
    perror("ERR: read");
    goto cleanup;
  }
  puts("OK");
 cleanup:
  close(fd);
  unlink(path);
}

void main_sock() {
  // Get a listener from the pre-opened file descriptor.
  // The listener is the first pre-open, with a file-descriptor of 3.
  int listener_fd = 3;

  int nfd = -1;
  // Some runtimes set the fd to NONBLOCK
  // so we loop until we no longer get EAGAIN.
  while (true) {
    nfd = accept(listener_fd, NULL, NULL);
    if (nfd >= 0) {
      break;
    }
    if (errno == EAGAIN) {
      sleep(1);
      continue;
    } else {
      perror("ERR: accept");
      return;
    }
  }

  // Wait data to be available on nfd for 1 sec.
  char buf[32];
  struct timeval tv = {1, 0};
  fd_set set;
  FD_ZERO(&set);
  FD_SET(nfd, &set);
  int ret = select(nfd+1, &set, NULL, NULL, &tv);

  // If some data is available, read it.
  if (ret) {
    // Assume no error: we are about to quit
    // and we will check `buf` anyway.
    recv(nfd, buf, sizeof(buf), 0);
    printf("%s\n", buf);
  } else {
    puts("ERR: failed to read data");
  }
}

void main_nonblock(char* fpath) {
  struct timespec tim, tim2;
  tim.tv_sec = 0;
  tim.tv_nsec = 100 * 1000000; // 100 msec
  int fd = open(fpath, O_RDONLY | O_NONBLOCK);
  char buf[32];
  ssize_t newLen = 0;
  while (newLen == 0) {
    newLen = read(fd, buf, sizeof(buf));
    // If an empty string is read, newLen might be 1,
    // causing the loop to terminate.
    if (strlen(buf) == 0) {
      newLen = 0;
    }
    if (errno == EAGAIN || newLen == 0) {
      printf(".");
      nanosleep(&tim , &tim2) ;
      continue;
    }
  }
  printf("\n%s\n", buf);
  close(fd);
}

int main(int argc, char** argv) {
  if (strcmp(argv[1],"ls")==0) {
    bool repeat = false;
    if (argc > 3) {
      repeat = strcmp(argv[3],"repeat")==0;
    }
    main_ls(argv[2], repeat);
  } else if (strcmp(argv[1],"stat")==0) {
    main_stat();
  } else if (strcmp(argv[1],"poll")==0) {
    int timeout = 0;
    int usec = 0;
    if (argc > 2) {
        timeout = atoi(argv[2]);
    }
    if (argc > 3) {
        usec = atoi(argv[3]);
    }
    main_poll(timeout, usec);
  } else if (strcmp(argv[1],"sleepmillis")==0) {
    int timeout = 0;
    if (argc > 2) {
        timeout = atoi(argv[2]);
    }
    main_sleepmillis(timeout);
  } else if (strcmp(argv[1],"open-rdonly")==0) {
    main_open_rdonly();
  } else if (strcmp(argv[1],"open-wronly")==0) {
    main_open_wronly();
  } else if (strcmp(argv[1],"sock")==0) {
    main_sock();
  } else if (strcmp(argv[1],"nonblock")==0) {
    main_nonblock(argv[2]);
  } else {
    fprintf(stderr, "unknown command: %s\n", argv[1]);
    return 1;
  }
  return 0;
}
