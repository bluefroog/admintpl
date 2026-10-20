package cache

import (
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bluefroog/admintpl/common/config"
	"github.com/sirupsen/logrus"
)

// 缓存后端驱动（cache.driver / CACHE_DRIVER）
const (
	// DriverFile 本地文件：进程内 map + gob 落盘（默认，单机部署无需外部依赖）
	DriverFile = "file"
	// DriverMemory 纯进程内内存：不持久化，仅测试或临时排查时使用
	DriverMemory = "memory"
	// DriverMemcache memcached
	DriverMemcache = "memcache"
	// DriverRedis redis（单机 / 主从 / 哨兵 / 集群）
	DriverRedis = "redis"
)

// redis 部署模式（cache.redis.mode / CACHE_REDIS_MODE）
const (
	// RedisModeStandalone 单机
	RedisModeStandalone = "standalone"
	// RedisModeMasterSlave 主从：addr 或 addrs[0] 为主节点，其余为从节点（主节点故障时逐个尝试）
	RedisModeMasterSlave = "master-slave"
	// RedisModeSentinel 哨兵：sentinel-addrs + master-name 自动发现主节点
	RedisModeSentinel = "sentinel"
	// RedisModeCluster 集群：cluster-addrs 为集群节点列表
	RedisModeCluster = "cluster"
)

// 默认值（配置缺失时使用）
const (
	DefaultDriver           = DriverFile
	DefaultFileDir          = "runtime/cache"
	DefaultFileFlushSeconds = 5
	DefaultMemcacheAddr     = "127.0.0.1:11211"
	DefaultMemcacheTimeout  = 1 * time.Second
	DefaultRedisAddr        = "127.0.0.1:6379"
	DefaultRedisMasterName  = "mymaster"
	DefaultRedisTimeout     = 1 * time.Second
	DefaultRedisPoolSize    = 10
)

// 缓存后端配置项（config/config.yaml -> cache.*）
const (
	// KeyDriver 后端驱动：file / memory / memcache / redis
	KeyDriver = "cache.driver"
	// KeyPrefix 缓存键前缀（多环境共用一个 memcache/redis 时用于隔离）
	KeyPrefix = "cache.prefix"

	// KeyFileDir 本地文件目录
	KeyFileDir = "cache.file.dir"
	// KeyFileFlushInterval 本地文件落盘间隔（秒）
	KeyFileFlushInterval = "cache.file.flush-interval"

	// KeyMemcacheServers memcache 节点列表（逗号分隔）
	KeyMemcacheServers = "cache.memcache.servers"
	// KeyMemcacheTimeout memcache 超时（秒）
	KeyMemcacheTimeout = "cache.memcache.timeout"

	// KeyRedisMode redis 部署模式：standalone / master-slave / sentinel / cluster
	KeyRedisMode = "cache.redis.mode"
	// KeyRedisAddr redis 单机/主节点地址
	KeyRedisAddr = "cache.redis.addr"
	// KeyRedisAddrs redis 主从节点列表（主节点在前，逗号分隔）
	KeyRedisAddrs = "cache.redis.addrs"
	// KeyRedisClusterAddrs redis 集群节点列表（逗号分隔）
	KeyRedisClusterAddrs = "cache.redis.cluster-addrs"
	// KeyRedisMasterName 哨兵模式的主节点名
	KeyRedisMasterName = "cache.redis.master-name"
	// KeyRedisSentinelAddrs 哨兵节点列表（逗号分隔）
	KeyRedisSentinelAddrs = "cache.redis.sentinel-addrs"
	// KeyRedisSentinelPassword 哨兵自身密码
	KeyRedisSentinelPassword = "cache.redis.sentinel-password"
	// KeyRedisUsername redis 6+ ACL 用户名
	KeyRedisUsername = "cache.redis.username"
	// KeyRedisPassword redis 密码
	KeyRedisPassword = "cache.redis.password"
	// KeyRedisDB 逻辑库编号（单机/主从/哨兵有效，集群固定 0）
	KeyRedisDB = "cache.redis.db"
	// KeyRedisTimeout 连接/读写超时（秒）
	KeyRedisTimeout = "cache.redis.timeout"
	// KeyRedisPoolSize 连接池大小
	KeyRedisPoolSize = "cache.redis.pool-size"
)

var (
	// ErrNotFound 缓存键不存在（含已过期）
	ErrNotFound = errors.New("缓存键不存在")
	// ErrExpired 缓存键已过期（本地文件 / 内存后端可精确区分）
	ErrExpired = errors.New("缓存键已过期")
	// ErrUnsupported 当前后端不支持该能力（如 memcache 无法枚举缓存键）
	ErrUnsupported = errors.New("当前缓存后端不支持该操作")
)

// RawEntry 缓存条目原始数据（后端枚举用）
type RawEntry struct {
	Key        string // 完整缓存键（含前缀）
	Payload    []byte // 序列化后的缓存值
	Expiration int64  // 过期时间（毫秒时间戳），0 表示永不过期
}

