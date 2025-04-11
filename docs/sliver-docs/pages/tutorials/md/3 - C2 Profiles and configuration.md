# This course is intended for the 1.6 version of Sliver, which is not yet published

When generating implants Sliver uses a C2Profile configuration to define how implant callbacks will look over HTTP/s. This allowed operators to control headers, cookies proxies, URL paths and so on. When the implant is generated the sliver server will then select portions of that configuration to generate the implant C2Profile.

For example if configured to use `/admin` and `/demo` as callback urls, it might use one, the other or both depending on your configuration allowing two implants using the same configuration to still seem slightly different from a network traffic perspective.

C2 profile configurations can be seen using the `c2profile` command, which also has import and export features.

The full list of possible configuration option can be found in the references section below, but for now lets instead customise the existing configuration.

Lets imagine we’re trying to breach a customer and want to look like we're talking to Wordpress.
We would want to update the session messages and staging with something more realistic and replace all references to `woff` for example with something less suspicious like `css`, `js` or `php`.

We would also use a list of common Urls and filenames for Wordpress like `https://github.com/danielmiessler/SecLists/blob/master/Discovery/Web-Content/URLs/urls-wordpress-3.3.1.txt` for the `files` and `paths` variables. You could alternatively reuse Urls discovered while enumerating your target's external perimeter in a similar way.

We will split the urls using a script like the example below, and then update the files and paths variables in our configuration file.

```python
import json
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
    
    jsonC2Profile["implant_config"]["extensions"] = extensions
    jsonC2Profile["implant_config"]["paths"] =  paths
    jsonC2Profile["implant_config"]["files"] = filenames
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

