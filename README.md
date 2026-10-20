# 项目说明
> 使用 gin + gorm 实现一个后台管理系统快速开发框架

# 内置功能

* [已实现]登录认证：账号密码登录（bcrypt 校验）、令牌签发与校验、退出登录、图形验证码、登录失败锁定。
* [已实现]用户管理：用户是系统操作者，该功能主要完成系统用户配置。
* [已实现]部门管理：配置系统组织机构（公司、部门、小组），树结构展现支持数据权限。
* [已实现]岗位管理：配置系统用户所属担任职务。
* [已实现]菜单管理：配置系统菜单，操作权限，按钮权限标识等。
* [已实现]角色管理：角色菜单权限分配、设置角色按机构进行数据范围权限划分。
* [已实现]字典管理：对系统中经常使用的一些较为固定的数据进行维护。
* [已实现]参数管理：对系统动态配置常用参数（参数值进程内缓存，支持一键刷新缓存）。
* [已实现]字典/参数缓存：字典数据按类型缓存、参数按键名缓存，变更即失效，支持一键刷新缓存。
* [已实现]操作日志：系统正常操作日志记录和查询；系统异常信息日志记录和查询。
* [已实现]登录日志：系统登录日志记录查询包含登录异常。
* [已实现]在线用户：当前系统中活跃用户状态监控，支持强退。
* [已实现]缓存监控：查询内存缓存的键值、读写统计，支持按缓存名称/缓存键过滤与清空。
* [已实现]服务监控：监视当前系统 CPU、内存、磁盘、Go 运行时等信息。
* [已实现]数据权限：按角色划分数据范围（全部/自定义/本部门/本部门及以下/仅本人），作用于用户、部门、角色列表。
* [已实现]定时任务：任务配置（增删改查/启停/立即执行一次）+ 调度执行 + 任务日志；cron 表达式为 Quartz 语义，调用目标只能在已注册的方法集合内调用。
* [已实现]代码生成：数据库表结构导入、生成配置维护（表配置/字段配置/树表参数）、同步数据库表结构。
* [已实现]代码生成模板渲染：按生成配置渲染 domain/service/controller/接口测试/建表语句/接入说明，支持在线预览与 zip 下载。

# 项目初始化
```bash
# 初始化（检查依赖 + 下载依赖）
just init
```

# 登录认证

