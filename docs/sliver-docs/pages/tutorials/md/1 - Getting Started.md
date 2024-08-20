# This course is intended for the 1.6 version of Sliver, which is not yet published

`sliver-server` is the binary you want to use to run the Sliver C2 server, `sliver-client` is solely a client to connect to a Sliver C2 server. Sliver server also acts as a client on its own, so you don’t necessarily run sliver server and client separately.

First time running Sliver will take a couple seconds as it's retrieving its dependencies. Consecutive executions will be much faster. Go ahead and launch the `sliver-server`.

```asciinema
{"src": "/asciinema/startup.cast", "cols": "132", "rows": "28", "idleTimeLimit": 8}
```

Let's take a couple minutes to discuss what Sliver actually is and how it's set up.

![Alt text](/images/Architecture.png)

Now that Sliver is running, lets generate and execute your first implant to try out some of the basic features of Sliver, for now we’re going to run everything on the local host.

Here's what we're going to do: 
* Generate your implant using the `generate` command as shown below.
* Start HTTP listener on port 80
* Execute implant in a separate terminal

```asciinema
{"src": "/asciinema/first-implant.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

Now let’s select our implant and run our first command using the `use` command.

```bash
[server] sliver > use
? Select a session or beacon: 
SESSION  1884a365  RELATED_EARDRUM  [::1]:49153      test.local  tester  darwin/amd64
[*] Active session RELATED_EARDRUM (1884a365-085f-4506-b28e-80c481730fd0)

[server] sliver (RELATED_EARDRUM) > pwd

[*] /Users/tester/tools
```

Once you have reached this point, go ahead and explore some of the commands listed below. In each case, first check out the command's help using the **`-h`** flag then try it out!

```bash
Exploring and interacting with the filesystem

Filesystem
  cat               Dump file to stdout
  cd                Change directory
  cp                Copy a file
  download          Download a file
  grep              Search for strings that match a regex within a file or directory
  head              Grab the first number of bytes or lines from a file
  ls                List current directory
  memfiles          List current memfiles
  mkdir             Make a directory
  mount             Get information on mounted filesystems
  mv                Move or rename a file
  pwd               Print working directory
  rm                Remove a file or directory
  tail              Grab the last number of bytes or lines from a file
  upload            Upload a file
```

```asciinema
{"src": "/asciinema/filesystem.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

Getting some environmental information
```bash
Info
  env               List environment variables
  getgid            Get session process GID
  getpid            Get session pid
  getuid            Get session process UID
  info              Get session info
  ping              Send round trip message to implant (does not use ICMP)
  screenshot        Take a screenshot
  whoami            Get session user execution context
```
Execute a binary

```asciinema
{"src": "/asciinema/execute.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

Running an interactive shell

```asciinema
{"src": "/asciinema/shell.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```
