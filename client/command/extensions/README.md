Extensions
===========

Allows to load and execute 3rd party extensions.

Extensions directory structures can be arbitrary, however in the root of the directory or .tar.gz there
must be a `extension.json` or a `alias.json` file. All paths are relative to the manifest/root directory,
parent directories are not allowed. Only files listed in the manifest are copied, any other files will
be ignored.

```shell
/path/to/extension/folder/
├── extension.json
└── windows
│    └── extension.x86.dll
│    └── extension.x64.dll
└── linux
│    └── extension.x86.so
│    └── extension.x64.so
└── darwin
     └── extension.x86.dylib
     └── extension.x64.dylib
```

Here's an example manifest (i.e., the `extension.json` or a `alias.json`):

```json
{
    "name": "foo",
    "version": "1.0.0",
    "extension_author": "ac1d-burn",
    "original_author": "zer0-cool",
    "repo_url": "https://github.com/foo/bar",
    "help": "Help for foo command",
    "entrypoint": "RunFoo",
    "init" :"NimMain",
    "depends_on": "bar",
    "files": [
        {
            "os": "windows",
            "arch": "amd64",
            "path": "extension.x64.o",
        }
    ],
    "arguments": [
        {"name": "pid", "type": "int", "desc": "pid", "optional": false},
    ]
}
```

The structure is the following one:

- `name`: name of the extension, which will also be the name of the command in the sliver client
- `help`: the documentation for the new command
- `entrypoint`: the name of the exported function to call
- `files`: a list of object pointing to the extensions files to load for each architectures and operating systems
- `init`: the initialization function name (if relevant, can be omitted)
- `arguments`: an optional list of objects (for DLLs), but mandatory for BOFs
- `depends_on`: the name of an extension required by the current extension (won't load if the dependency is not loaded)

The `type` of an argument can be one of the following:

- `string`: regular ASCII string
- `wstring`: string that will be UTF16 encoded
- `int`: will be parsed as a 32 bit unsigned integer
- `short`: will be parsed as a 16 bit unsigned integer
- `file`: a string to a file path on the client side which content will be passed to the BOF