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
- 响应：成功 `response.OK(data, c)` / `response.Success("提示", c)`；失败 `response.Fail(-1, "错误", c)`。
- 树形表（部门、菜单）删除前必须检查子节点，存在子节点禁止删除。
- 新增 `.go`/`.md`/`.http` 文件使用 **CRLF** 行尾。

## 2. 开发工具：just（开发 / 测试 / 打包 / 发布）

- 统一使用 `just`（命令运行器，项目根目录 `justfile`）管理开发、测试、打包、发布。
- 常用命令：
  - `just init` — 初始化：检查项目依赖（有则绿勾 ✔，缺失自动安装）+ `go mod tidy`
  - `just dev` — 开发：启动项目（`go run main.go`）
  - `just test` — 测试：`go test ./...`
  - `just vet` — 检查：`go vet ./...`
  - `just build` — 打包：编译静态二进制 `rouyi-go-service`
  - `just release` — 发布：构建 Docker 镜像并推送远程仓库
  - `just docs` — 生成 swagger 文档
  - `just` / `just --list` — 查看全部命令
- 开发前先 `just vet` + `just test`，开发完成打包用 `just build`，发布用 `just release`。

## 3. 开发完成流程（每次开发完成后必须执行）

**每次开发完成后，必须自动执行 git commit 并 git push 到远程仓库，推送成功后才算本次开发完成。**

1. 先验证：`just vet` 与 `just test` 必须无错误。
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
