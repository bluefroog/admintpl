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
* [已实现]字典/参数缓存：字典数据按类型缓存、参数按键名缓存，变更即失效，支持一键刷新缓存（与若依缓存语义一致）。
* [已实现]操作日志：系统正常操作日志记录和查询；系统异常信息日志记录和查询。
* [已实现]登录日志：系统登录日志记录查询包含登录异常。
* [已实现]在线用户：当前系统中活跃用户状态监控，支持强退。
* [已实现]缓存监控：查询内存缓存的键值、读写统计，支持按缓存名称/缓存键过滤与清空。
* [已实现]服务监控：监视当前系统 CPU、内存、磁盘、Go 运行时等信息。
* [已实现]数据权限：按角色划分数据范围（全部/自定义/本部门/本部门及以下/仅本人），作用于用户、部门、角色列表。
* [已实现]定时任务：任务配置（增删改查/启停/立即执行一次）+ 调度执行 + 任务日志；cron 表达式与调用目标均为若依（Quartz）语义。
* [待实现]代码生成：前后端代码的生成。

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
| `/getRouters` | GET | 获取当前用户的前端路由（Layout/ParentView、外链等若依约定结构） |
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
| `ry` | `admin123` | 普通角色（角色 ID 2，岗位 4，部门 105） |

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

**cron 表达式**（6~7 段：`秒 分 时 日 月 周 [年]`，与若依 Quartz 一致）：

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

**调用目标**（`bean.method(args)`，与若依一致）：只能调用「已注册」的方法，
未注册的调用目标在新增/修改时即被拒绝（等价于若依的调用目标白名单校验，避免任意方法调用）。
内置示例目标定义在 `app/cron/task/task.go`：

