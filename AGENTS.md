# AGENTS.md — 项目开发约定（Agent 必须遵守）

> 项目：Go + Gin + GORM 后台管理系统快速开发框架（模块 `github.com/bluefroog/admintpl`）
> 详细规范见 `docs/开发规范.md`，本文档为 Agent 开发时必须遵守的核心约定。

## 1. 模块划分与代码放置规范

模块（各自独立，互为兄弟）：

| 模块 | 目录 | 路由前缀 | 内容 |
| ---- | ---- | -------- | ---- |
| 系统管理 | `app/system` | `/system/...` | 用户/角色/部门/菜单/岗位/字典/参数/公告/日志/认证 |
| 系统监控 | `app/monitor` | `/monitor/...` | 在线用户/登录日志/操作日志/服务监控/缓存监控 |
| 定时任务 | `app/cron` | `/cron/job...`、`/cron/job-log/...` | 任务配置/调度器/任务日志/内置调用目标 |
| 代码生成 | `app/gen` | `/gen/...` | 表结构导入/生成配置/同步数据库表结构/模板渲染与下载 |

每个模块包含 `domain/`（实体 + 请求结构体）、`service/`（业务逻辑）、`controller/`（HTTP 控制器）、`initialize.go`（初始化入口，负责注册路由与模块级启动逻辑）、`test/http/`（接口测试文件）。

- 每张表对应一套文件，文件名 = 表名（蛇形），放置位置（`<模块>` 取上表目录）：
  - `<模块>/controller/<table>.go` — HTTP 控制器（只做：绑定参数 → `validate.Struct` 校验 → 调 service → 返回响应）
  - `<模块>/service/<table>.go` — 业务逻辑（用 `common/database`.DB，标准方法：GetCount/GetList/parseFilter/Create/Update/Delete/Detail + 唯一性校验）
  - `<模块>/domain/<table>.go` — 实体（`ID uint` + `XxxBase` 嵌入 `core.AuditModel` + `TableName()`）与请求结构体（`XxxSearchRequest` 嵌入 `request.PageInfo/SortInfo/core.AuditSearchRequest`、`XxxCreateRequest`、`XxxUpdateRequest` 嵌入 `request.GetById`）
  - `<模块>/test/http/<table>.http` — 接口测试文件（`###` 分隔用例）
- 新增表模型必须注册到所属模块 `domain/init.go` 的 `Models()`；路由必须注册到所属模块 `controller/init.go`。
  `bootstrap/bootstrap.go` 统一执行「数据库初始化 → `AutoMigrateAll(各模块 Models()...)` → 各模块 Initialize」，新增模块时在 `bootstrap` 中追加两行即可。
- 路由统一 `<前缀>/<表名>/<list|create|update|delete|detail>`：
  - 业务表：全量 5 个接口；日志表：仅 list/delete/detail；关系表（复合主键）：仅 list/create/delete。
  - 需登录的接口必须注册到所属模块 `controller/init.go` 的 `authorized` 分组内（已挂 `middleware.Auth()`）；
    免登录接口（`/login`、`/captcha-image`）注册在该分组之前。
  - 扩展动作（`tree`、`check-*`、`roles`、`menu`、`data-scope`、`clear`、`force-logout` 等）见 `docs/开发规范.md` 1.3 节。
- 响应：成功 `response.OK(data, c)` / `response.Success("提示", c)`；失败 `response.Fail(-1, "错误", c)`。
- 树形表（部门、菜单）删除前必须检查子节点，存在子节点禁止删除。
- 删除接口必须同步清理关联表（用户：`sys_user_role`/`sys_user_post`；角色：`sys_role_menu`/`sys_role_dept`/`sys_user_role`）；事务内执行，避免孤儿数据。
- 新增 `.go`/`.md`/`.http` 文件使用 **CRLF** 行尾。
- 其它模块（如 `app/gen` 的代码生成 `/gen/**`；仅菜单占位的 `/tool/build` 表单构建、`/tool/swagger` 系统接口）按同一模块划分方式新建 `app/<模块>`，不要塞进 `app/system`。
- **路由前缀 = 模块名**：`app/cron` → `/cron/**`、`app/gen` → `/gen/**`、`app/monitor` → `/monitor/**`，不要借用其它模块前缀（历史别名除外）。

## 2. 开发工具：just（开发 / 测试 / 打包 / 发布）