// Store 缓存后端接口：Cache 只依赖该接口，具体实现见 store_*.go
//
// 约定：key 为调用方传入的完整键（Cache 已拼接前缀），后端不再做前缀处理。
type Store interface {
	// Kind 后端类型（file / memory / memcache / redis，监控页展示用）
	Kind() string
	// Set 写入原始载荷（expiration 为毫秒时间戳，0 表示永不过期）
	Set(key string, payload []byte, expiration int64) error
	// Get 读取原始载荷，键不存在返回 ErrNotFound，已过期返回 ErrExpired
	Get(key string) ([]byte, error)
	// Delete 删除键（不存在不报错）
	Delete(keys ...string) error
	// Entries 返回指定前缀下的全部有效条目，后端不支持枚举时返回 ErrUnsupported
	Entries(prefix string) ([]RawEntry, error)
	// Clear 删除指定前缀下的全部键
	Clear(prefix string) error
	// Close 释放后端资源（本地文件落盘、redis 连接关闭）
	Close() error
}

// Sweeper 支持主动清理过期键的后端（本地文件 / 内存）
type Sweeper interface {
	// Sweep 清理指定前缀下的过期键，返回清理数量
	Sweep(prefix string) (int, error)
}

// Pinger 支持探活的后端（redis / memcache）
type Pinger interface {
	// Ping 探活
	Ping() error
}

// FileOptions 本地文件后端配置
type FileOptions struct {
	Dir           string        // 数据文件目录（cache.gob）
	FlushInterval time.Duration // 落盘间隔
}

// MemcacheOptions memcache 后端配置
type MemcacheOptions struct {
	Servers []string      // 节点列表
	Timeout time.Duration // 超时
}

// RedisOptions redis 后端配置
type RedisOptions struct {
	Mode             string        // standalone / master-slave / sentinel / cluster
	Addr             string        // 单机/主节点地址
	Addrs            []string      // 主从节点列表（主节点在前）
	ClusterAddrs     []string      // 集群节点列表
	MasterName       string        // 哨兵主节点名
	SentinelAddrs    []string      // 哨兵节点列表
	SentinelPassword string        // 哨兵密码
	Username         string        // ACL 用户名
	Password         string        // 密码
	DB               int           // 逻辑库
	Timeout          time.Duration // 超时
	PoolSize         int           // 连接池大小
}

// Options 缓存后端配置
type Options struct {
	Driver   string          // 后端驱动
	Prefix   string          // 键前缀（Cache 拼接：prefix + 缓存名 + ":" + key）
	File     FileOptions     // 本地文件配置
	Memcache MemcacheOptions // memcache 配置
	Redis    RedisOptions    // redis 配置
}

// LoadOptions 从配置文件（cache.*）读取缓存后端配置
// @return: Options 缓存后端配置
func LoadOptions() Options {
	fileFlush := config.Int(KeyFileFlushInterval)
	if fileFlush <= 0 {
		fileFlush = DefaultFileFlushSeconds
	}
	memcacheTimeout := config.Int(KeyMemcacheTimeout)
	if memcacheTimeout <= 0 {
		memcacheTimeout = int(DefaultMemcacheTimeout / time.Second)
	}
	redisTimeout := config.Int(KeyRedisTimeout)
	if redisTimeout <= 0 {
		redisTimeout = int(DefaultRedisTimeout / time.Second)
	}
	poolSize := config.Int(KeyRedisPoolSize)
	if poolSize <= 0 {
		poolSize = DefaultRedisPoolSize
	}

	opts := Options{
		Driver: resolveDriver(),
		Prefix: config.String(KeyPrefix),
		File: FileOptions{
			Dir:           stringValue(KeyFileDir, DefaultFileDir),
			FlushInterval: time.Duration(fileFlush) * time.Second,
		},
		Memcache: MemcacheOptions{
			Servers: listValue(KeyMemcacheServers, DefaultMemcacheAddr),
			Timeout: time.Duration(memcacheTimeout) * time.Second,
		},
		Redis: RedisOptions{
			Mode:             normalizeRedisMode(config.String(KeyRedisMode)),
			Addr:             stringValue(KeyRedisAddr, DefaultRedisAddr),
			Addrs:            listValue(KeyRedisAddrs, ""),
			ClusterAddrs:     listValue(KeyRedisClusterAddrs, ""),
			MasterName:       stringValue(KeyRedisMasterName, DefaultRedisMasterName),
			SentinelAddrs:    listValue(KeyRedisSentinelAddrs, ""),
			SentinelPassword: config.String(KeyRedisSentinelPassword),
			Username:         config.String(KeyRedisUsername),
			Password:         config.String(KeyRedisPassword),
			DB:               config.Int(KeyRedisDB),
			Timeout:          time.Duration(redisTimeout) * time.Second,
			PoolSize:         poolSize,
		},
	}
	return opts
}

