# ─────────────────────────────────────────────────────────────
# 生产环境配置（pro）
# 使用方式：just env pro   （将本文件复制为 .env，.env 不入库）
# 说明：值中不要写行内注释；系统环境变量优先级高于本文件
# ─────────────────────────────────────────────────────────────

# ---- 服务 ----
SERVICE_PORT=:8080
SERVICE_DEBUG=false

# ---- 数据库 ----
DB_TYPE=mysql
DB_DSN=root:123456@tcp(127.0.0.1:3306)/ruoyi?charset=utf8mb4&parseTime=True&loc=Local
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
