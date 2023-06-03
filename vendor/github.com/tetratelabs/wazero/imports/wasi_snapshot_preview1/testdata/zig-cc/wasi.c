#include <dirent.h>
#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <stdbool.h>
#include <sys/select.h>

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

void main_poll() {
  int ret = 0;
  fd_set rfds;
  struct timeval tv;

  FD_ZERO(&rfds);
  FD_SET(0, &rfds);

  tv.tv_sec = 0;
  tv.tv_usec = 0;
  ret = select(1, &rfds, NULL, NULL, &tv);
  if ((ret > 0) && FD_ISSET(0, &rfds)) {
    printf("STDIN\n");
  } else {
    printf("NOINPUT\n");
  }
}

int main(int argc, char** argv) {
  if (strcmp(argv[1],"ls")==0) {
    main_ls(argv[2], strcmp(argv[3],"repeat")==0);
  } else if (strcmp(argv[1],"stat")==0) {
    main_stat();
  } else if (strcmp(argv[1],"poll")==0) {
    main_poll();
  } else {
    fprintf(stderr, "unknown command: %s\n", argv[1]);
    return 1;
  }
  return 0;
}
