---
name: creds
description: Harvest credentials, tokens, and secrets from the target
---
Search the target system for credentials, tokens, and secrets. READ-ONLY — do not modify any files.

## Search locations

**Files**:
- SSH: ~/.ssh/id_*, ~/.ssh/config, ~/.ssh/known_hosts
- Cloud: ~/.aws/credentials, ~/.azure/, ~/.config/gcloud/, ~/.kube/config
- Git: ~/.gitconfig, .git/config (look for credentials in remote URLs)
- Environment: .env, .env.local, .env.production files
- Application configs: database.yml, wp-config.php, settings.py, appsettings.json, application.properties
- Package managers: ~/.npmrc, ~/.pypirc, ~/.docker/config.json, ~/.m2/settings.xml
- Password managers: KeePass databases (.kdbx), browser password stores

**Runtime**:
- Environment variables containing KEY, TOKEN, SECRET, PASSWORD, CREDENTIAL, API
- Process command lines with embedded credentials (ps aux)
- Shell history entries containing passwords or tokens

**Windows-specific**:
- Saved WiFi passwords (netsh wlan show profiles)
- Registry stored credentials
- Unattend.xml / sysprep files
- IIS web.config with connection strings

## Output

For each credential found, report:
- Source (file path or command)
- Type (SSH key, API token, password, etc.)
- Value preview (first/last 4 chars only, mask the rest: `sk-ab...xyz`)
- Scope (what service/system it accesses)

$ARGUMENTS
