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
* [已实现]参数管理：对系统动态配置常用参数。
* [已实现]操作日志：系统正常操作日志记录和查询；系统异常信息日志记录和查询。
* [已实现]登录日志：系统登录日志记录查询包含登录异常。
* [已实现]在线用户：当前系统中活跃用户状态监控，支持强退。
* [待实现]定时任务：任务配置已实现（增删改查/启停/日志），调度执行未实现。
* [待实现]代码生成：前后端代码的生成。
* [待实现]服务监控：监视当前系统CPU、内存、磁盘、堆栈等相关信息。

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
| `/system/user-online/*` | - | 在线用户列表 / 详情 / 强退（强退后令牌立即失效） |

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
| `AUTH_ENABLED` | true | 关闭后所有 `/system` 接口免登录（仅用于本地调试） |
| `AUTH_TOKEN_EXPIRE` | 720 | 令牌有效期（分钟） |
| `AUTH_MAX_RETRY` | 5 | 连续登录失败次数上限，超过后锁定账号 |
| `AUTH_LOCK_TIME` | 10 | 登录失败锁定时间（分钟） |
| `AUTH_PERMISSION_ENABLED` | false | 开启后按请求路径推导权限标识（如 `system/user/list` -> `system:user:list`）做角色权限校验 |
| `CAPTCHA_ENABLED` | dev=false / pro=true | 是否开启登录图形验证码 |
| `CAPTCHA_LENGTH` / `CAPTCHA_EXPIRE` | 4 / 120 | 验证码位数 / 有效期（秒） |
| `SEED_ENABLED` | dev=true / pro=false | 空库时自动初始化基础数据（部门/岗位/角色/用户/菜单/字典/参数） |

基础数据默认账号（`SEED_ENABLED=true` 且对应表为空时写入，幂等）：

| 账号 | 密码 | 说明 |
| ---- | ---- | ---- |
| `admin` | `admin123` | 超级管理员（`*:*:*` 全部权限），部门 103 |
| `bluefrog` | `admin123` | 普通角色（角色 ID 2，仅部分菜单） |

> 启动日志会打印生效开关，例如
> `system 初始化：登录认证=true 权限校验=false 图形验证码=false 基础数据初始化=true`，
> 部署后请核对，避免「配置未随部署更新」导致开关静默失效。

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