采用「令牌（token）+ 在线用户表」实现认证，登录成功后签发 64 位随机令牌，令牌同时写入 `user_online` 表，
后续请求通过请求头携带令牌，鉴权中间件校验令牌有效性、有效期与账号状态。

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/captchaImage` | GET | 获取图形验证码（返回 uuid + base64 PNG，验证码开关关闭时 `captchaEnabled=false`） |
| `/login` | POST | 登录（`{username, password, code?, uuid?}`），返回 `{token, expiresIn}` |
| `/logout` | POST | 退出登录（删除令牌对应的在线记录） |
| `/getInfo` | GET | 获取当前登录用户信息、角色标识与权限标识 |
| `/getRouters` | GET | 获取当前用户的前端路由（Layout/ParentView、外链等前端约定结构） |
| `/monitor/online/*` | - | 在线用户列表 / 详情 / 强退（强退后令牌立即失效） |

令牌携带方式（三选一，优先级从上到下）：

```bash
# 1. Authorization: Bearer <token>
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8080/getInfo
# 2. 自定义请求头
curl -H "token: $TOKEN" http://127.0.0.1:8080/getInfo
# 3. 查询参数
curl "http://127.0.0.1:8080/getInfo?token=$TOKEN"
```

配置开关（`.env.dev` / `.env.pro` 中调整，详见 `config/config.yaml` 末尾变量清单）：

| 变量 | 默认值 | 说明 |
| ---- | ------ | ---- |
| `CONFIG_ENCRYPT_ENABLED` | false | 配置敏感值是否使用 AES 密文（详见「配置加密」） |
| `CONFIG_SECRET_KEY` | 空 | 配置加密密钥（任意字符串，经 SHA-256 派生 32 字节密钥） |
| `AUTH_ENABLED` | true | 关闭后所有业务接口（`/system/...`、`/monitor/...`）免登录（仅用于本地调试） |
| `AUTH_TOKEN_EXPIRE` | 720 | 令牌有效期（分钟） |
| `AUTH_MAX_RETRY` | 5 | 连续登录失败次数上限，超过后锁定账号 |
| `AUTH_LOCK_TIME` | 10 | 登录失败锁定时间（分钟） |
| `AUTH_PERMISSION_ENABLED` | false | 开启后按请求路径推导权限标识（如 `system/user/list` -> `system:user:list`、`monitor/online/list` -> `monitor:online:list`）做角色权限校验 |
| `CAPTCHA_ENABLED` | dev=false / pro=true | 是否开启登录图形验证码 |
| `CAPTCHA_LENGTH` / `CAPTCHA_EXPIRE` | 4 / 120 | 验证码位数 / 有效期（秒） |
| `SEED_ENABLED` | dev=true / pro=false | 空库时自动初始化基础数据（部门/岗位/角色/用户/菜单/字典/参数/定时任务） |
| `JOB_ENABLED` | true | 是否启动定时任务调度器（在 `app/cron` 模块启动时生效） |

基础数据默认账号（`SEED_ENABLED=true` 且对应表为空时写入，幂等）：

| 账号 | 密码 | 说明 |
| ---- | ---- | ---- |
| `admin` | `admin123` | 超级管理员（`*:*:*` 全部权限），部门 103 |
| `test` | `admin123` | 普通角色（角色 ID 2，岗位 4，部门 105） |

> 启动日志会打印生效开关，例如
> `system 初始化：登录认证=true 权限校验=false 图形验证码=false 数据权限=true 基础数据初始化=true`，
> `cron 初始化：定时任务调度=true 基础数据初始化=true`
> 部署后请核对，避免「配置未随部署更新」导致开关静默失效。

# 定时任务调度

调度器使用标准库实现（无第三方依赖，不引入 Quartz/gcron）：常驻协程按秒扫描 cron 表达式，
命中即在独立协程中执行「调用目标」，并把执行结果写入 `sys_job_log`。

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/monitor/job/list` `/create` `/update` `/delete` `/detail` | - | 任务增删改查（新增/修改会校验 cron 与调用目标） |
| `/monitor/job/change-status` | PUT | 启用（`status=0`）/ 暂停（`status=1`），立即生效 |
| `/monitor/job/run` | PUT | 立即执行一次（暂停中的任务同样可以执行） |
| `/monitor/jobLog/list` `/detail` `/delete` `/clear` | - | 任务执行日志查询 / 删除 / 清空 |

**cron 表达式**（6~7 段：`秒 分 时 日 月 周 [年]`，Quartz 语义）：

```bash
0/20 * * * * ?          # 每 20 秒执行一次
0 0 2 1 * ?             # 每月 1 日凌晨 2 点执行
0 15 10 ? * MON-FRI     # 周一到周五 10:15:00 执行
0 0 12 L * ?            # 每月最后一天 12:00:00
0 0 12 15W * ?          # 每月最接近 15 号的工作日
0 0 12 ? * 6L           # 每月最后一个周五
```

* 日（日）与周（周）都指定时取「或」关系（Quartz 语义），未使用的一侧写 `?`；
* 周字段 `1~7` 依次表示周日~周六（Quartz 约定），同时兼容 `0` 表示周日；
* 支持 `*` `?` `,` `-` `/` `L` `W` `#` 与月份/星期英文缩写（`JAN-DEC` / `SUN-SAT`）。

**调用目标**（`bean.method(args)`）：只能调用「已注册」的方法，
未注册的调用目标在新增/修改时即被拒绝（白名单校验，避免任意方法调用）。
内置示例目标定义在 `app/cron/task/task.go`：

```bash
demoTask.noParams()                              # 无参
demoTask.params('demo')                          # 单个字符串参数
demoTask.multipleParams('demo', true, 2000, 1.0) # 多类型参数
testTask.test1() / testTask.test2('hello')     # job_args 传参示例任务
```

新增任务目标：在 `app/cron/task/task.go` 中调用 `task.Register(bean, method, remark, fn)` 注册后即可使用；
任务表的 `job_args`（以 `|` 分隔）在调用目标本身不带参数时作为位置参数传入。

| 变量 | 默认值 | 说明 |
| ---- | ------ | ---- |
| `JOB_ENABLED` | true | 是否启动调度器；关闭后任务不再自动触发，手动「执行一次」仍可用 |
| `JOB_LOG_RETAIN_DAYS` | 30 | 任务日志保留天数（启动时清理，`<=0` 不清理） |

其他约定：

* `concurrent=1`（禁止并发）时，上次执行未结束会跳过本次触发；
* `misfire_policy`：`1` 立即执行 / `2` 执行一次 / `3` 放弃执行——调度器启动时据此补偿停机期间错过的执行（`3` 不补偿）；
* 删除任务时同时清理该任务的执行日志（日志按 `job_name + job_group` 匹配）；
* 内置示例任务默认「暂停」，`SEED_ENABLED=true` 且 `sys_job` 为空时写入 4 条示例任务。

# 数据权限

按「角色」划分可见数据范围（数据范围语义），当前作用于用户列表、部门列表/树、角色列表：

| data_scope | 含义 | 过滤条件（以用户/部门表为例） |
| ---------- | ---- | ---------------------------- |
| 1 | 全部数据权限 | 不追加条件 |
| 2 | 自定义数据权限 | `department_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id = ?)`（角色勾选的部门） |
| 3 | 本部门数据权限 | `department_id = 登录用户部门ID` |
| 4 | 本部门及以下数据权限 | `department_id IN (SELECT dept_id FROM sys_dept WHERE dept_id = ? OR FIND_IN_SET(?, ancestors))` |
| 5 | 仅本人数据权限 | `id = 登录用户ID`（部门表没有用户列，该条件恒不成立，即查不到数据） |

* 一个用户有多个角色时按 OR 合并（`(条件1 OR 条件2)`），任一角色为「全部数据权限」则不限制；
* 用户没有任何角色时不追加条件；超级管理员（用户 ID 1 或 `is_admin`）不受限制；
* 角色列表按「角色下的用户是否在可见范围内」过滤（子查询 `sys_user_role`）；
* 角色数据范围只允许 1~5：`0`（未传）按 `1` 处理，其它值返回「数据范围不正确」；
* 内部唯一性校验、子节点检查等场景以 `loginUser=nil` 调用，不受数据权限影响（保持原有行为）；
* 删除用户会同步清理 `sys_user_role`、`sys_user_post`，删除角色会同步清理 `sys_role_menu`、`sys_role_dept`、`sys_user_role`（避免孤儿关联数据）。

| 变量 | 默认值 | 说明 |
| ---- | ------ | ---- |
| `DATA_SCOPE_ENABLED` | true | 关闭后列表接口不再按数据范围过滤（排查「数据查不到」时可临时关闭） |
| `CACHE_ENABLED` | true | 参数/字典缓存开关（关闭后每次查询都回源数据库，仅建议排查问题时使用） |

# 服务监控 / 缓存监控

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/monitor/server` | GET | 服务监控：CPU（核心数/使用率/平均负载）、内存、Go 运行时、主机（名称/IP/系统/架构）、磁盘 |
| `/monitor/cache` | GET | 缓存监控：各内存缓存的键数量与读写统计（`info`/`dbSize`/`commandStats`/`names`） |
| `/monitor/cache/list` | GET | 缓存列表：分页查询缓存键值，支持 `cacheName`、`cacheKey` 过滤 |
| `/monitor/cache/clear` | DELETE | 清空缓存：`cacheName` 为空清空全部，指定名称不存在时返回错误 |

* 响应字段与前端监控页面约定保持一致（`cpuNum`/`memTotal`/`goVersion`/`sysFiles`/`dbSize`/`commandStats` 等），便于前端直接对接；
* **零新增依赖**：不使用 `gopsutil`，CPU/负载/内存/磁盘均读取 Linux `/proc`（`/proc/stat`、`/proc/meminfo`、`/proc/loadavg`、`/proc/mounts`）+ `syscall.Statfs` 实现；非 Linux 平台仅返回 Go 运行时与主机基础信息，系统指标为零值；
* 内存/磁盘单位均为 MB，使用率单位为 %；容器内根目录（`overlay`）与 `/dev/*` 设备都会作为磁盘列出；
* 缓存监控基于进程内内存缓存（`common/cache`，无 Redis）。除默认缓存外，图形验证码缓存（`captcha`）、参数缓存（`sys_config`）、字典缓存（`sys_dict`）也注册到监控中；缓存列表中的过期项不展示；
* 权限标识：`monitor:server:list`、`monitor:cache:list`、`monitor:cacheList:list`（已有种子菜单），认证开启权限校验时按 `/monitor/...` 路径推导。

# 参数 / 字典缓存

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/system/config/get-by-key` | GET | 按参数键名取值，缓存优先（未命中查库并写入 `sys_config` 缓存） |
| `/system/config/refresh-cache` | DELETE | 刷新参数缓存（清空后按数据库重新加载，返回缓存条目数） |
| `/system/dict-data/get-by-type` | GET | 按字典类型取字典数据，缓存优先（未命中查库并写入 `sys_dict` 缓存） |
| `/system/dict-type/refresh-cache` | DELETE | 刷新字典缓存（清空后按数据库重新加载，返回缓存的字典类型数） |

* **缓存语义**：`sys_config` 以「参数键名 -> 参数键值」缓存，`sys_dict` 以「字典类型 -> 字典数据列表」缓存；
* **变更即失效**：参数新增/修改/删除后失效对应键（键名变更时新旧键都会失效）；字典数据新增/修改/删除后失效对应字典类型；字典类型修改/删除后清理相关字典缓存；
* **启动预热**：服务启动时把参数表与字典数据全量加载进缓存（`CACHE_ENABLED=true`），并打印预热条目数；
* **排查手段**：`CACHE_ENABLED=false` 时全部查询直接回源数据库；缓存内容可在「缓存监控 / 缓存列表」页面查看（按 `sys_config` / `sys_dict` 名称过滤）。

# 代码生成

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/tool/gen/list` | GET | 代码生成表列表：支持 `table_name`、`table_comment`、`begin_time`、`end_time` 过滤 |
| `/tool/gen/db-list` | GET | 数据库表列表：`information_schema.tables` 中尚未导入的表（已排除 `qrtz_%`、`gen_%` 与已导入表） |
| `/tool/gen/import` | POST | 导入表结构：`tables` 支持 JSON 数组或逗号分隔，已导入的表自动跳过，全为已导入时返回错误 |
| `/tool/gen/detail` | GET | 代码生成表详情：表配置 + `columns`（字段配置）+ `pk_column`（主键列）+ 树表参数 |
| `/tool/gen/column-list` | GET | 字段配置列表：按 `table_id` 查询 |
| `/tool/gen/update` | PUT | 修改生成配置：表配置与字段配置（`columns` 可省略），树表参数写入 `options` |
| `/tool/gen/synch-db` | POST | 同步数据库表结构：新增列、删除已移除列、列类型变化时重置列配置 |
| `/tool/gen/delete` | DELETE | 删除生成配置：同一事务内清理 `gen_table_column` |
| `/tool/gen/preview` | GET | 预览生成代码：返回「文件相对路径 -> 文件内容」映射（domain/service/controller/接口测试/建表 SQL/接入说明） |
| `/tool/gen/download` | GET | 下载生成代码：`application/zip` 附件，文件名取自表名去前缀后的蛇形名（如 `notice.zip`） |

- **渲染约定**：`app/gen/service/render.go` + `app/gen/service/templates/*.tmpl`（`embed.FS` 内置，**只用标准库 `text/template`**）；
  生成的 Go 代码经 `go/format` 格式化，语法错误会在预览/下载时直接报错；
- **命名来源**：类名/功能名/作者优先取 `gen_table` 中按表配置的值（页面可编辑），缺失时回落到 `system.gen.*` 与表名推导；
  文件名固定为「表名去前缀后的蛇形名」，与「一表一套文件」约定一致；
- **主键要求**：仅支持整型主键（`bigint`/`int`），主键实体字段统一为 `ID uint`，否则生成时报错；
- **生成内容**：`create_by` 与 `update_by` 同时存在时实体嵌入 `core.AuditModel`，并在 service 中填充审计列。

* **表结构来源**：字段名称/类型/主键/自增/必填全部来自 `information_schema.columns`（按 `ordinal_position` 排序），表描述来自 `information_schema.tables`，不依赖任何第三方元数据表；
* **默认配置推导**：字符串 -> `string`（`text` 系列或长度 >= 500 为 `textarea`，否则 `input`）、时间 -> `Time`/`datetime`、数字 -> `int`/`int64`/`float64`；列名含 `name`/`status` 默认必填，`status` 列默认 `radio`，`type`/`sex` 结尾的列默认 `select`，`*name` 列默认 `LIKE` 查询；`id`/`create_by`/`create_time`/`del_flag`/`update_by`/`update_time` 默认不参与编辑、列表（`remark` 不参与查询）；
* **类名与包名来自配置**：`system.gen.auto-remove-pre` + `system.gen.table-prefix` 决定实体类名（`sys_notice` -> `Notice`），`system.gen.package-name`/`module-name`/`author` 决定包路径、模块名与作者；
* **主键列**：优先取 `is_pk=1` 的列，表无主键时取第一列（避免生成代码时主键为空）；
* **删除/同步均在事务内完成**，同步不会丢失用户已配置的显示类型（列类型未变化时只同步列名、描述与排序列）；
* `gen_table` 只能通过导入接口写入，因此没有独立的 `create` 接口（避免出现与实际表结构不一致的脏配置）。

| 变量 | 默认值 | 说明 |
| ---- | ------ | ---- |
| `GEN_PACKAGE_NAME` | github.com/bluefroog/admintpl | 生成代码的包路径（Go module 前缀） |
| `GEN_MODULE_NAME` | system | 生成代码所属模块（`app/<module>`） |
| `GEN_AUTHOR` | bluefrog | 生成功能作者 |
| `GEN_AUTO_REMOVE_PRE` | true | 表名生成实体类名时是否去掉表前缀 |
| `GEN_TABLE_PREFIX` | sys_ | 表前缀，多个以英文逗号分隔 |

# 模块划分

```
app/system    系统管理（用户/角色/部门/菜单/岗位/字典/参数/公告/日志/认证）
app/monitor   系统监控（在线用户/登录日志/操作日志/服务监控/缓存监控）
app/cron      定时任务（任务配置/调度器/任务日志/内置调用目标）
app/gen       代码生成（表结构导入/生成配置/同步数据库表结构/模板渲染与下载）
common/database  数据库初始化与迁移（DB/InitDB/AutoMigrateAll/TableEmpty）
common/sysinfo   服务器指标采集（标准库 /proc 实现）
common/cache     进程内内存缓存（含监控统计与注册表）
common/secret    配置敏感值加解密（crypto-go AES/GCM，ENC(...) 密文）
cmd/encrypt      配置值加密/解密命令行（just encrypt / decrypt 调用）
bootstrap     启动编排：初始化数据库 → 迁移各模块模型 → 依次注册各模块路由
```

* 每个模块自带 `domain`（实体/请求结构体）、`service`（业务逻辑）、`controller`（HTTP 控制器）、`initialize.go`（初始化入口）；
* 新增监控/定时任务类业务时，在对应模块内新增文件即可，`bootstrap` 无需改动（模型通过各模块 `domain.Models()` 汇总）；
* 接口测试文件放在各模块的 `test/http/` 下（如 `app/monitor/test/http/cache.http`）。

# 配置与环境切换

配置项统一放在根目录 `.env`，`config/config.yaml` 只保留结构并通过 `${KEY}` / `${KEY:-默认值}` 引用：

```bash
# 切换到开发环境（.env.dev  -> .env）
just env dev

# 切换到生产环境（.env.pro -> .env）
just env pro
```

- 优先级：**系统环境变量 > `.env` 文件 > `config.yaml` 默认值**
- `.env` 不入库；环境模板 `.env.dev`、`.env.pro` 入库，按需增删变量
- 可用 `ENV_FILE` 指定其它 env 文件，用 `CONFIG_NAME` 指定其它配置文件

# 配置加密（明文 / AES-256-GCM / SM4 密文）

敏感配置（数据库密码、令牌密钥等）支持两种写法，由开关切换，**解密在配置加载阶段统一完成**，
因此数据库等所有中间件拿到的都是明文，无需各自处理解密：

| 开关 | 值的要求 | 说明 |
| ---- | -------- | ---- |
| `CONFIG_ENCRYPT_ENABLED=false`（默认） | 明文 | 与历史配置完全兼容；若检测到 `ENC(...)` 密文会**启动失败**并提示开启开关 |
| `CONFIG_ENCRYPT_ENABLED=true` | 密文或明文 | 自动解密所有含 `ENC(...)` 片段的值；密钥缺失或解密失败直接启动失败（快速失败，不会拿密文连库） |

算法由 `CONFIG_SECRET_ALGORITHM` 配置（只影响**新生成**的密文）：

| 算法 | 说明 | 密文形态 |
| ---- | ---- | -------- |
| `AES`（默认） | AES-256-GCM（AEAD，带完整性校验） | `ENC(AES:xxxx)` |
| `SM4` | 国密 SM4-CBC（分组模式，PKCS7 填充，无 AEAD） | `ENC(SM4:xxxx)` |

密钥 `CONFIG_SECRET_KEY`：任意字符串（内部 SHA-256 派生，AES 取 32 字节、SM4 取 16 字节），
通过 `.env` / 容器环境变量注入；更换密钥需要重新加密已有密文。

> **算法随密文走**：密文自带算法标签（`ENC(AES:...)` / `ENC(SM4:...)`），解密时按标签选算法，
> 因此切换 `CONFIG_SECRET_ALGORITHM` 不会导致历史密文失效（可逐步迁移）；
> 无标签的旧密文 `ENC(xxxx)` 一律按 AES 处理，保持向后兼容。

```bash
# 1. 先生成密钥并写入 .env（只需一次）
echo 'CONFIG_SECRET_KEY=my-strong-passphrase' >> .env

# 2. 加密敏感值（默认 AES），输出形如 ENC(AES:xxxxx)
just encrypt 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'

# 2'. 指定算法改用国密 SM4（也可在 .env 里设 CONFIG_SECRET_ALGORITHM=SM4 作为默认算法）
just encrypt 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local' SM4

# 2''. 或者直接输出可粘贴到 .env 的配置片段（含开关、算法、密钥、加密后的 DB_DSN）
just encrypt-env 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'

# 3. 把密文写进 .env，并打开开关
#    CONFIG_ENCRYPT_ENABLED=true
#    CONFIG_SECRET_ALGORITHM=AES
#    DB_DSN=ENC(AES:xxxxx)

# 4. 排查用：解密回读（算法由密文标签自动识别，无需指定）
just decrypt 'ENC(AES:xxxxx)'
```

- 支持**整体加密**（`DB_DSN=ENC(...)`）与**局部加密**（`DB_DSN=root:ENC(...)@tcp(127.0.0.1:3306)/admintpl?...`），
  局部加密可以只保护密码，主机、库名保持可读；
- 明文与密文可以混用，便于逐个变量迁移；
- 加解密统一复用 `github.com/freewu/crypto-go` 的封装：AES 用 `aes.GetInstance("GCM")`（nonce 由该封装拼在密文头部），
  SM4 用 `sm4.GetInstance("CBC")`（随机 IV 拼在密文头部，PKCS7 填充由 gmsm 完成）；
  密钥固定 SHA-256 派生（AES 32 字节 / SM4 16 字节），规避「密钥长度必须为 16/24/32 字节」的限制；
- 密文被篡改或密钥不匹配都会解密失败，绝不静默返回脏数据：AES-GCM 由认证标签保证；
  SM4-CBC 本身没有认证能力，因此在密文载荷内附一段截断的 SHA-256 校验和（16 字节），
  篡改 IV、密文或换错密钥都会校验失败；
  合规场景需要国密时选 `SM4`，安全优先仍建议用默认的 `AES`（AEAD，完整性保证更强）；
- 密文使用 base64 URL 安全字母表，不含 `+ / =` 与括号，可安全嵌入 DSN / URL；
- 用户登录口令与配置加密是两件事：用户口令以 bcrypt 落库，配置敏感值以 AES/SM4 密文落配置。

# 项目开发 / 测试 / 打包 / 发布（使用 just）
```bash
# 查看全部命令
just --list

# 初始化：检查项目依赖（有则绿勾 ✔，缺失自动安装）
just init

# 环境切换：dev / pro
just env dev

# 配置加密：加密敏感值 / 输出 .env 片段 / 解密排查
just encrypt 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'
just encrypt-env 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'
just decrypt 'ENC(xxxxx)'

# 开发：启动项目（默认监听 :8080）
just dev

# 测试：运行全部单元测试
just test

# 检查：go vet
just vet

# 冒烟：对已启动的实例跑认证 + 读接口 + 全模块 CRUD 回归（自动登录携带令牌）
just smoke
just smoke http://192.168.122.129:8080
just gen-check http://192.168.122.129:8080   # 代码生成深度校验（需 mysql 客户端）
SMOKE_USER=admin SMOKE_PASS=admin123 just smoke    # 自定义登录账号

# 打包：编译 linux/amd64 静态二进制（镜像内运行）
just build

# 发布：构建 Docker 镜像并推送远程仓库
just release

# 容器运行：本地以容器方式启动（用 .env 注入配置）
just docker-run
```

# 容器部署

镜像只包含二进制与 `config/`（配置结构），配置值通过环境变量注入，优先级：
**容器环境变量 > `.env` > `config/config.yaml` 默认值**。

```bash
just build
just env pro          # 选好生产配置
just release          # 构建并推送镜像

# 方式一：指定 env 文件
docker run -d --name admintpl -p 8080:8080 --env-file .env admintpl-server:latest

# 方式二：直接注入环境变量
docker run -d --name admintpl -p 8080:8080 \
  -e DB_DSN='root:123456@tcp(10.0.0.5:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local' \
  -e SERVICE_DEBUG=false \
  admintpl-server:latest
```

> 注：`just build` 编译的是 **linux/amd64 静态二进制**（供镜像内运行）；本地调试请用 `just dev`（`go run`）。
>
> 注意：`config/` 目录需与代码同步更新（新增开关若未同步，配置会静默失效）。可用
> `scp -r config bluefrog@<服务器>:~/yjbuild/` 覆盖，或先删除服务器上的旧 `config/` 再拷贝。

# 内网 / 离线部署（服务器无外网、无 Go 环境）

在开发机（有 Go + Docker）完成构建，导出镜像文件后拷贝到服务器加载：

```bash
# 开发机：构建并导出镜像
just env pro          # 选好生产配置（生成 .env）
just build            # 编译 linux/amd64 静态二进制
just release          # 构建并推送镜像；仅本地构建可改用 docker build . -t admintpl-server:latest
docker save admintpl-server:latest | gzip > admintpl-server.tar.gz
scp admintpl-server.tar.gz bluefrog@192.168.122.129:~

# 服务器：加载并启动（.env 由手工编写或从开发机拷贝）
gunzip -c ~/admintpl-server.tar.gz | docker load
docker run -d --name admintpl --restart always -p 8080:8080 \
  --env-file ~/.env admintpl-server:latest
docker logs -f admintpl          # 正常应看到 GIN-debug 路由注册日志

# 部署后自检（读接口 + 全模块 CRUD 往返，会自建自删测试数据）
just smoke http://192.168.122.129:8080
```

# 接口文档生成
```bash
# 生成文档
just docs   # 等价于 swag init -o ./app/swagger

# 访问
http://127.0.0.1:8080/swagger.json
```

# 使用组件
```
github.com/gin-gonic/gin 路由分发
gorm.io/gorm db操作
github.com/sirupsen/logrus 日志
github.com/spf13/viper 配置文件
github.com/go-playground/validator/v10 结构体参数验证
github.com/swaggo/swag 接口文档
golang.org/x/crypto bcrypt 用户口令
github.com/freewu/crypto-go 配置敏感值加解密（AES/GCM 封装）
```

# swag 安装
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```
