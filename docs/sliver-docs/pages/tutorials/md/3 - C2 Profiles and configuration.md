When generating implants Sliver uses a C2Profile configuration to define how implant callbacks will look over HTTP/s. This allowed operators to control headers, cookies proxies, URL paths and so on. When the implant is generated the sliver server will then select portions of that configuration to generate the implant C2Profile.

For example if configured to use `/admin` and `/demo` as callback urls, it might use one, the other or both depending on your configuration allowing two implants using the same configuration to still seem slightly different from a network traffic perspective.

C2 profile configurations can be seen using the `c2profile` command, which also has import and export features.

The full list of possible configuration option can be found in the references section below, but for now lets instead customise the existing configuration.

Lets imagine we’re trying to breach a customer and want to look like we're talking to Wordpress.
We would want to update the session messages and staging with something more realistic and replace all references to `woff` for example with something less suspicious like `css`, `js` or `php`.

We would also use a list of common Urls and filenames for Wordpress like `https://github.com/danielmiessler/SecLists/blob/master/Discovery/Web-Content/URLs/urls-wordpress-3.3.1.txt` for the `files` and `paths` variables. You could alternatively reuse Urls discovered while enumerating your target's external perimeter in a similar way.

You can use `c2profiles generate -f urls-wordpress-3.3.1.txt -n wordpress -i` to generate a new c2 profile using the urls we just downloaded. By default this command will use the default c2 profile as a template for all other variables, if you want to edit any of those you can export and re-import the modified profile. 

At this point we can generate a new implant using our new profile.

```asciinema
{"src": "/asciinema/implant_custom_c2profile.cast", "cols": "132", "rows": "14", "idleTimeLimit": 8}
```

If we review the debug logs of our implant we can see that the connections now use our new profile.

```asciinema
{"src": "/asciinema/implant_debug_logs.cast", "cols": "132", "rows": "28", "idleTimeLimit": 8}
```

Ideally during engagements your recon phase should inform your C2 infrastructure, reusing similar hosting providers, technologies and communication protocols can help your implant fly under the radar. 