- 统一使用 `just`（命令运行器，项目根目录 `justfile`）管理开发、测试、打包、发布。
- 常用命令：
  - `just init` — 初始化：检查项目依赖（有则绿勾 ✔，缺失自动安装）+ `go mod tidy`
  - `just dev` — 开发：启动项目（`go run main.go`）
  - `just test` — 测试：`go test ./...`
  - `just vet` — 检查：`go vet ./...`
  - `just smoke [BASE_URL]` — 冒烟：对已启动实例跑读接口 + 全模块 CRUD 回归（默认 `http://127.0.0.1:8080`，脚本 `scripts/smoke.sh`）
  - `just gen-check [BASE_URL] [MYSQL_CMD]` — 代码生成深度校验：建临时表 → 导入 → 校验列配置推导 → 同步数据库 → 清理（脚本 `scripts/gen-check.sh`）
  - `just env dev` / `just env pro` — 环境切换：将 `.env.dev` / `.env.pro` 复制为 `.env`
  - `just encrypt '<明文>' [AES|SM4]` — 加密配置敏感值（默认 AES-256-GCM，可选国密 SM4-CBC），输出 `ENC(...)` 密文
  - `just encrypt-env '<明文>' [AES|SM4]` — 加密并输出可粘贴到 `.env` 的配置片段（开关/算法/密钥/密文）
  - `just decrypt 'ENC(...)'` — 解密配置敏感值（排查用）
  - `just build` — 打包：编译供镜像使用的 linux/amd64 静态二进制 `admintpl-server`
  - `just release` — 发布：构建 Docker 镜像并推送远程仓库
  - `just docs` — 生成 swagger 文档
  - `just` / `just --list` — 查看全部命令
- 开发前先 `just vet` + `just test`，开发完成打包用 `just build`，发布用 `just release`。

### 配置约定（环境变量优先）

- `config/config.yaml` **只保留配置结构**，具体值写在根目录 `.env`；yaml 中用 `${KEY}` / `${KEY:-默认值}` 引用。
- 优先级：**系统环境变量 > `.env` > `config.yaml` 默认值**；`.env` 不入库，环境模板 `.env.dev`、`.env.pro` 入库。
- 新增配置项：先在 `config/config.yaml` 加占位符，再在 `.env.dev` / `.env.pro` 补同名变量，不要在 yaml 里写死具体值。
- **敏感值加密（可选）**：`CONFIG_ENCRYPT_ENABLED=true` 时，`common/config` 会解密所有含 `ENC(...)` 片段的值
  （`common/secret`，基于 `github.com/freewu/crypto-go` 的 AES/GCM 与 SM4/CBC 封装，密钥 `CONFIG_SECRET_KEY`
  经 SHA-256 派生 32/16 字节），中间件（数据库等）读到的始终是明文，
  **不要在业务代码里再写解密逻辑**；新增敏感配置只需 `just encrypt '<明文>' [AES|SM4]` 生成密文写入 `.env`。
  算法由 `CONFIG_SECRET_ALGORITHM` 配置（默认 AES），但**密文自带算法标签**（`ENC(AES:...)` / `ENC(SM4:...)`），
  解密按标签走，切换算法不会让历史密文失效（无标签的旧密文按 AES 处理）。
  开关关闭却出现密文、密钥缺失或解密失败都会**启动失败**（快速失败，避免拿密文连库）。
- **不要直接读 `os.Getenv`**：所有配置统一走 `common/config`（`String` / `Int` / `Bool` / `IsSet`），
  才能享受 `.env`、占位符展开与密文解密。

## 3. 开发完成流程（每次开发完成后必须执行）

**每次开发完成后，必须自动执行 git commit 并 git push 到远程仓库，推送成功后才算本次开发完成。**

1. 先验证：`just vet` 与 `just test` 必须无错误（涉及接口/数据库字段改动时，另跑 `just smoke` 对已启动实例回归）。
2. 总结本次开发内容，用于 git commit message（模板已配置 `.gitmessage`，`git commit` 自动载入）：
   - 格式：`<type>(<scope>): <中文描述>` + 文件清单 + 功能点 + 验证结果
   - type：`feat` 新增 / `fix` 修复 / `refactor` 重构 / `docs` 文档 / `chore` 杂项
3. **自动提交并推送**（使用 just 一键完成）：
   - `just commit "<message>"` — 自动 `git add -A` + `git commit`（带 -m 消息），并手动 `just push`
   - `just ship` — 一键自动 `git add -A` + `git commit`（打开编辑器载入 .gitmessage 模板）+ `git push` 远程仓库
   - 推送成功后才算本次开发完成；推送失败必须排查后重新推送。

