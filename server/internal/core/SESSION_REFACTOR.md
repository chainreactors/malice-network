# Session Management Refactor

## 背景

Session 管理存在多个一致性问题，偏离了设计目标：**alive session 在内存管理，dead session 自动落库，重新上线则恢复到内存**。本次重构修复了 6 个 bug，补充了 54 个测试用例（含 17 个原有 SafeGo 测试），覆盖了所有状态转换路径和边界场景。

---

## 修复前后对比

### Bug 1: `Recover()` 使用错误的 key 类型（HIGH）

| | 修复前 | 修复后 |
|---|---|---|
| 数据源 | `db.ListTasks()` — 加载**全量** task | `s.Tasks.All()` — 仅当前 session 的 task |
| Map key | `*clientpb.Task` 指针 | `task.Id` (`uint32`) |
| 影响 | `GetResp(uint32)` 永远匹配不到，恢复的 task response channel 不可达 | 与 `GetResp`/`StoreResp` key 类型一致，channel 可正常使用 |

```go
// 修复前
func (s *Session) Recover() error {
    modelTasks, _ := db.ListTasks()          // 全量加载
    for _, task := range modelTasks.Tasks {
        s.responses.Store(task, ch)           // key = *clientpb.Task
    }
}

// 修复后
func (s *Session) Recover() error {
    for _, task := range s.Tasks.All() {     // 仅当前 session
        if task.Cur < task.Total {
            s.responses.Store(task.Id, ch)    // key = uint32
        }
    }
}
```

### Bug 2: `GetSession` RPC 意外复活 dead session（HIGH）

| | 修复前 | 修复后 |
|---|---|---|
| 行为 | 从 DB 加载后 `RecoverSession()` + `Sessions.Add()` | 直接返回 `dbSess.ToProtobuf()`，不加入内存 |
| 副作用 | 客户端仅查询就把 dead session 拉回内存，下一轮 ticker 又杀掉，引起事件抖动 | 只读查询，无副作用 |
| 恢复入口 | GetSession / Checkin / Register | 仅 Checkin / Register |

```go
// 修复前
func GetSession(...) {
    dbSess, _ := db.FindSession(req.SessionId)
    session, _ = core.RecoverSession(dbSess)
    core.Sessions.Add(session)          // 副作用：加入内存
    return session.ToProtobuf(), nil
}

// 修复后
func GetSession(...) {
    dbSess, _ := db.FindSession(req.SessionId)
    return dbSess.ToProtobuf(), nil     // 只读返回
}
```

### Bug 3: `Remove()` 不 Cancel context（MEDIUM）

| | 修复前 | 修复后 |
|---|---|---|
| context | 不取消 — 派生的 task context 永远不会被通知 | `parentSession.Cancel()` — 所有派生 context 级联取消 |
| goroutine | 等待 `sess.Ctx` 的 goroutine 泄漏 | 干净退出 |

```go
// 修复前
func (s *sessions) Remove(sessionID string) {
    parentSession.ResetKeepalive()
    s.active.Delete(parentSession.ID)
    // 缺少 Cancel()
}

// 修复后
func (s *sessions) Remove(sessionID string) {
    parentSession.ResetKeepalive()
    parentSession.Cancel()              // 级联取消所有 task context
    s.active.Delete(parentSession.ID)
}
```

### Bug 4: Ticker 死代码 + 执行顺序错误（MEDIUM）

| | 修复前 | 修复后 |
|---|---|---|
| `sessModel` | 创建后设 `IsAlive=false`，但从未用于保存（死代码） | 移除 |
| 执行顺序 | `Remove()` → `Save()` | `Save()` → `Remove()` — 先落库再移出内存 |
| 保存策略 | 每轮 ticker 对**所有** alive session 都 Save | 仅在 session 死亡时 Save；alive session 在 Checkin 时 Save |

```go
// 修复前
for _, session := range newSessions.All() {
    sessModel := session.ToModel()           // 创建了但从未使用
    if !session.isAlived() {
        sessModel.IsAlive = false            // 修改了但从未保存
        newSessions.Remove(session.ID)       // 先移除
    }
    err := session.Save()                    // 后保存（所有 session 都保存）
}

// 修复后
for _, session := range newSessions.All() {
    if !session.isAlived() {
        session.Save()                       // 先保存
        session.Publish(CtrlSessionDead, ...)
        newSessions.Remove(session.ID)       // 后移除
    }
}
```

### Bug 5: Re-register 不发事件（LOW）

| | 修复前 | 修复后 |
|---|---|---|
| 事件 | `sess.Publish(CtrlSessionReborn, ...)` 被注释掉 | 发送 `CtrlSessionUpdate` 事件 |
| 客户端感知 | 无法感知 re-register | 收到 session_update 通知 |

### Bug 6: Checkin 不 Save — LastCheckin 可能丢失（LOW）

| | 修复前 | 修复后 |
|---|---|---|
| 持久化 | LastCheckin 仅更新内存，依赖 ticker 30s 后批量 Save | Checkin 后立即 `sess.Save()` |
| 崩溃恢复 | server 崩溃时 LastCheckin 最多延迟 30s，可能误判 session 为 dead | LastCheckin 实时落库 |

---

## 新增可测试性设施

### DB 函数变量

在 `session.go` 中新增包级别函数变量，允许测试时替换 DB 调用：

```go
var (
    sessionDBSave        = func(s *models.Session) error { return db.SaveSessionModel(s) }
    sessionDBGetArtifact = func(name string) (*models.Artifact, error) { return db.GetArtifactByName(name) }
    sessionDBGetProfile  = func(name string) (*models.Profile, error) { return db.GetProfileByName(name) }
)
```

