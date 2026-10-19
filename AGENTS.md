# AGENTS.md — 项目开发约定（Agent 必须遵守）

> 项目：Go + Gin + GORM 后台管理系统快速开发框架（模块 `github.com/bluefroog/admintpl`）
> 详细规范见 `docs/开发规范.md`，本文档为 Agent 开发时必须遵守的核心约定。

## 1. 代码放置规范

- 每张表对应一套文件，文件名 = 表名（蛇形），放置位置：
  - `app/system/controller/<table>.go` — HTTP 控制器（只做：绑定参数 → `validate.Struct` 校验 → 调 service → 返回响应）
  - `app/system/service/<table>.go` — 业务逻辑（用 `domain.DB`，标准方法：GetCount/GetList/parseFilter/Create/Update/Delete/Detail + 唯一性校验）
  - `app/system/domain/<table>.go` — 实体（`ID uint` + `XxxBase` 嵌入 `core.RuoyiModel` + `TableName()`）与请求结构体（`XxxSearchRequest` 嵌入 `request.PageInfo/SortInfo/core.RouyiSearchRequest`、`XxxCreateRequest`、`XxxUpdateRequest` 嵌入 `request.GetById`）
  - `app/system/test/http/<table>.http` — 接口测试文件（`###` 分隔用例）
- 新增表模型必须注册到 `app/system/domain/init.go` 的 `autoMigrate`；路由必须注册到 `app/system/controller/init.go`。
- 路由统一 `/system/<表名>/<list|create|update|delete|detail>`：
  - 业务表：全量 5 个接口；日志表：仅 list/delete/detail；关系表（复合主键）：仅 list/create/delete。
  - 需登录的接口必须注册到 `app/system/controller/init.go` 的 `authorized` 分组内（已挂 `middleware.Auth()`）；
    免登录接口（`/login`、`/captchaImage`）注册在该分组之前。
  - 扩展动作（`tree`、`check-*`、`roles`、`menu`、`data-scope`、`clear`、`force-logout` 等）见 `docs/开发规范.md` 1.3 节。
- 响应：成功 `response.OK(data, c)` / `response.Success("提示", c)`；失败 `response.Fail(-1, "错误", c)`。
- 树形表（部门、菜单）删除前必须检查子节点，存在子节点禁止删除。
- 删除接口必须同步清理关联表（用户：`sys_user_role`/`sys_user_post`；角色：`sys_role_menu`/`sys_role_dept`/`sys_user_role`），与若依一致，避免孤儿数据。
- 新增 `.go`/`.md`/`.http` 文件使用 **CRLF** 行尾。

## 2. 开发工具：just（开发 / 测试 / 打包 / 发布）

- 统一使用 `just`（命令运行器，项目根目录 `justfile`）管理开发、测试、打包、发布。
- 常用命令：
  - `just init` — 初始化：检查项目依赖（有则绿勾 ✔，缺失自动安装）+ `go mod tidy`
  - `just dev` — 开发：启动项目（`go run main.go`）
  - `just test` — 测试：`go test ./...`
  - `just vet` — 检查：`go vet ./...`
  - `just smoke [BASE_URL]` — 冒烟：对已启动实例跑读接口 + 全模块 CRUD 回归（默认 `http://127.0.0.1:8080`，脚本 `scripts/smoke.sh`）
  - `just env dev` / `just env pro` — 环境切换：将 `.env.dev` / `.env.pro` 复制为 `.env`
  - `just build` — 打包：编译供镜像使用的 linux/amd64 静态二进制 `rouyi-go-service`
  - `just release` — 发布：构建 Docker 镜像并推送远程仓库
  - `just docs` — 生成 swagger 文档
  - `just` / `just --list` — 查看全部命令
- 开发前先 `just vet` + `just test`，开发完成打包用 `just build`，发布用 `just release`。

### 配置约定（环境变量优先）

- `config/config.yaml` **只保留配置结构**，具体值写在根目录 `.env`；yaml 中用 `${KEY}` / `${KEY:-默认值}` 引用。
- 优先级：**系统环境变量 > `.env` > `config.yaml` 默认值**；`.env` 不入库，环境模板 `.env.dev`、`.env.pro` 入库。
- 新增配置项：先在 `config/config.yaml` 加占位符，再在 `.env.dev` / `.env.pro` 补同名变量，不要在 yaml 里写死具体值。

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
- **权限标识**：路径 `/system/<模块>/<动作>` 对应权限 `system:<模块>:<动作>`（如 `/system/user/list` -> `system:user:list`），
  与菜单 `perms` 字段一致；`system/user/profile|password|avatar` 为登录用户自助接口，不参与权限校验。
