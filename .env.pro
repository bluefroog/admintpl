# ─────────────────────────────────────────────────────────────
# 生产环境配置（pro）
# 使用方式：just env pro   （将本文件复制为 .env，.env 不入库）
# 说明：值中不要写行内注释；系统环境变量优先级高于本文件
# ─────────────────────────────────────────────────────────────

# ---- 配置加密（AES-256-GCM） ----
# 生成密文：just encrypt 'root:123456@tcp(127.0.0.1:3306)/admintpl?charset=utf8mb4&parseTime=True&loc=Local'
# 开启后 DB_DSN 等敏感值可写成 ENC(...) 密文；密钥需与部署环境一致（更换密钥需重新加密）
CONFIG_ENCRYPT_ENABLED=false
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
# 参数 / 字典缓存（关闭后每次查询回源数据库，仅建议排查问题时使用）
CACHE_ENABLED=true

# ---- 代码生成 ----
GEN_PACKAGE_NAME=github.com/bluefroog/admintpl
GEN_MODULE_NAME=system
GEN_AUTHOR=bluefrog
GEN_AUTO_REMOVE_PRE=true
GEN_TABLE_PREFIX=sys_
