# ─────────────────────────────────────────────────────────────
# 运行镜像：只包含静态编译产物 + 配置结构（config/）
# 配置值通过「容器环境变量」或「.env」注入，不打进镜像
#   构建：just build && just release
#   运行：just docker-run   （等价 docker run --env-file .env）
# ─────────────────────────────────────────────────────────────
FROM alpine:3.20

# 时区与根证书（DSN 中 loc=Local、外部 https 调用需要）
RUN apk add --no-cache tzdata ca-certificates \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /

# 二进制（由 just build 生成）
ADD admintpl-server /admintpl-server
RUN chmod +x /admintpl-server

# 配置结构（只含 ${...} 占位符，值由环境变量提供）
ADD config/ /config/

ENV TZ=Asia/Shanghai

EXPOSE 8080

ENTRYPOINT [ "/admintpl-server" ]