- **配置**：新增开关必须同时改 `config/config.yaml`（`${KEY:-默认值}` 占位符）与 `.env.dev` / `.env.pro`；
  `config/config.yaml` 末尾维护变量清单，`common/config/config_test.go` 会校验代码读取的键在配置文件中存在，缺失即测试失败。
- **GORM 写入零值陷阱**：带 `default:` 标签的列在「结构体方式 Create」时，若字段取零值会被数据库默认值覆盖
  （如 `sys_menu.is_frame` 默认 1、`sys_user.status` 默认 1、`sys_role.menu_check_strictly` 默认 1）。
  此类表的 Create 必须使用 `map[string]interface{}` 显式写入（参考 `service/menu.go`、`service/user.go`、`service/role.go`），
  Update 已统一使用 map。
- **基础数据初始化**：`app/system/service/seed.go` 仅在对应表为空时写入（幂等、事务内），入口开关 `SEED_ENABLED`；
  修改种子数据后需清空对应表并重启服务验证，或直接改库核对。
- **自检**：开发完成后本地跑 `just vet` + `just test`，对已启动实例跑 `just smoke <BASE_URL>`
  （脚本会自动登录携带令牌，并校验零值字段落库、SQL 列名错配、定时任务调度等回归项）。

## 6. 数据权限约定

- **实现位置**：`app/system/service/data-scope.go`；查询入口统一为 `DataScope{}.Apply(db, loginUser, DataScopeOptions{...})`，
  角色列表用 `DataScope{}.ApplyToRoles(db, loginUser)`；列表方法一律接收 `loginUser *domain.LoginUser` 参数。
- **语义与若依 `@DataScope` 一致**：`1` 全部（不过滤）、`2` 自定义（`sys_role_dept`）、`3` 本部门、`4` 本部门及以下（`ancestors` 递归）、`5` 仅本人；
  多角色按 OR 合并，任一角色为 `1` 则不限制；无角色 / 超级管理员 / 未登录（`loginUser=nil`）不加条件。
- **取值校验**：角色新增、修改、数据范围保存统一走 `NormalizeDataScope`（`0` 归一为 `1`，`1~5` 合法，其它值报「数据范围不正确」）。
- **开关**：`system.data-scope.enabled`（`DATA_SCOPE_ENABLED`，默认 true）；关闭后列表不再过滤，便于排查「数据查不到」。
- **新增列表接口必须接数据权限**：带部门/用户归属的业务列表在拼接查询后调用 `DataScope{}.Apply(...)`，
  并同步给控制器方法传 `middleware.GetLoginUser(c)`；内部唯一性校验、子节点检查等场景传 `nil` 保持原有行为。
- **约定列名**：`DataScopeOptions.DeptColumn` 用表内真实列名（用户表 `department_id`、部门表 `dept_id`），
  `UserColumn` 仅在支持「仅本人」的表上传（部门表无用户列，传空则 `5` 恒不成立）。

## 7. 定时任务约定

- **cron 表达式**：统一使用 `common/cron`（若依/Quartz 语义，6~7 段 `秒 分 时 日 月 周 [年]`），
  禁止再引入 Quartz/gcron/cron 第三方库；日与周都指定时取「或」关系，周字段 `1~7` 为周日~周六（兼容 `0` 表示周日）。
- **调用目标**：`bean.method(args)` 形式，只能调用在 `app/system/task/task.go` 中通过 `task.Register(...)` 显式注册的方法；
  新增任务目标 = 注册目标方法 + 建任务并启用，未注册的目标会在新增/修改时被拒绝（等价于若依的调用目标白名单）。
- **调度器**：`app/system/service/job-scheduler.go`（按表命名规范之外的模块级文件），按秒扫描、独立协程执行、
  并发策略与错过执行补偿按 `sys_job.concurrent` / `sys_job.misfire_policy` 生效，执行结果写入 `sys_job_log`；
  开关 `system.job.enabled`（`JOB_ENABLED`）。
- **任务变更必须联动调度器**：新增/修改/启停后调用 `JobScheduler{}.Refresh(id)`，删除调用 `Remove(id)`
  （`service/job.go` 已内置）；新任务表的 Create 同样使用 map 写入（`concurrent`/`status`/`misfire_policy` 有默认值）。
- **任务日志写入**：使用 map 写入（`status=0`、`exception_info=''` 是零值）；日志清理天数由 `JOB_LOG_RETAIN_DAYS` 控制。
