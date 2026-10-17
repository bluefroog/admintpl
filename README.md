# 项目说明
> 使用 gin + gorm 实现一个后台管理系统快速开发框架

# 内置功能

* [待实现]用户管理：用户是系统操作者，该功能主要完成系统用户配置。
* [待实现]部门管理：配置系统组织机构（公司、部门、小组），树结构展现支持数据权限。
* [待实现]岗位管理：配置系统用户所属担任职务。
* [待实现]菜单管理：配置系统菜单，操作权限，按钮权限标识等。
* [待实现]角色管理：角色菜单权限分配、设置角色按机构进行数据范围权限划分。
* [待实现]字典管理：对系统中经常使用的一些较为固定的数据进行维护。
* [待实现]参数管理：对系统动态配置常用参数。
* [待实现]操作日志：系统正常操作日志记录和查询；系统异常信息日志记录和查询。
* [待实现]登录日志：系统登录日志记录查询包含登录异常。
* [待实现]在线用户：当前系统中活跃用户状态监控。
* [待实现]定时任务：在线（添加、修改、删除)任务调度包含执行结果日志。
* [待实现]代码生成：前后端代码的生成。
* [待实现]服务监控：监视当前系统CPU、内存、磁盘、堆栈等相关信息。

# 项目初始化
```bash
# 初始化（检查依赖 + 下载依赖）
just init
```

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

# 打包：编译静态二进制
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
