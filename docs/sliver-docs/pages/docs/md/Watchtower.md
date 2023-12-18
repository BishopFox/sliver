The Sliver server has a built-in capability to periodically monitor VirusTotal and IBM X-Force for implant hashes to determine whether implant builds have been uploaded to such platform.
To set it up, update your `$HOME/.sliver/configs/server.json` and set the following values in the `watchtower` object:

```json
{
  "watch_tower": {
    "vt_api_key": "YOUR_VIRUSTTOTAL_API_KEY",
    "xforce_api_key": "YOUR_XFORCE_API_KEY",
    "xforce_api_password": "YOUR_XFORCE_API_PASSWORD"
  }
}
```

Once that is done, restart the Sliver server. You can now use the `monitor start` and `monitor stop` commands to start and stop periodic monitoring of your implant builds on VirusTotal and IBM X-Force.
The server is configured to stay below the limits of requests for the free tier component for each providers. This means 4 requests per minute / 500 requests per day for VirusTotal, and 6 requests per hour for X-Force. More details on the implementation can be found [here](https://github.com/lesnuages/snitch).