测试中通过 `installTestDBMocks()` 一键替换为 no-op mock，无需真实数据库。

---

## 状态机

```
                         ┌──────────────────────────┐
                         │      NOT EXIST           │
                         └────────────┬─────────────┘
                                      │ Register RPC
                                      │ CtrlSessionRegister
                                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          ALIVE (内存)                               │
│                                                                     │
│  Checkin RPC → LastCheckin++ → Save() → CtrlSessionCheckin          │
│  Register RPC (re-register) → Update() → CtrlSessionUpdate         │
└──────┬──────────────────────────────────────┬───────────────────────┘
       │                                      │
       │ Ticker: isAlived()=false             │ 用户 delete
       │ Save → CtrlSessionDead → Remove      │ Remove + db.RemoveSession
       │ Cancel context                        │ Cancel context
       ▼                                      ▼
┌─────────────────────────┐    ┌──────────────────────────────┐
│       DEAD (仅DB)        │    │      SOFT-DELETED (仅DB)     │
│  IsAlive=false           │    │  IsRemoved=true              │
│  IsRemoved=false         │    │                              │
│                          │    │                              │
│  GetSession RPC:          │    │                              │
│  返回 DB protobuf         │    │                              │
│  不改变状态 ✓             │    │                              │
└──────────┬───────────────┘    └──────────────┬───────────────┘
           │ Checkin RPC                       │ Checkin RPC
           │ RecoverSession                    │ RecoverRemovedSession
           │ CtrlSessionReborn                 │ CtrlSessionReborn
           └──────────────┬────────────────────┘
                          ▼
                    ALIVE (内存) ← 新 Context
```

### 关键不变量

1. **内存中的 session 一定是 alive** — ticker 自动清除 dead session
2. **只有 implant 主动连接才会将 session 拉回内存** — GetSession 查询无副作用
3. **Remove 一定会 Cancel context** — task context 级联取消，防止 goroutine 泄漏
4. **Checkin 时立即 Save** — LastCheckin 不因 server 崩溃丢失
5. **先 Save 再 Remove** — dead session 的最终状态一定落库
6. **Recover 使用 uint32 key** — response channel 可被 GetResp 正确检索

---

## 已知设计边界

| 场景 | 当前行为 | 说明 |
|------|---------|------|
| 空 Expression | `cronexpr.Parse("")` 失败 → `isAlived()` 返回 `true` | 无 timer 的 session 永不超时，需手动管理 |
| `Add()` 覆盖同 ID session | 旧 session 的 context 不会被 Cancel | 当前所有 re-register 路径用同一对象，不存在泄漏 |
| Register (而非 Checkin) 恢复 dead session | 创建全新 session，旧数据丢失 | Register = implant 重启，丢弃旧状态是预期行为 |
| SysInfo RPC 对 dead session | 直接返回 "not found" | 正常流程中 Register 后立即 SysInfo，间隔极短不会被 ticker 清除 |
| 并发 Remove (ticker + user delete) | 两者都安全执行，Cancel 幂等 | `sync.Map` 保证并发安全，Cancel 多次调用无害 |

---

## 测试覆盖

### 文件结构

```
server/internal/core/
├── session.go                    # 核心实现（已修改）
├── session_test.go               # 单元测试（新增，14 个）
├── session_lifecycle_test.go     # 生命周期 + 边界测试（新增，23 个）
└── safe_test.go                  # 原有 SafeGo 测试（17 个）
```

### 测试分类

| 分类 | 数量 | 覆盖范围 |
|------|------|---------|
| **Session CRUD** | 4 | Add/Get/Remove/All 基本操作 |
| **isAlived 判定** | 7 | BindPipeline / 过期 / 最近 / nil / 空表达式 / 零 Jitter / 边界 90s |
| **Recover** | 2 | uint32 key 正确性 / 混合完成/未完成 task |
| **Task 管理** | 2 | 序号递增 / response channel CRUD |
| **DB Mock** | 2 | Save 调用 / Checkin 时间戳持久化 |
| **Keepalive** | 2 | 状态切换 / 死亡时重置 |
| **Ticker 生命周期** | 4 | 标记死亡 / 保持存活 / 批量死亡 / 混合状态 |
| **完整周期** | 3 | Register→Checkin→Dead→Reborn / 长时间沉默→恢复 / 快速抖动 |
| **并发安全** | 3 | Checkin+Ticker 并发 / 并发 Remove / 20 轮压力测试 |
| **边界防御** | 5 | 双重 Remove / Add(nil) / ID 覆盖 / response 清理 / context 级联 |
| **设计验证** | 3 | Get 不复活 / Remove→ReAdd / response channel 死亡后保留 |

### 运行

```bash
# 运行全部测试
go test ./server/internal/core/ -v -timeout 60s

# 仅运行 session 相关
go test ./server/internal/core/ -v -run "TestSession|TestLifecycle|TestEdge"
```

---

## 修改文件清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `server/internal/core/session.go` | 修改 | Bug 1/3/4 修复 + DB 函数变量 |
| `server/rpc/rpc-session.go` | 修改 | Bug 2 修复 (GetSession) |
| `server/rpc/rpc-implant.go` | 修改 | Bug 5/6 修复 (Register/Checkin) |
| `server/internal/core/session_test.go` | 新增 | 14 个单元测试 |
| `server/internal/core/session_lifecycle_test.go` | 新增 | 23 个生命周期/边界测试 |
