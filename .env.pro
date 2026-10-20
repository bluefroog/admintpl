# ─────────────────────────────────────────────────────────────
# 生产环境配置（pro）
# 使用方式：just env pro   （将本文件复制为 .env，.env 不入库）
# 说明：值中不要写行内注释；系统环境变量优先级高于本文件
# ─────────────────────────────────────────────────────────────

# ---- 配置加密（AES-256-GCM / SM4-CBC） ----
# 生成密文：just encrypt 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'
# 算法可选 AES（AES-256-GCM，默认）/ SM4（国密 SM4-CBC）：just encrypt '明文' SM4
# 开启后 DB_DSN 等敏感值可写成 ENC(...) 密文；密钥需与部署环境一致（更换密钥需重新加密）
CONFIG_ENCRYPT_ENABLED=false
CONFIG_SECRET_ALGORITHM=AES
CONFIG_SECRET_KEY=

# ---- 服务 ----
SERVICE_PORT=:8080
SERVICE_DEBUG=false

# ---- 数据库 ----
DB_TYPE=mysql
DB_DSN=root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local
DB_PREFIX=
DB_MAX_IDLE=50
DB_MAX_OPEN=100
DB_MAX_LIFETIME=3600
DB_DEBUG=false

# ---- 登录认证 ----
AUTH_ENABLED=true
AUTH_TOKEN_EXPIRE=720
AUTH_MAX_RETRY=5
AUTH_LOCK_TIME=10
AUTH_PERMISSION_ENABLED=false

# ---- 图形验证码 ----
CAPTCHA_ENABLED=true
CAPTCHA_LENGTH=4
CAPTCHA_EXPIRE=120

# ---- 基础数据初始化（生产环境默认关闭） ----
SEED_ENABLED=false

# ---- 定时任务调度 ----
JOB_ENABLED=true
JOB_LOG_RETAIN_DAYS=30

# ---- 数据权限 ----
DATA_SCOPE_ENABLED=true
# ---- 参数 / 字典缓存 ----
# 是否启用缓存（关闭后每次查询回源数据库，仅建议排查问题时使用）
CACHE_ENABLED=true
# 后端：file 本地文件（默认）/ memory 内存 / memcache / redis
CACHE_DRIVER=file
# 键前缀（多环境共用一个 memcache/redis 时用于隔离）
CACHE_PREFIX=admintpl
# 本地文件后端：数据目录与落盘间隔（秒）
CACHE_FILE_DIR=runtime/cache
CACHE_FILE_FLUSH_INTERVAL=5
# memcache 后端：节点列表（逗号分隔）与超时（秒）
CACHE_MEMCACHE_SERVERS=127.0.0.1:11211
CACHE_MEMCACHE_TIMEOUT=1
# redis 后端：模式 standalone / master-slave / sentinel / cluster
CACHE_REDIS_MODE=standalone
CACHE_REDIS_ADDR=127.0.0.1:6379
CACHE_REDIS_ADDRS=
CACHE_REDIS_CLUSTER_ADDRS=
CACHE_REDIS_MASTER_NAME=mymaster
CACHE_REDIS_SENTINEL_ADDRS=
CACHE_REDIS_SENTINEL_PASSWORD=
CACHE_REDIS_USERNAME=
CACHE_REDIS_PASSWORD=
CACHE_REDIS_DB=0
CACHE_REDIS_TIMEOUT=1
CACHE_REDIS_POOL_SIZE=10

# ---- 代码生成 ----
GEN_PACKAGE_NAME=github.com/bluefroog/admintpl
GEN_MODULE_NAME=system
GEN_AUTHOR=bluefrog
GEN_AUTO_REMOVE_PRE=true
GEN_TABLE_PREFIX=sys_
