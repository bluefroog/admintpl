package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisScanCount SCAN 每批返回的键数量
const redisScanCount = 200

// redisKV redis 命令集合（*redis.Client 与 *redis.ClusterClient 均实现）
type redisKV interface {
	Ping(ctx context.Context) *redis.StatusCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	PTTL(ctx context.Context, key string) *redis.DurationCmd
	Close() error
}

// redisScanFunc 键枚举策略（单机/集群/主从各不相同）
type redisScanFunc func(ctx context.Context, match string) ([]string, error)

// redisStore redis 后端：单机 / 主从 / 哨兵 / 集群共用一套 Store 实现
type redisStore struct {
	deployMode string          // 部署模式（监控显示用）
	kv         redisKV         // 命令集合
	scan       redisScanFunc   // 键枚举策略
	ctx        context.Context // 命令上下文
}

// newRedisStore 创建 redis 后端
// @param: opts RedisOptions redis 配置
// @return: *redisStore, error 错误信息
func newRedisStore(opts RedisOptions) (*redisStore, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultRedisTimeout
	}
	poolSize := opts.PoolSize
	if poolSize <= 0 {
		poolSize = DefaultRedisPoolSize
	}
	store := &redisStore{deployMode: opts.Mode, ctx: context.Background()}

	switch opts.Mode {
	case RedisModeCluster:
		addrs := opts.ClusterAddrs
		if len(addrs) == 0 {
			addrs = opts.Addrs
		}
		if len(addrs) == 0 && opts.Addr != "" {
			addrs = []string{opts.Addr}
		}
		if len(addrs) == 0 {
			return nil, errors.New("redis 集群模式需要配置 system.cache.redis.cluster-addrs（CACHE_REDIS_CLUSTER_ADDRS）")
		}
		cluster := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        addrs,
			Username:     opts.Username,
			Password:     opts.Password,
			DialTimeout:  timeout,
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
			PoolSize:     poolSize,
		})
		store.kv = cluster
		store.scan = func(ctx context.Context, match string) ([]string, error) {
			return scanCluster(ctx, cluster, match)
		}

	case RedisModeSentinel:
		sentinels := opts.SentinelAddrs
		if len(sentinels) == 0 {
			sentinels = opts.Addrs
		}
		if len(sentinels) == 0 && opts.Addr != "" {
			sentinels = []string{opts.Addr}
		}
		if len(sentinels) == 0 {
			return nil, errors.New("redis 哨兵模式需要配置 system.cache.redis.sentinel-addrs（CACHE_REDIS_SENTINEL_ADDRS）")
		}
		client := redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       opts.MasterName,
			SentinelAddrs:    sentinels,
			Username:         opts.Username,
			Password:         opts.Password,
			SentinelPassword: opts.SentinelPassword,
			DB:               opts.DB,
			DialTimeout:      timeout,
			ReadTimeout:      timeout,
			WriteTimeout:     timeout,
			PoolSize:         poolSize,
			MaxRetries:       1, // 缓存失败即回源数据库，不做过多次重试
		})
		store.kv = client
		store.scan = func(ctx context.Context, match string) ([]string, error) {
			return scanClient(ctx, client, match)
		}

	case RedisModeMasterSlave:
		addrs := opts.Addrs
		if len(addrs) == 0 {
			addrs = []string{opts.Addr}
		}
		if len(addrs) == 0 || addrs[0] == "" {
			return nil, errors.New("redis 主从模式需要配置 system.cache.redis.addrs（CACHE_REDIS_ADDRS，主节点在前）")
		}
		nodes := make([]*redis.Client, 0, len(addrs))
		for _, addr := range addrs {
			nodes = append(nodes, redis.NewClient(&redis.Options{
				Addr:         addr,
				Username:     opts.Username,
				Password:     opts.Password,
				DB:           opts.DB,
				DialTimeout:  timeout,
				ReadTimeout:  timeout,
				WriteTimeout: timeout,
				PoolSize:     poolSize,
				MaxRetries:   1, // 缓存失败即回源数据库，不做过多次重试
			}))
		}
		failover := &failoverKV{master: nodes[0], replicas: nodes[1:]}
		store.kv = failover
		store.scan = func(ctx context.Context, match string) ([]string, error) {
			return failover.scanAll(ctx, match)
		}

	default: // RedisModeStandalone
		addr := opts.Addr
		if addr == "" {
			addr = DefaultRedisAddr
		}
		client := redis.NewClient(&redis.Options{
			Addr:         addr,
			Username:     opts.Username,
			Password:     opts.Password,
			DB:           opts.DB,
			DialTimeout:  timeout,
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
			PoolSize:     poolSize,
			MaxRetries:   1, // 缓存失败即回源数据库，不做过多次重试
		})
		store.kv = client
		store.scan = func(ctx context.Context, match string) ([]string, error) {
			return scanClient(ctx, client, match)
		}
	}
	return store, nil
}

