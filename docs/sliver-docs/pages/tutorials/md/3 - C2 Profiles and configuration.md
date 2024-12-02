# This course is intended for the 1.6 version of Sliver, which is not yet published

When generating implants sliver uses a C2Profile configuration, which will be used to generate the effective network configuration of the implant. For example if configured to use /admin and /demo as callback urls, it might use one, the other or both allowing two implants using the same configuration to still seem slightly different from a network traffic perspective.

C2 profile configurations can be seen using the `c2profile` command, which also allows import and export features.

The full list of possible configuration option can be found in the references section below, but for now lets instead customise the existing configuration.

Lets imagine we’re trying to breach a customer we've noticed uses ruby-on-rails for their applications. By default sliver will use the following extensions:

- `.woff` for staging
- `.js` for poll requests
- `.html` for key exchanges
- `.png` for close session
- `.php` for session messages

We will need to update the session messages and staging with something more realistic and replace all references to `woff` or `php` with something less suspicious like `css`, `rb` or `erb`.

We will also use a list of common Urls and filenames for Ruby on Rails like `https://github.com/danielmiessler/SecLists/blob/master/DiscoveryWeb-Content/ror.txt` for the `*_files` and `*_paths` variables. You could also reuse Urls discovered while enumerating your target's external perimeter in a similar way.

We will split the urls using a script like the example below, and then update the files and paths variables in our configuration file.

```python
import json
import math
import sys
import random


def updateProfile(c2ProfileName, urls, cookieName):
    data = open(urls).readlines()
    c2Profile = open(c2ProfileName, "r").read()
    jsonC2Profile = json.loads(c2Profile)

    paths, filenames, extensions = [], [], []
    for line in data:
        line = line.strip()
        if "." in line:
            extensions.append(line.split(".")[-1])

        if "/" in line:
            segments = line.split("/")
            paths.extend(segments[:-1])
            filenames.append(segments[-1].split(".")[0])

    extensions = list(set(extensions))
    if "" in extensions:
        extensions.remove("")
    random.shuffle(extensions)

    filenames = list(set(filenames))
    if "" in filenames:
        filenames.remove("")

    paths = list(set(paths))
    if "" in paths:
        paths.remove("")

    if len(extensions) < 5:
        print(f'Got {len(extensions)} extensions, need at least 5.')
        exit(0)

    if len(paths) < 5:
        print(f'Got {len(paths)} paths need at least 5.')
        exit(0)

    if len(filenames) < 5:
        print(f'Got {len(filenames)} paths need at least 5.')
        exit(0)

    exts = ['poll_file_ext','stager_file_ext', 'start_session_file_ext', 'session_file_ext', 'close_file_ext' ]
    for ext in exts:
        jsonC2Profile["implant_config"][ext] = extensions[0]
        extensions.pop(0)

    pathTypes = ['poll_paths','stager_paths', 'session_paths', 'close_paths' ]
    for x, pathType in enumerate(pathTypes):
        jsonC2Profile["implant_config"][pathType] =  paths[math.floor(x*(len(paths)/len(pathTypes))):math.floor((x+1)*(len(paths)/len(pathTypes)))]

    fileTypes = ['poll_files','stager_files', 'session_files', 'close_files']
    for x, fileType in enumerate(fileTypes):
        jsonC2Profile["implant_config"][fileType] = filenames[math.floor(x*(len(filenames)/len(fileTypes))):math.floor((x+1)*(len(filenames)/len(fileTypes)))]

    jsonC2Profile["server_config"]["cookies"] = [cookieName]
    c2Profile = open(c2ProfileName, "w")
    c2Profile.write(json.dumps(jsonC2Profile))
    print("C2 Profile updated !")


if __name__ == "__main__":
    if len(sys.argv) < 4:
        print("Usage: updateProfile.py myC2Profile myurls.txt cookieName")
        exit(0)

    updateProfile(sys.argv[1], sys.argv[2], sys.argv[3])
```
The example below demonstrates how to change and import a profile.

```asciinema
{"src": "/asciinema/custom_c2profile.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

At this point we can generate a new implant using our new profile.

```asciinema
{"src": "/asciinema/implant_custom_c2profile.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

If we review the debug logs of our implant we can see that the connections now use our new profile.

```asciinema
{"src": "/asciinema/implant_debug_logs.cast", "cols": "132", "rows": "28", "idleTimeLimit": 8}
```

Ideally during engagements your recon phase should inform your C2 infrastructure, reusing similar hosting providers, technologies and communication protocols can help your implant fly under the radar. 

