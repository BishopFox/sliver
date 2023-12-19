Sliver v1.5 and later support the loading and execution of BOFs and COFFs, generally no code changes are needed to use BOFs with a Sliver implant. However, you may need to define a manifest file so that Sliver is aware of BOF arguments and their types.

### BOF Extensions

BOF support is provided via the [COFF Loader](https://github.com/sliverarmory/COFFLoader) extension, you'll need it installed to run pretty much any BOF. However, the COFF Loader will be installed automatically if you install a BOF extension from the [armory](/docs?name=Armory).

The easiest way to install a BOF extension, for example [`nanodump`](https://github.com/sliverarmory/nanodump), is using the [armory](/docs?name=Armory) package manager:

**IMPORTANT:** BOF Extensions are installed per-sliver client, they are not stored on the server. Thus extensions are not shared across operators, each operator must install the extension to use it.

```
sliver > armory install nanodump

[*] Installing extension 'nanodump' (v0.0.5) ... done!

sliver > nanodump -h

A Beacon Object File that creates a minidump of the LSASS process.

Usage:
======
  nanodump [flags] pid dump-name write-file signature

Args:
=====
  pid         int       The PID of the process you want to dump.
  dump-name   string    The name of the dump file.
  write-file  int       1 = write file, 0 = fileless
  signature   string    Signature used for evasion, PMDM = default

Flags:
======
  -h, --help           display help
  -s, --save           Save output to disk
  -t, --timeout int    command timeout in seconds (default: 60)
```

### Converting BOFs to Sliver

Converting existing BOFs to work with Sliver is usually pretty easy, and shouldn't require any code changes. You'll need to define an `extension.json` though based on what arguments/etc. the BOF accepts.

To determine the arguments a BOF accepts and their types, you'll need to read `.cna` script that accompanies a given BOF. For example, the [CredMan](https://github.com/sliverarmory/CredManBOF/blob/main/CredMan.cna) `.cna` script is show below:

```
alias CredMan {
	local('$handle $data $args $2');

    if(size(@_) != 3){
        berror($1, "CredMan: not enough arguments,Usage: CredMan [user process id] ");
        return;
    }

    $handle = openf(script_resource("CredMan.o"));
    $data   = readb($handle, -1);
    closef($handle)

    $args = bof_pack($1,"i",$2);

    btask($1, "Running CredMan");


    beacon_inline_execute($1,$data,"go",$args);
}

beacon_command_register(
"CredMan",
"Enables SeTrustedCredManAccess Privilege in a token stolen from winlogon.exe to dump Windows Credential Manager");
```

| Type | Description                      | Sliver Type                  |
| ---- | -------------------------------- | ---------------------------- |
| b    | binary data                      | `file` (path to binary data) |
| i    | 4-byte integer                   | `int` or `integer`           |
| s    | 2-byte short integer             | `short`                      |
| z    | zero-terminated+encoded string   | `string`                     |
| Z    | zero-terminated wide-char string | `wstring`                    |

Looking at the script we can see the BOF requires a single integer argument. The corresponding Sliver `extension.json` file is shown below. Note that BOFs will always rely on the `coff-loader` extension but other kinds of extensions may not. If the `coff-loader` extension is not already installed on your system, it can be installed using `armory install coff-loader`. You can list installed extensions by simply running the `extensions` command with no arguments.

```json
{
  "name": "credman",
  "version": "1.0.0",
  "command_name": "credman",
  "extension_author": "lesnuages",
  "original_author": "jsecu",
  "repo_url": "https://github.com/sliverarmory/CredManBOF",
  "help": "Dump credentials using the CredsBackupCredentials API",
  "long_help": "",
  "depends_on": "coff-loader",
  "entrypoint": "go",
  "files": [
    {
      "os": "windows",
      "arch": "amd64",
      "path": "credman.x64.o"
    },
    {
      "os": "windows",
      "arch": "386",
      "path": "credman.x86.o"
    }
  ],
  "arguments": [
    {
      "name": "pid",
      "desc": "PID of a process started by the target user",
      "type": "int",
      "optional": false
    }
  ]
}
```

Once the manifest is defined load it into your client using `extensions load`, locally loaded extensions do not need to be cryptographically signed. The paths in the manifest should be relative to the manifest file, parent directories are not allowed.

More details can be found on the [Aliases & Extensions](/docs?name=Aliases+and+Extensions) page.

**IMPORTANT:** For BOF extensions to be properly detected by the Sliver client, the `path` must use the `.o` file extension.