// Kind 后端类型
func (s *redisStore) Kind() string { return DriverRedis }

// Mode redis 部署模式
func (s *redisStore) Mode() string { return s.deployMode }

// Set 写入缓存（过期时间交给 redis 自身维护）
func (s *redisStore) Set(key string, payload []byte, expiration int64) error {
	var ttl time.Duration
	if expiration > 0 {
		ttl = time.Until(time.UnixMilli(expiration))
		if ttl <= 0 {
			return nil // 已过期，无需写入
		}
	}
	return s.kv.Set(s.ctx, key, payload, ttl).Err()
}

// Get 读取缓存
func (s *redisStore) Get(key string) ([]byte, error) {
	value, err := s.kv.Get(s.ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return value, nil
}

// Delete 删除缓存
func (s *redisStore) Delete(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.kv.Del(s.ctx, keys...).Err()
}

// Entries 枚举缓存条目（SCAN + GET + PTTL）
func (s *redisStore) Entries(prefix string) ([]RawEntry, error) {
	keys, err := s.scan(s.ctx, escapeGlob(prefix)+"*")
	if err != nil {
		return nil, err
	}
	entries := make([]RawEntry, 0, len(keys))
	now := time.Now()
	for _, key := range keys {
		payload, err := s.kv.Get(s.ctx, key).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue // SCAN 与 GET 之间被删除
			}
			return nil, err
		}
		var expiration int64
		if ttl, err := s.kv.PTTL(s.ctx, key).Result(); err == nil && ttl > 0 {
			expiration = now.Add(ttl).UnixMilli()
		}
		entries = append(entries, RawEntry{Key: key, Payload: payload, Expiration: expiration})
	}
	return entries, nil
}

// Clear 删除指定前缀下的缓存（SCAN + DEL，分批删除避免单次命令过大）
func (s *redisStore) Clear(prefix string) error {
	keys, err := s.scan(s.ctx, escapeGlob(prefix)+"*")
	if err != nil {
		return err
	}
	if s.deployMode == RedisModeCluster {
		// 集群模式下 DEL 多个键要求键落在同一 slot，否则报 CROSSSLOT，因此逐个删除
		for _, key := range keys {
			if err = s.kv.Del(s.ctx, key).Err(); err != nil {
				return err
			}
		}
		return nil
	}
	for start := 0; start < len(keys); start += redisScanCount {
		end := start + redisScanCount
		if end > len(keys) {
			end = len(keys)
		}
		if err = s.kv.Del(s.ctx, keys[start:end]...).Err(); err != nil {
			return err
		}
	}
	return nil
}

// Ping 探活
func (s *redisStore) Ping() error { return s.kv.Ping(s.ctx).Err() }

// Close 关闭连接
func (s *redisStore) Close() error { return s.kv.Close() }

// failoverKV redis 主从：写读都优先主节点，主节点不可用时按顺序尝试从节点
//
// 说明：主从模式的从节点是只读副本，正常情况不会用于写入；
// 这里仅在主节点连接失败（如主节点已故障但尚未完成主从切换）时尝试后续节点，
// 既能读到最新数据，也能在主节点漂移后自动恢复写入。
type failoverKV struct {
	master   *redis.Client
	replicas []*redis.Client
}

// nodes 全部节点（主节点在前）
func (f *failoverKV) nodes() []*redis.Client { return append([]*redis.Client{f.master}, f.replicas...) }

// do 依次尝试各节点，返回第一个成功的结果
func (f *failoverKV) do(op func(client *redis.Client) error) error {
	var firstErr error
	for _, node := range f.nodes() {
		err := op(node)
		if err == nil {
			return nil
		}
		if errors.Is(err, redis.Nil) {
			return err
		}
		if firstErr == nil {
			firstErr = err
		}
		if !isConnectError(err) {
			break // 非连接类错误（如权限/命令错误）换节点也无意义
		}
	}
	return firstErr
}

