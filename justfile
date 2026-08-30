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

# 初始化：检查项目依赖（有则绿勾 ✔，缺失自动安装）
init:
    @echo "== 项目依赖检查 =="
    # Go 工具链
    @if command -v go >/dev/null 2>&1; then printf '  \033[32m✔\033[0m go        %s\n' "$(go version | awk '{print $3}')"; else printf '  \033[31m✘\033[0m go        未安装，请先安装 Go\n'; exit 1; fi
    # Git
    @if command -v git >/dev/null 2>&1; then printf '  \033[32m✔\033[0m git       %s\n' "$(git --version)"; else printf '  \033[33m✘\033[0m git       未安装\n'; fi
    # swag（接口文档）
    @if command -v swag >/dev/null 2>&1; then printf '  \033[32m✔\033[0m swag      %s\n' "$(swag --version 2>/dev/null | head -1)"; else printf '  \033[33m✘\033[0m swag      未安装，正在自动安装...\n'; go install github.com/swaggo/swag/cmd/swag@latest; fi
    # just（命令运行器）
    @if command -v just >/dev/null 2>&1; then printf '  \033[32m✔\033[0m just      %s\n' "$(just --version)"; else printf '  \033[33m✘\033[0m just      未安装，正在自动安装...\n'; go install github.com/casey/just@latest; fi
    # docker（发布需要）
    @if command -v docker >/dev/null 2>&1; then printf '  \033[32m✔\033[0m docker    %s\n' "$(docker --version)"; else printf '  \033[33m✘\033[0m docker    未安装（仅发布需要，可忽略）\n'; fi
    @echo "== 同步依赖模块（go mod tidy）=="
    go mod tidy
    @echo "== 初始化完成 ✔ =="

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
    rm -f ./{{ srv }}
    go env -w CGO_ENABLED=0
    go build -a -ldflags '-extldflags "-static"' -o {{ srv }} main.go
    go env -w CGO_ENABLED=1

# 发布：构建 Docker 镜像并推送到远程仓库
release:
    docker login
    docker build . -t {{ image }}
    docker push {{ image }}
    rm -f ./{{ srv }}

# 文档：生成 swagger 接口文档
docs:
    swag init -o ./app/swagger

# 提交：暂存所有变更并提交
# 用法：just commit "feat(system): 新增xxx模块"
# 不带参数则打开编辑器（自动载入 .gitmessage 模板）
commit message="":
    git add -A
    @if [ -n "{{ message }}" ]; then git commit -m "{{ message }}"; else git commit; fi

# 推送：推送到远程仓库
push:
    git push

# 提交并推送：开发完成一键 commit + push（推送成功才算开发完成）
ship:
    git add -A
    git commit
    git push
