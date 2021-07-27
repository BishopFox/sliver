Extensions
====

Allows to load and execute 3rd party extensions.

Extensions must be loaded from a directory with the following architecture:

```shell
/path/to/extension/folder
   ├── extension.x64.o
   ├── extension.x86.o
   └── manifest.json
```

Here's an example manifest:

```json
{
    "name": "foo",
    "help": "Help for foo command",
    "entrypoint": "RunFoo",
    "init" :"NimMain",
    "files": [
        {
            "os": "windows",
            "files":{
                "x86": "extension.x86.o",
                "x64": "extension.x64.o",
            }
        }
    ],
    "arguments": [
        {"name": "pid", "type": "int", "desc": "pid"},
    ]
}
```

The structure is the following one:

- `name`: name of the extension, which will also be the name of the command in the sliver client
- `help`: the documentation for the new command
- `entrypoint`: the name of the exported function to call
- `files`: a list of object pointing to the extensions files to load for each architectures and operating systems
- `init`: the initialization function name (if relevant)
- `arguments`: an optional list of objects (for DLLs), but mandatory for BOFs

The `type` of an argument can be one of the following:

- `string`: regular ASCII string
- `wstring`: string that will be UTF16 encoded
- `int`: will be parsed as a 32 bit unsigned integer
- `short`: will be parsed as a 16 bit unsigned integer
- `file`: a string to a file path on the client side which content will be passed to the BOF.