// Ping 探活
func (f *failoverKV) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := f.master.Ping(ctx)
	if cmd.Err() == nil || !isConnectError(cmd.Err()) {
		return cmd
	}
	for _, node := range f.replicas {
		if next := node.Ping(ctx); next.Err() == nil {
			return next
		}
	}
	return cmd
}

// Set 写入缓存
func (f *failoverKV) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	var result *redis.StatusCmd
	_ = f.do(func(node *redis.Client) error {
		result = node.Set(ctx, key, value, expiration)
		return result.Err()
	})
	if result == nil {
		result = f.master.Set(ctx, key, value, expiration)
	}
	return result
}

// Get 读取缓存
func (f *failoverKV) Get(ctx context.Context, key string) *redis.StringCmd {
	var result *redis.StringCmd
	_ = f.do(func(node *redis.Client) error {
		result = node.Get(ctx, key)
		return result.Err()
	})
	if result == nil {
		result = f.master.Get(ctx, key)
	}
	return result
}

// Del 删除缓存
func (f *failoverKV) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	var result *redis.IntCmd
	_ = f.do(func(node *redis.Client) error {
		result = node.Del(ctx, keys...)
		return result.Err()
	})
	if result == nil {
		result = f.master.Del(ctx, keys...)
	}
	return result
}

// PTTL 剩余有效期
func (f *failoverKV) PTTL(ctx context.Context, key string) *redis.DurationCmd {
	var result *redis.DurationCmd
	_ = f.do(func(node *redis.Client) error {
		result = node.PTTL(ctx, key)
		return result.Err()
	})
	if result == nil {
		result = f.master.PTTL(ctx, key)
	}
	return result
}

// Close 关闭全部节点连接
func (f *failoverKV) Close() error {
	var firstErr error
	for _, node := range f.nodes() {
		if err := node.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// scanAll 枚举键（优先主节点）
func (f *failoverKV) scanAll(ctx context.Context, match string) ([]string, error) {
	var keys []string
	err := f.do(func(node *redis.Client) error {
		found, scanErr := scanClient(ctx, node, match)
		if scanErr != nil {
			return scanErr
		}
		keys = found
		return nil
	})
	return keys, err
}

// scanClient 通过 SCAN 游标遍历单个节点
func scanClient(ctx context.Context, client *redis.Client, match string) ([]string, error) {
	keys := make([]string, 0, 32)
	var cursor uint64
	for {
		batch, next, err := client.Scan(ctx, cursor, match, redisScanCount).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

// scanCluster 遍历集群中每个主节点的键
func scanCluster(ctx context.Context, cluster *redis.ClusterClient, match string) ([]string, error) {
	keys := make([]string, 0, 32)
	err := cluster.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		found, scanErr := scanClient(ctx, client, match)
		if scanErr != nil {
			return scanErr
		}
		keys = append(keys, found...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// escapeGlob 转义 SCAN MATCH 中的通配符，避免前缀里的特殊字符被当作 glob
func escapeGlob(s string) string {
	var builder strings.Builder
	for _, char := range s {
		switch char {
		case '*', '?', '[', ']', '\\':
			builder.WriteByte('\\')
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

// isConnectError 判断是否为连接层错误（用于主从模式的节点切换）
func isConnectError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, redis.Nil) {
		return false
	}
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, keyword := range []string{
		"connection refused", // Linux/macOS：连接被拒绝
		"actively refused",   // Windows：目标机器积极拒绝
		"no connection could be made",
		"connect: no route to host", "network is unreachable",
		"i/o timeout", "connection reset", "broken pipe", "eof",
		"no such host", "connection attempt failed",
	} {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

// redisModeName 部署模式的中文名（日志/监控展示用）
func redisModeName(mode string) string {
	switch mode {
	case RedisModeMasterSlave:
		return "主从"
	case RedisModeSentinel:
		return "哨兵"
	case RedisModeCluster:
		return "集群"
	default:
		return "单机"
	}
}

// String 便于日志输出
func (s *redisStore) String() string {
	return fmt.Sprintf("redis(%s)", redisModeName(s.deployMode))
}
