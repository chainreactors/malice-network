---
name: persist
description: Establish persistence mechanism on the target
---
Establish a persistence mechanism on the target system. Choose the most appropriate method based on the current user's privilege level and OS.

## Methods (select based on context)

**Linux (unprivileged)**:
- Crontab entry (`crontab -e`)
- ~/.bashrc or ~/.profile command injection
- Systemd user service (~/.config/systemd/user/)
- XDG autostart (~/.config/autostart/)

**Linux (root)**:
- Systemd system service (/etc/systemd/system/)
- /etc/cron.d/ entry
- rc.local modification

**Windows (unprivileged)**:
- Registry Run key (HKCU\Software\Microsoft\Windows\CurrentVersion\Run)
- Startup folder shortcut
- Scheduled task (user context)

**Windows (admin)**:
- Registry Run key (HKLM)
- Scheduled task (SYSTEM context)
- Service creation

## Rules

- First detect OS and privilege level
- Choose the LEAST intrusive method that works
- The payload to persist is: `$0`
- Use callback interval/schedule: $1
- Output the exact persistence mechanism used and how to remove it later

$ARGUMENTS
