Extensions
====

Allows to load and execute 3rd party extensions.

Extensions must be loaded from a directory with the following architecture:

```shell
/path/to/extension/folder/
├── manifest.json
└── windows
    ├── 386
    │   └── extension.x86.dll
    └── amd64
        └── extension.x64.dll
```

The extension folder structure must follow the `GOOS/GOARCH/` scheme.

Here's an example manifest:

```json
[
{
    "name": "foo",
    "help": "Help for foo command",
    "entrypoint": "RunFoo",
    "init" :"NimMain",
    "dependsOn": "bar",
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
        {"name": "pid", "type": "int", "desc": "pid", "optional": false},
    ]
}
]
```

The structure is the following one:

- `name`: name of the extension, which will also be the name of the command in the sliver client
- `help`: the documentation for the new command
- `entrypoint`: the name of the exported function to call
- `files`: a list of object pointing to the extensions files to load for each architectures and operating systems
- `init`: the initialization function name (if relevant, can be omitted)
- `arguments`: an optional list of objects (for DLLs), but mandatory for BOFs
- `dependsOn`: the name of an extension required by the current extension (won't load if the dependency is not loaded)

The `type` of an argument can be one of the following:

- `string`: regular ASCII string
- `wstring`: string that will be UTF16 encoded
- `int`: will be parsed as a 32 bit unsigned integer
- `short`: will be parsed as a 16 bit unsigned integer
- `file`: a string to a file path on the client side which content will be passed to the BOF