## 4. 接口 swag 注解

- **每个 HTTP 接口 handler 都必须写完整 swag 注解**：`@Summary` / `@Description` / `@Tags` / `@Param` / `@Success` / `@Router` / `@Security`。
- 参考模板：`app/system/controller/department.go`。
- 删除接口参数：业务表 `request.IdsRequest`、关系表 `XxxDeleteRequest`；详情接口参数 `request.GetById`。
- 文档生成：`swag init -o ./app/swagger`。

## 5. 认证、配置与数据写入约定

- **认证**：登录成功签发 64 位令牌并写入 `user_online` 表；请求头 `Authorization: Bearer <token>`（兼容 `token` 请求头与 `?token=`）携带。
  鉴权逻辑集中在 `app/system/middleware/auth.go`，业务方法取当前用户用 `middleware.GetLoginUser(c)` / `middleware.GetUserID(c)`，禁止再写 `// todo 通过登录服务获取`。
- **权限标识**：路径 `<模块前缀>/<子模块>/<动作>` 对应权限 `<模块前缀>:<子模块>:<动作>`
  （如 `/system/user/list` -> `system:user:list`、`/monitor/online/list` -> `monitor:online:list`），与菜单 `perms` 字段一致；
  参与校验的前缀为 `system/`、`monitor/`、`tool/`（`app/system/middleware/auth.go` 的 `PathPermission`），
  无法由路径推导的（如 `/monitor/server`、`/monitor/cache`、`/monitor/cache/list`）在 `pathPermissionAlias` 中显式映射；
  `system/user/profile|password|avatar` 为登录用户自助接口，不参与权限校验。
  注意：种子菜单的按钮权限命名（`xxx:remove` / `xxx:edit` / `cron:job:change-status` 等），
  开启 `AUTH_PERMISSION_ENABLED` 前需在 `pathPermissionAlias` 中补齐这类映射，否则非超管角色会被拒绝。
- **配置**：新增开关必须同时改 `config/config.yaml`（`${KEY:-默认值}` 占位符）与 `.env.dev` / `.env.pro`；
  `config/config.yaml` 末尾维护变量清单，`common/config/config_test.go` 会校验代码读取的键在配置文件中存在，缺失即测试失败。
- **GORM 写入零值陷阱**：带 `default:` 标签的列在「结构体方式 Create」时，若字段取零值会被数据库默认值覆盖
  （如 `sys_menu.is_frame` 默认 1、`sys_user.status` 默认 1、`sys_role.menu_check_strictly` 默认 1）。
  此类表的 Create 必须使用 `map[string]interface{}` 显式写入（参考 `service/menu.go`、`service/user.go`、`service/role.go`），
  Update 已统一使用 map。
- **基础数据初始化**：各模块的 `service/seed.go`（`app/system`、`app/cron`）仅在对应表为空时写入（幂等、事务内，判空用 `database.TableEmpty`），入口开关 `SEED_ENABLED`；
  修改种子数据后需清空对应表并重启服务验证，或直接改库核对。
- **自检**：开发完成后本地跑 `just vet` + `just test`，对已启动实例跑 `just smoke <BASE_URL>`
  （脚本会自动登录携带令牌，并校验零值字段落库、SQL 列名错配、定时任务调度等回归项）。

## 6. 数据权限约定

- **实现位置**：`app/system/service/data-scope.go`；查询入口统一为 `DataScope{}.Apply(db, loginUser, DataScopeOptions{...})`，
  角色列表用 `DataScope{}.ApplyToRoles(db, loginUser)`；列表方法一律接收 `loginUser *domain.LoginUser` 参数。
- **数据范围语义**：`1` 全部（不过滤）、`2` 自定义（`sys_role_dept`）、`3` 本部门、`4` 本部门及以下（`ancestors` 递归）、`5` 仅本人；
  多角色按 OR 合并，任一角色为 `1` 则不限制；无角色 / 超级管理员 / 未登录（`loginUser=nil`）不加条件。
