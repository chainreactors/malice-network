## 基本命令

### background

**Command**

```
background
```

**About:** 返回到根上下文

---

### version

**Command**

```
version
```

**About:** 显示服务器版本

---



### observe

**Command**

```
observe <session id>
```

**About:** 观察会话

**Flags:**

- `-r`, `--remove`: 移除观察。
- `-l`, `--list`: 列出所有观察者。

---

### login

**Command**

```
login
```

![login](../assets/login.gif)

**About:** 上下选择对应的用户文件，按下回车登录到服务器



## Session管理

### sessions

**Command**

```
sessions
```

**About:** 列出会话，选择对应session按下回车进行连接。

![](../assets/YUGBbuPRyoikQDxjNdrcZnaFnFd.jpg)

---

### tasks

**Command**

```
tasks
```

**About:** 列出任务

![](../assets/EIUjbCi2LoIo9WxP2tzcJe0vnng.png)

---

### use

**Command**

```
use <sid>
```

**About:** 使用会话

**Arguments:**

- `sid`: 要使用的会话ID。

---

### note

**Command**

```
note <name>
```

**About:** 添加注释到会话

**Arguments:**

- `name`: 会话的新标记名。

**Flags:**

- `--id`: 会话ID。

---

### group

**Command**

```
group <group name>
```

**About:** 分组会话

**Arguments:**

- `group name`: 会话的新组名。

 **Flags:**

- `--id`: 会话ID。

---

### remove

**Command**

```
remove
```

**About:** 删除会话

**Flags:**

- `--id`: 会话ID。

---

## Server管理


### listener

**Command**

```
listener
```

**About:** 列出所有listener

![image-20240816190442913](../assets/image-20240816190442913.png)

**Subcommands:**

