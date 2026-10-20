# ─────────────────────────────────────────────────────────────
# just 命令集：开发 / 测试 / 打包 / 发布
# 用法：just <recipe>，例如 `just dev`、`just build`
# 查看全部：just --list
# ─────────────────────────────────────────────────────────────

# 变量
srv := "admintpl-server"
version := "latest"
image := srv + ":" + version

# 将 recipe 参数以位置参数（$1、$2...）传给 shell，避免消息中的 $ 被 shell 展开
set positional-arguments

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

# 环境切换：将 .env.<环境> 复制为 .env（可选 dev / pro）
# 用法：just env dev / just env pro
env env_name="dev":
    @if [ ! -f ".env.{{ env_name }}" ]; then echo "✘ 环境文件 .env.{{ env_name }} 不存在（可选：dev / pro）"; exit 1; fi
    @cp -f ".env.{{ env_name }}" .env
    @echo "== 已切换到 {{ env_name }} 环境：.env.{{ env_name }} -> .env =="

# 内部：确保 .env 存在（缺失时默认按 dev 初始化）
_ensure-env:
    @if [ ! -f .env ]; then cp -f .env.dev .env; echo "== 未找到 .env，已按 dev 环境初始化（just env pro 可切换生产）=="; fi

# 加密配置敏感值（AES-256-GCM），密钥取 .env 的 CONFIG_SECRET_KEY
# 用法：just encrypt 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'
#      输出形如 ENC(xxxx)，写入 .env 的 DB_DSN 并设置 CONFIG_ENCRYPT_ENABLED=true 即启用
encrypt plaintext:
    @go run ./cmd/encrypt -e "$1"

# 加密并输出 .env 配置片段（CONFIG_ENCRYPT_ENABLED / CONFIG_SECRET_KEY / DB_DSN）
encrypt-env plaintext:
    @go run ./cmd/encrypt -env -e "$1"

# 解密配置敏感值（排查用，密钥取 .env 的 CONFIG_SECRET_KEY）
# 用法：just decrypt 'ENC(xxxx)'
decrypt ciphertext:
    @go run ./cmd/encrypt -d "$1"

# 开发：启动项目（默认监听 :8080）
dev: _ensure-env
    go run main.go

# 测试：运行全部单元测试
test:
    go test ./...

# 检查：静态检查（go vet）
vet:
    go vet ./...

# 冒烟：对已启动的实例做读接口 + 全模块 CRUD 回归
# 用法：just smoke 或 just smoke http://192.168.122.129:8080
smoke base_url="http://127.0.0.1:8080":
    bash ./scripts/smoke.sh {{ base_url }}

# 代码生成深度校验：建临时表 -> 导入 -> 校验列配置 -> 同步数据库 -> 清理
# 用法：just gen-check 或 just gen-check http://192.168.122.129:8080
#      第二个参数为 mysql 客户端命令（默认走本机 docker exec mysql）
gen-check base_url="http://127.0.0.1:8080" mysql_cmd="docker exec mysql mysql -uroot -p123456 --batch --skip-column-names admintpl -e":
    bash ./scripts/gen-check.sh {{ base_url }} "{{ mysql_cmd }}"

# 打包：编译 linux/amd64 静态二进制（Docker 镜像内运行，勿用本机平台二进制）
build:
    rm -f ./{{ srv }}
    WSLENV=GOOS:GOARCH:CGO_ENABLED GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o {{ srv }} main.go
    @file ./{{ srv }} | grep -q ELF || { echo "!! 产物不是 Linux ELF：交叉编译环境变量未生效（WSL 调用 Windows go 需 WSLENV 透传）"; exit 1; }
    @echo "== 已生成 linux/amd64 静态二进制：{{ srv }} ✔ =="

# 发布：构建 Docker 镜像并推送到远程仓库
release: _ensure-env
    docker login
    docker build . -t {{ image }}
    docker push {{ image }}
    rm -f ./{{ srv }}

# 容器运行：本地以容器方式启动（用 .env 注入环境变量，监听 :8080）
docker-run: _ensure-env
    docker run --rm -it -p 8080:8080 --env-file .env {{ image }}

# 文档：生成 swagger 接口文档
docs:
    swag init -o ./app/swagger

# 提交：暂存所有变更并提交
# 用法：just commit "feat(system): 新增xxx模块"
# 不带参数则打开编辑器（自动载入 .gitmessage 模板）
commit message="":
    git add -A
    @if [ -n "$1" ]; then printf '%s' "$1" | git commit -F -; else git commit; fi

# 推送：推送到远程仓库
push:
    git push

# 提交并推送：开发完成一键 commit + push（推送成功才算开发完成）
ship:
    git add -A
    git commit
    git push