- **取值校验**：角色新增、修改、数据范围保存统一走 `NormalizeDataScope`（`0` 归一为 `1`，`1~5` 合法，其它值报「数据范围不正确」）。
- **开关**：`system.data-scope.enabled`（`DATA_SCOPE_ENABLED`，默认 true）；关闭后列表不再过滤，便于排查「数据查不到」。
- **新增列表接口必须接数据权限**：带部门/用户归属的业务列表在拼接查询后调用 `DataScope{}.Apply(...)`，
  并同步给控制器方法传 `middleware.GetLoginUser(c)`；内部唯一性校验、子节点检查等场景传 `nil` 保持原有行为。
- **约定列名**：`DataScopeOptions.DeptColumn` 用表内真实列名（用户表 `department_id`、部门表 `dept_id`），
  `UserColumn` 仅在支持「仅本人」的表上传（部门表无用户列，传空则 `5` 恒不成立）。

## 7. 定时任务约定

- **cron 表达式**：统一使用 `common/cron`（Quartz 语义，6~7 段 `秒 分 时 日 月 周 [年]`），
  禁止再引入 Quartz/gcron/cron 第三方库；日与周都指定时取「或」关系，周字段 `1~7` 为周日~周六（兼容 `0` 表示周日）。
- **调用目标**：`bean.method(args)` 形式，只能调用在 `app/cron/task/task.go` 中通过 `task.Register(...)` 显式注册的方法；
  新增任务目标 = 注册目标方法 + 建任务并启用，未注册的目标会在新增/修改时被拒绝（调用目标白名单校验）。
- **调度器**：`app/cron/service/job-scheduler.go`（按表命名规范之外的模块级文件），按秒扫描、独立协程执行、
  并发策略与错过执行补偿按 `sys_job.concurrent` / `sys_job.misfire_policy` 生效，执行结果写入 `sys_job_log`；
  开关 `system.job.enabled`（`JOB_ENABLED`）。
- **任务变更必须联动调度器**：新增/修改/启停后调用 `JobScheduler{}.Refresh(id)`，删除调用 `Remove(id)`
  （`service/job.go` 已内置）；新任务表的 Create 同样使用 map 写入（`concurrent`/`status`/`misfire_policy` 有默认值）。
- **任务日志写入**：使用 map 写入（`status=0`、`exception_info=''` 是零值）；日志清理天数由 `JOB_LOG_RETAIN_DAYS` 控制。

## 8. 参数 / 字典缓存约定

- **缓存介质**：`common/cache`（可插拔后端：`file` 本地文件（默认）/ `memory` 内存 / `memcache` / `redis` 单机·主从·哨兵·集群），
  由 `system.cache.driver`（`CACHE_DRIVER`）选择，业务代码不感知后端；参数缓存名 `sys_config`（`config_key -> config_value`），
  字典缓存名 `sys_dict`（`dict_type -> []DictData`），两者都注册到缓存监控。
- **键格式**：`<CACHE_PREFIX>:<缓存名>:<业务键>`（如 `admintpl:sys_config:sys.index.skinName`）；`Cache` 内部统一补 `:`，
  禁止手写前缀拼接；清空/枚举只作用于本缓存实例的前缀。
- **跨进程后端注意**：值经 gob 编码，外部后端（memcache/redis）跨进程反序列化要求类型已注册 ——
  自定义结构体必须 `cache.RegisterType(domain.Xxx{})`（在对应 service 的 `init()` 中调用），否则缓存永远未命中。
- **后端能力差异**：`memcache` 不支持枚举键（缓存监控 `keysSupported=false`，键数量与列表为空），清空只能 FlushAll；
  写业务代码只依赖 `Set/Get/Delete/Clear`，不要假设一定可以枚举键。
- **可用性要求**：后端不可用只告警不阻塞启动，缓存读写失败必须回源数据库（业务不中断）；本地文件缓存在
  `SIGTERM`/`SIGINT` 时立即落盘（`cache.Shutdown()` 已接入 `main.go`）。
- **配置示例**：`config/config.example.yaml` + `.env.example` 展示全部缓存配置项与文字说明，改配置后同步两者。
- **实现位置**：参数在 `app/system/service/config.go`（`LoadCache`/`RefreshCache`/`GetByKeyFromCache`/`EvictCache`），
  字典在 `app/system/service/dict-data.go`（`LoadCache`/`RefreshCache`/`GetByTypeFromCache`/`EvictCache`/`ClearCache`）。
