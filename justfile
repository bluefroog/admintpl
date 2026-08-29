# ─────────────────────────────────────────────────────────────
# just 命令集：开发 / 测试 / 打包 / 发布
# 用法：just <recipe>，例如 `just dev`、`just build`
# 查看全部：just --list
# ─────────────────────────────────────────────────────────────

# 变量
srv := "rouyi-go-service"
version := "latest"
image := srv + ":" + version

# 默认：列出所有可用命令
default:
    @just --list

# 开发：启动项目（默认监听 :8080）
dev:
    go run main.go

# 测试：运行全部单元测试
test:
    go test ./...

# 检查：静态检查（go vet）
vet:
    go vet ./...

# 打包：编译静态二进制文件
build:
    rm -f ./{{srv}}
    go env -w CGO_ENABLED=0
    go build -a -ldflags '-extldflags "-static"' -o {{srv}} main.go
    go env -w CGO_ENABLED=1

# 发布：构建 Docker 镜像并推送到远程仓库
release:
    docker login
    docker build . -t {{image}}
    docker push {{image}}
    rm -f ./{{srv}}

# 文档：生成 swagger 接口文档
docs:
    swag init -o ./app/swagger

# 提交：暂存所有变更并提交
# 用法：just commit "feat(system): 新增xxx模块"
# 不带参数则打开编辑器（自动载入 .gitmessage 模板）
commit message="":
    git add -A
    @if [ -n "{{message}}" ]; then git commit -m "{{message}}"; else git commit; fi

# 推送：推送到远程仓库
push:
    git push

# 提交并推送：开发完成一键 commit + push（推送成功才算开发完成）
ship:
    git add -A
    git commit
    git push