```bash
ryTask.ryNoParams()                            # 无参
ryTask.ryParams('ry')                          # 单个字符串参数
ryTask.ryMultipleParams('ry', true, 2000, 1.0) # 多类型参数
testTask.test1() / testTask.test2('hello')     # 迁移自 yjgo 的示例任务
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
* 删除任务时同时清理该任务的执行日志（与若依一致，日志按 `job_name + job_group` 匹配）；
* 内置示例任务默认「暂停」，`SEED_ENABLED=true` 且 `sys_job` 为空时写入 4 条示例任务。

# 数据权限

按「角色」划分可见数据范围（与若依 `@DataScope` 语义一致），当前作用于用户列表、部门列表/树、角色列表：

| data_scope | 含义 | 过滤条件（以用户/部门表为例） |
| ---------- | ---- | ---------------------------- |
| 1 | 全部数据权限 | 不追加条件 |
| 2 | 自定义数据权限 | `department_id IN (SELECT dept_id FROM sys_role_dept WHERE role_id = ?)`（角色勾选的部门） |
| 3 | 本部门数据权限 | `department_id = 登录用户部门ID` |
| 4 | 本部门及以下数据权限 | `department_id IN (SELECT dept_id FROM sys_dept WHERE dept_id = ? OR FIND_IN_SET(?, ancestors))` |
| 5 | 仅本人数据权限 | `id = 登录用户ID`（部门表没有用户列，该条件恒不成立，即查不到数据） |

* 一个用户有多个角色时按 OR 合并（`(条件1 OR 条件2)`），任一角色为「全部数据权限」则不限制；
* 用户没有任何角色时不追加条件（与若依一致）；超级管理员（用户 ID 1 或 `is_admin`）不受限制；
* 角色列表按「角色下的用户是否在可见范围内」过滤（子查询 `sys_user_role`），与若依一致；
* 角色数据范围只允许 1~5：`0`（未传）按 `1` 处理，其它值返回「数据范围不正确」；
* 内部唯一性校验、子节点检查等场景以 `loginUser=nil` 调用，不受数据权限影响（保持原有行为）；
* 删除用户会同步清理 `sys_user_role`、`sys_user_post`，删除角色会同步清理 `sys_role_menu`、`sys_role_dept`、`sys_user_role`（与若依一致，避免孤儿关联数据）。

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

* 响应字段与若依前端（`ruoyi-ui` 的监控页面）保持一致（`cpuNum`/`memTotal`/`goVersion`/`sysFiles`/`dbSize`/`commandStats` 等），便于直接复用前端页面；
* **零新增依赖**：不使用 `gopsutil`，CPU/负载/内存/磁盘均读取 Linux `/proc`（`/proc/stat`、`/proc/meminfo`、`/proc/loadavg`、`/proc/mounts`）+ `syscall.Statfs` 实现；非 Linux 平台仅返回 Go 运行时与主机基础信息，系统指标为零值；
* 内存/磁盘单位均为 MB，使用率单位为 %；容器内根目录（`overlay`）与 `/dev/*` 设备都会作为磁盘列出；
* 缓存监控基于进程内内存缓存（`common/cache`，无 Redis）。除默认缓存外，图形验证码缓存（`captcha`）、参数缓存（`sys_config`）、字典缓存（`sys_dict`）也注册到监控中；缓存列表中的过期项不展示；
* 权限标识：`monitor:server:list`、`monitor:cache:list`、`monitor:cacheList:list`（已有种子菜单），认证开启权限校验时按 `/monitor/...` 路径推导。

# 参数 / 字典缓存（若依缓存语义）

| 接口 | 方法 | 说明 |
| ---- | ---- | ---- |
| `/system/config/get-by-key` | GET | 按参数键名取值，缓存优先（未命中查库并写入 `sys_config` 缓存） |
| `/system/config/refresh-cache` | DELETE | 刷新参数缓存（清空后按数据库重新加载，返回缓存条目数） |
| `/system/dict-data/get-by-type` | GET | 按字典类型取字典数据，缓存优先（未命中查库并写入 `sys_dict` 缓存） |
| `/system/dict-type/refresh-cache` | DELETE | 刷新字典缓存（清空后按数据库重新加载，返回缓存的字典类型数） |

* **缓存语义与若依一致**：`sys_config` 以「参数键名 -> 参数键值」缓存，`sys_dict` 以「字典类型 -> 字典数据列表」缓存；
* **变更即失效**：参数新增/修改/删除后失效对应键（键名变更时新旧键都会失效）；字典数据新增/修改/删除后失效对应字典类型；字典类型修改/删除后清理相关字典缓存；
* **启动预热**：服务启动时把参数表与字典数据全量加载进缓存（`CACHE_ENABLED=true`），并打印预热条目数；
* **排查手段**：`CACHE_ENABLED=false` 时全部查询直接回源数据库；缓存内容可在「缓存监控 / 缓存列表」页面查看（按 `sys_config` / `sys_dict` 名称过滤）。

# 模块划分

```
app/system    系统管理（用户/角色/部门/菜单/岗位/字典/参数/公告/日志/认证）
app/monitor   系统监控（在线用户/登录日志/操作日志/服务监控/缓存监控）
app/cron      定时任务（任务配置/调度器/任务日志/内置调用目标）
common/database  数据库初始化与迁移（DB/InitDB/AutoMigrateAll/TableEmpty）
common/sysinfo   服务器指标采集（标准库 /proc 实现）
common/cache     进程内内存缓存（含监控统计与注册表）
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

# 项目开发 / 测试 / 打包 / 发布（使用 just）
```bash
# 查看全部命令
just --list

# 初始化：检查项目依赖（有则绿勾 ✔，缺失自动安装）
just init

# 环境切换：dev / pro
just env dev

# 开发：启动项目（默认监听 :8080）
just dev

# 测试：运行全部单元测试
just test

# 检查：go vet
just vet

# 冒烟：对已启动的实例跑认证 + 读接口 + 全模块 CRUD 回归（自动登录携带令牌）
just smoke
just smoke http://192.168.122.129:8080
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
docker run -d --name admintpl -p 8080:8080 --env-file .env rouyi-go-service:latest

# 方式二：直接注入环境变量
docker run -d --name admintpl -p 8080:8080 \
  -e DB_DSN='root:123456@tcp(10.0.0.5:3306)/ruoyi?charset=utf8mb4&parseTime=True&loc=Local' \
  -e SERVICE_DEBUG=false \
  rouyi-go-service:latest
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
just release          # 构建并推送镜像；仅本地构建可改用 docker build . -t rouyi-go-service:latest
docker save rouyi-go-service:latest | gzip > rouyi-go-service.tar.gz
scp rouyi-go-service.tar.gz bluefrog@192.168.122.129:~

# 服务器：加载并启动（.env 由手工编写或从开发机拷贝）
gunzip -c ~/rouyi-go-service.tar.gz | docker load
docker run -d --name admintpl --restart always -p 8080:8080 \
  --env-file ~/.env rouyi-go-service:latest
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
```

# swag 安装
```bash
go install github.com/swaggo/swag/cmd/swag@latest
```