// resolveDriver 解析缓存驱动：配置 > 环境变量 > 本地文件
//
// 特殊处理：完全没有配置文件（如单元测试、未部署配置的裸跑）时使用内存后端，
// 避免在任意工作目录下生成 cache.gob。
func resolveDriver() string {
	if raw := strings.TrimSpace(config.String(KeyDriver)); raw != "" {
		return normalizeDriver(raw)
	}
	if !config.IsSet(KeyDriver) && strings.TrimSpace(os.Getenv("CACHE_DRIVER")) == "" {
		return DriverMemory
	}
	return DefaultDriver
}

// normalizeDriver 归一化驱动名（缺省为本地文件）
func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", DriverFile, "local", "local-file", "localfile":
		return DriverFile
	case DriverMemory, "in-memory", "inmemory", "local-memory":
		return DriverMemory
	case DriverMemcache, "memcached":
		return DriverMemcache
	case DriverRedis:
		return DriverRedis
	default:
		logrus.Warnf("未知的缓存后端 %q，已回退为 %s（可选：file / memory / memcache / redis）", driver, DefaultDriver)
		return DefaultDriver
	}
}

// normalizeRedisMode 归一化 redis 部署模式（缺省为单机）
func normalizeRedisMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", RedisModeStandalone, "single", "master":
		return RedisModeStandalone
	case RedisModeMasterSlave, "master_slave", "replication":
		return RedisModeMasterSlave
	case RedisModeSentinel, "failover":
		return RedisModeSentinel
	case RedisModeCluster:
		return RedisModeCluster
	default:
		logrus.Warnf("未知的 redis 模式 %q，已回退为 %s（可选：standalone / master-slave / sentinel / cluster）",
			mode, RedisModeStandalone)
		return RedisModeStandalone
	}
}

// stringValue 读取字符串配置，空值回退默认值
func stringValue(key, fallback string) string {
	if value := strings.TrimSpace(config.String(key)); value != "" {
		return value
	}
	return fallback
}

// listValue 读取列表配置：兼容 YAML 数组与「逗号分隔字符串」两种写法
func listValue(key, fallback string) []string {
	values := make([]string, 0, 4)
	for _, item := range config.StringArray(key) {
		values = append(values, splitList(item)...)
	}
	if len(values) == 0 && fallback != "" {
		values = append(values, fallback)
	}
	return values
}

// splitList 把「逗号分隔字符串」拆成列表（去空、去首尾空格）
func splitList(value string) []string {
	values := make([]string, 0, 4)
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			values = append(values, part)
		}
	}
	return values
}

// newStore 按配置创建缓存后端
// @param: opts Options 缓存后端配置
// @return: Store 缓存后端, error 错误信息
func newStore(opts Options) (Store, error) {
	switch opts.Driver {
	case DriverMemory:
		return newMemoryStore(), nil
	case DriverMemcache:
		return newMemcacheStore(opts.Memcache)
	case DriverRedis:
		return newRedisStore(opts.Redis)
	case DriverFile:
		return newFileStore(opts.File)
	default:
		return nil, errors.New("未知的缓存后端: " + opts.Driver)
	}
}

var (
	sharedMu    sync.Mutex
	sharedOnce  sync.Once
	sharedValue Store
)

// sharedStore 进程级共享缓存后端（按配置创建一次，各缓存实例复用连接/文件）
// @return: Store 缓存后端
func sharedStore() Store {
	sharedOnce.Do(func() {
		opts := LoadOptions()
		store, err := newStore(opts)
		if err != nil {
			logrus.Errorf("缓存后端初始化失败（%s），已回退为本地文件: %s", opts.Driver, err)
			if fallback, fallbackErr := newStore(Options{Driver: DriverFile, File: opts.File}); fallbackErr == nil {
				store = fallback
			} else {
				store = newMemoryStore()
			}
		}
		// 探活：外部后端连不上时不阻塞启动，仅记录告警（缓存失效会自动回源数据库）
		if pinger, ok := store.(Pinger); ok {
			if err := pinger.Ping(); err != nil {
				logrus.Warnf("缓存后端 %s 探活失败（缓存将自动回源数据库）: %s", store.Kind(), err)
			} else {
				logrus.Infof("缓存后端已就绪: %s", store.Kind())
			}
		} else {
			logrus.Infof("缓存后端已就绪: %s", store.Kind())
		}
		sharedMu.Lock()
		sharedValue = store
		sharedMu.Unlock()
	})
	sharedMu.Lock()
	defer sharedMu.Unlock()
	return sharedValue
}

// Shutdown 释放进程级缓存后端（本地文件落盘、关闭 redis/memcache 连接）
// 建议在服务退出前调用，避免丢失落盘间隔内的缓存写入
func Shutdown() {
	sharedMu.Lock()
	store := sharedValue
	sharedMu.Unlock()
	if store != nil {
		if err := store.Close(); err != nil {
			logrus.Warnf("缓存后端关闭失败: %s", err)
		}
	}
}
