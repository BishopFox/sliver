Log
======

The log package contains some wrappers around a logrus infrastructure, and has access to the console prompt: it can then print logs
in an asynchronous way, while never messing up with the user console. User packages can use a ready to use logrus logger, and 
everything will be handled transparently.

The log system is controllable (with log levels for each component) via the `log` command on the console.
Settings apply only to the console being used: it does not affect other user's consoles.

 * `client.go`  - In charge of synchronizing the prompt, the commands and the incoming logs, and then to print them.
 * `log.go`     - Basic logrus logger settings, and init method.