- **读取一律缓存优先**：`Config{}.GetByKey`、`DictData{}.GetByType` 先查缓存，未命中再查库并回填。
- **写操作必须失效缓存**：参数新增/修改/删除后失效对应键（键名变更时新旧键都失效）；字典数据变更后失效对应字典类型；
  字典类型修改/删除后清理相关字典缓存。新增「修改配置/字典」的接口时不要绕过 service。
- **启动预热**：`app/system/initialize.go` 在 `CACHE_ENABLED=true` 时调用 `LoadCache` 并打印条目数；
  刷新接口 `DELETE /system/config/refresh-cache`、`DELETE /system/dict-type/refresh-cache`（返回缓存条目数）。
- **排查**：`system.cache.enabled`（`CACHE_ENABLED`）置 false 时全部查询回源数据库；
  缓存内容可在「缓存监控 / 缓存列表」按 `sys_config` / `sys_dict` 过滤查看。

## 9. 代码生成（app/gen）约定

- **模块位置**：代码生成独立成 `app/gen` 模块，路由前缀 `/gen/**`（与 `app/system`、`app/monitor`、`app/cron` 平级），
  按「一表一套文件」建立 `domain/gen-table.go`、`domain/gen-table-column.go`、`service/table.go`、`controller/table.go`，
  模型在 `app/gen/domain/init.go` 的 `Models()` 中声明（`bootstrap` 汇总迁移，新增模块无需改迁移逻辑）。
- **接口**：`/gen/list`、`/gen/db-list`、`/gen/column-list`、`/gen/detail`、`/gen/import`、
  `/gen/update`、`/gen/synch-db`、`/gen/delete`、`/gen/preview`、`/gen/download`；
  `gen_table` 只能由导入接口写入，**不提供 create 接口**。
- **模板渲染**：`app/gen/service/render.go` + `app/gen/service/templates/*.tmpl`（`embed.FS` 内置，**只用标准库 `text/template`**，
  **禁止引入模板引擎/代码生成库**）；渲染结果按「项目约定」产出 domain/service/controller/接口测试/建表 SQL/接入说明，
  Go 代码必须经 `go/format` 格式化（语法错误直接报错）。新增模板必须同步补 `render_test.go` 断言与 `scripts/gen-check.sh` 校验。
- **表结构读取**：一律通过 `information_schema.tables` / `information_schema.columns` 原始查询（GORM `Table(...)`），
  不要引入第三方元数据表；已导入表用子查询排除，`qrtz_%`、`gen_%` 前缀表不参与导入。
- **默认配置推导**：字段类型/必填/编辑/列表/查询/显示类型的推导集中在 `app/gen/service/field.go`
  （`InitTable`/`InitColumnField`/`SetPkColumn`）；修改推导规则必须同步补 `field_test.go` 用例。
- **命名冲突**：`domain.Table` 已有 GORM 需要的 `TableName()` 方法，因此表名字段命名为 `Name`（JSON 仍为 `table_name`），
  不要再添加同名方法/字段。
- **事务与清理**：导入、修改、同步、删除都在事务内完成；删除与同步必须同时维护 `gen_table_column`（不留孤儿字段配置）。
- **配置项**：包路径/模块名/作者/表前缀去掉规则来自 `system.gen.*`（`GEN_PACKAGE_NAME`、`GEN_MODULE_NAME`、`GEN_AUTHOR`、
  `GEN_AUTO_REMOVE_PRE`、`GEN_TABLE_PREFIX`），禁止在代码里写死。

## 10. 服务监控 / 缓存监控约定

- **服务监控** `/monitor/server`（权限 `monitor:server:list`）：采集逻辑在 `common/sysinfo`，**只用标准库**（Linux 读 `/proc/stat`、`/proc/meminfo`、`/proc/loadavg`、`/proc/mounts` + `syscall.Statfs`），
  禁止引入 `gopsutil`；平台相关代码用构建标签拆分（`sysinfo_linux.go` / `sysinfo_other.go`），字符串解析逻辑放在平台无关文件以便单测；
  返回字段名与前端监控页面约定一致（`cpuNum`/`memTotal`/`goVersion`/`sysFiles` 等），单位 MB、使用率 %。
- **缓存监控** `/monitor/cache`、`/monitor/cache/list`、`/monitor/cache/clear`（权限 `monitor:cache:list`、`monitor:cache-list:list`）：
  基于进程内内存缓存 `common/cache`（无 Redis）；新增业务缓存后用 `cache.Register("名称", 实例)` 注册即会被监控页面列出，
  缓存列表只展示未过期条目，缓存值超过 200 字符截断。