- [tcp](#tcp)
- [website](#website)

---

### tcp

**Command**

```
tcp <listener_id>
```

**About:** 列出listener中的 TCP 流水线

**Arguments:**

- `listener_id`: listener id。

**Subcommands:**

- [start](#tcp-start)
- [stop](#tcp-stop)

---

### tcp start

**Command**

```
tcp start <name> <listener_id> <host> <port>
```

**About:** 启动 TCP  pipeline

**Arguments:**

- `name`: TCP  pipeline名称。
- `listener_id`: listener id。
- `host`: TCP  pipeline主机。
- `port`: TCP  pipeline端口。

**Flags**

- `--cert_path`: TCP  pipeline tls证书路径。
- `--key_path`: TCP  pipeline tls密钥路径。

**Arguments:** None

---

### tcp stop

**Command**

```
 tcp stop <name> <listener_id>
```

**About:** 停止 TCP pipeline

**Arguments:**

- `name`: TCP  pipeline名称。
- `listener_id`: listener id。

---

### website 🛠️

**Command**

```
website <listener_id>
```

**About:** 列出listener中的网站

**Arguments:**

- `listener_id`: listener id。

**Subcommands:**

- [start](#website-start)
- [stop](#website-stop)

---

### website start 🛠️

**Command**

```
website start <name> <listener_id> <port> <web-path> <content-path> <content-type>
```

**About:** 启动网站

**Arguments:**

- `name`: 网站名称。
- `listener_id`: listener id。
- `port`: 网站端口。
- `web-path`: 网站url根路径。
- `content-path`: 网站静态内容文件的路径。
- `content-type`: 网站内容类型。

**Flags**

- `--cert_path`: website tls证书路径。
- `--key_path`: website tls密钥路径。

---

### website stop 🛠️

**Command**

```
website stop <name> <listener_id>
```
**About:** 停止网站

**Arguments:**

- `name`: website 名称。
- `listener_id`: listener id。

---

## 插件管理

### list_module

**Command**

```
list_module
```

**About:** 列出模块

---

### load_module

**Command**

```
load_module <name> <path>
```

**About:** 加载模块

**Arguments:**

- `name`: 要加载的模块名称。

- `path`: 模块文件的路径。

---


### alias

**Command**

```
alias
```

**About:** 列出现有的别名

---

### alias load

**Command**

```
alias load <dir-path>
```

**About:** 加载命令别名

**Arguments:**

- `dir-path`: 别名目录的路径。

---

### alias install

**Command**

```
alias install <path>
```

**About:** 安装命令别名

**Arguments:**

- `path`: 别名目录或 tar.gz 文件的路径。

---

### alias remove

**Command**

```
alias remove <name>
```

**About:** 删除别名

**Arguments:**

- `name`: 要删除的别名名称。

---

### armory

**Command**

```
armory
```

**About:** 列出可用的武器库包

![armory](../assets/armory.gif)

**Flags:**

- `-p, --proxy <proxy>`: 代理 URL。
- `-t, --timeout <timeout>`: 超时时间。
- `-i, --insecure`: 禁用 TLS 验证。
- `--ignore-cache`: 忽略缓存。

---

### armory install

**Command**

```
armory install <name>
```

**About:** 安装命令武器库

**Flags:**

- `-a, --armory <armory>`: 要安装的武器库名称（默认："Default"）。
- `-f, --force`: 强制安装包，如果存在则覆盖。
- `-p, --proxy <proxy>`: 代理 URL。

**Arguments:**

- `name`: 要安装的包或捆绑包名称。

---

### armory update

**Command**

```
armory update
```

**About:** 更新已安装的武器库包

**Flags:**

- `-a, --armory <armory>`: 要更新的武器库名称。
- `-p, --proxy <proxy>`: 代理 URL。

---

### armory search

**Command**

```
armory search <name>
```

**About:** 搜索武器库包

**Arguments:**

- `name`: 要搜索的包名称。

---

### extension

**Command**

```
extension
```

**About:** 扩展命令

---

### extension list

**Command**

```
extension list
```

**About:** 列出所有扩展

---

### extension load

**Command**

```
extension load
```

**About:** 加载扩展

**Arguments:**

- `dir-path`: 扩展目录的路径。

---

### extension install

**Command**

```
extension install <path>
```

**About:** 安装扩展

**Arguments:**

- `path`: 扩展目录或 tar.gz 文件的路径。

---

### extension remove

**Command**

```
extension remove <name>
```

**About:** 删除扩展

**Arguments:**

- `<name>`: 要删除的扩展名称。

---

## Implant 交互

### pwd

**Command**

```
pwd
```

**About:** 打印远程工作目录

---

### cat

**Command**

```
cat <name>
```

**About:** 打印远程文件内容

**Arguments:**

- `name` `: 要打印的文件名。

---

### cd

**Command**

```
cd <path>
```

**About:** 切换远程目录

**Arguments:**

- `path`: 要切换的目录路径。

---

### chmod

**Command**

```
chmod <mode> <path>
```

**About:** 更改远程文件模式

**Arguments:**

- `mode`: 新的文件模式。

- `path`: 要更改模式的文件路径。

---

### chown

**Command**

```
chown <uid> <path>
```

**About:** 更改远程文件所有者

**Arguments:**

- `uid`: 用户ID
- `path`: 要更改所有者的文件路径。

**Flags:**

- `--gid`, `-g`: 新的组ID。
- `--recursive`, `-r`: 递归应用更改。

---

### cp

**Command**

```
cp <source_file> <target_file>
```

**About:** 复制远程文件

**Arguments:**

- `source`: 要复制的源文件。
- `target`: 复制后的目标文件。

---

### ls

**Command**

```
ls <path>
```

**About:** 列出远程目录内容

**Arguments:**

- `path`: 要列出的目录路径。

---

### mkdir

**Command**

```
mkdir <path>
```

**About:** 创建远程目录

**Arguments:**

- `path`: 要创建新目录的路径。

---

### mv

**Command**

```
mv <source_file> <target_file>
```

**About:** 移动远程文件

**Arguments:**

- `source_file`: 要移动的源文件。
- `target_file`: 移动后的目标文件。

---

### rm

**Command**

```
rm <name>
```

**About:** 删除远程文件

**Arguments:**

- `name`: 要删除的文件名。

---

### whoami

**Command**

```
whoami
```

**About:** 打印当前用户

---

### kill

**Command**

```
kill <pid>
```

**About:** 杀死远程进程

**Arguments:**

- `pid`：要杀死的进程ID。

---

### ps

**Command**

```
ps
```

**About:** 列出远程进程

---

### env

**Command**

```
env
```

**About:** 列出远程环境变量

---

### setenv

**Command**

```
setenv <env> <value>
```

**About:** 设置远程环境变量

**Arguments:**

- `env`: 要设置的环境变量。
- `value`: 要分配给环境变量的值。

---

### unsetenv

**Command**

```
unsetenv <env>
```

**About:** 取消设置远程环境变量

**Arguments:**

- `env`: 要取消设置的环境变量。

---

### netstat

**Command**

```
netstat
```

**About:** 列出远程网络连接

---

### info

**Command**

```
info
```
**About:** 获取基本远程系统信息

---

### download

**Command**

```
download <name> <path>
```

**About:** 下载文件

**Arguments:**

- `name`: 下载到本地的文件名。
- `path`: 要下载的文件的路径。

---

### sync

**Command**

```
sync <task_id>
```

**About:** 同步文件

Arguments:

- `taskID`: 同步操作的任务ID。

---

### upload

**Command**

```
upload <source> <destination>
```

**About:** 上传文件

**Arguments:**

- `source`: 文件的源路径。
- `destination`: 上传后的目标路径。

**Flags:**

- `--priv`: 文件权限，默认是 `0o644`。
- `--hidden`: 将文件名标记为隐藏。

---


### exec

**Command**

exec

**About:** 执行命令

**Flags:**

- `-o`, `--output`: 捕获命令输出（默认：true）。
- `-t`, `--timeout`: 命令超时时间，以秒为单位（默认：`assets.DefaultSettings.DefaultTimeout`）。
- `-O`, `--stdout`: 获取标准输出内容。
  - `-E`, `--stderr`: 获取标准错误内容。

**Arguments:**

- `command`: 要执行的命令。
- `arguments`: 命令的参数。

---

### execute_assembly

**Command**

```
execute_assembly <path>
```

**About:** 在子进程中加载并执行 .NET 程序集（仅限Windows）

**Arguments:**

- `path`: 程序集文件的路径。
- `args`: 传递给程序集入口点的参数（默认：空列表）。

**Flags**

- `-o`,`--output`: 需要输出。

---

### execute_shellcode

**Command**

```
execute_shellcode <path>
```

**About:** 在 malefic进程中执行给定的 shellcode

**Arguments:**

- `path`: shellcode 文件的路径。
- `args`: 传递给入口点的参数（默认：`notepad.exe`）。

**Flags**

- `-p`, `--ppid`: 要注入的进程 ID（0 表示注入自身）。
- `-b`, `--block_dll`: 阻止 DLL 注入。
- `-s`, `--sacrifice`: 需要牺牲进程。
- `-a`, `--argue`: 参数。

---

### inline_shellcode

**Command**

```
inline_shellcode <path>
```

**About:** 在 IOM 中执行给定的 inline shellcode

**Arguments:**

- `path`: shellcode 文件的路径。

---

### execute_dll

**Command**

```
execute_dll <path>
```

**About:** 在牺牲进程中执行给定的 DLL

**Arguments:**

- `path`: DLL 文件的路径。
- `args`: 传递给入口点的参数（默认：`C:\\Windows\\System32\\cmd.exe\x00`）。

**Flags**

- `-p`, `--ppid`: 要注入的进程 ID（0 表示注入自身）。
- `-b`, `--block_dll`: 阻止 DLL 注入。
- `-s`, `--sacrifice`: 需要牺牲进程。
- `-e`, `--entrypoint`: 入口点。
- `-a`, `--argue`: 参数。

---

### inline_dll 🛠️

**Command**

```
inline_dll <path>
```

**About:** 在当前进程中执行给定的 inline DLL

**Arguments:**

- `path`: DLL 文件的路径。
- `args`: 传递给入口点的参数。

**Flags**

- `-p`, `--ppid`: 要注入的进程 ID（0 表示注入自身）。
- `-b`, `--block_dll`: 阻止 DLL 注入。
- `-s`, `--sacrifice`: 需要牺牲进程。
- `-a`, `--argue`: 参数。

---

### execute_pe

**Command**

```
execute_pe <path>
```

**About:** 在牺牲进程中执行给定的 PE

**Arguments:**

- `path`: PE 文件的路径。
- `args`: 传递给入口点的参数（默认：`notepad.exe`）。

**Flags**

- `-p`, `--ppid`: 要注入的进程 ID（0 表示注入自身）。
- `-b`, `--block_dll`: 阻止 DLL 注入。
- `-s`, `--sacrifice`: 需要牺牲进程。
- `-a`, `--argue`: 参数。

---

### inline_pe 🛠️

**Command**

```
inline_pe <path>
```

**About:** 在当前进程中执行给定的 inline PE

**Arguments:**

- `path`: PE 文件的路径。
- `args`: 传递给入口点的参数。

---

### bof

**Command**

```
bof <path>
```

**About:** 加载并执行 Bof（仅限Windows）

**Arguments:**

- `path`: Bof 文件的路径。
- `args`: 传递给入口点的参数。

**Flags**

- `-t`, `--timeout`: 命令超时时间，以秒为单位。

---

### powershell

**Command**

```
powershell <path>
```

**About:** 加载并执行 powershell（仅限Windows）

**Arguments:**

- `path`: powershell文件的路径。

- `args`: 传递给入口点的参数。

**Flags**

- `-t`, `--timeout`: 命令超时时间，以秒为单位。

---

