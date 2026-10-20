// Package s3 提供统一的文件（对象）存储能力，业务代码只依赖 Storage 接口，切换后端无需改代码。
//
// 支持的后端（s3.driver / S3_DRIVER）：
//   - local  本地目录（默认）：文件写到 s3.local.dir，对外访问前缀 s3.local.url-prefix
//   - minio  MinIO：S3 兼容对象存储
//   - rustfs RustFS：S3 兼容对象存储（默认路径风格寻址，适合自建/裸金属部署）
//   - cos    腾讯云 COS
//   - oss    阿里云 OSS
//
// 典型用法：
//
//	st, err := s3.Default() // 读取 s3.* 配置并初始化（s3.enabled=false 时返回 ErrDisabled）
//	if err != nil { ... }
//	obj, err := st.Put(ctx, s3.GenerateKey("avatar.png"), file, size, "image/png")
//	url := st.URL(obj.Key)
package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bluefroog/admintpl/common/config"
	"github.com/sirupsen/logrus"
)

// 存储驱动（s3.driver / S3_DRIVER）
const (
	// DriverLocal 本地目录：使用指定的本地目录，不需要任何外部依赖
	DriverLocal = "local"
	// DriverMinio MinIO（S3 兼容，虚拟主机寻址优先）
	DriverMinio = "minio"
	// DriverRustFS RustFS（S3 兼容，默认路径风格寻址）
	DriverRustFS = "rustfs"
	// DriverCos 腾讯云 COS
	DriverCos = "cos"
	// DriverOss 阿里云 OSS
	DriverOss = "oss"
)

// 默认值（配置缺失时使用）
const (
	DefaultDriver            = DriverLocal
	DefaultLocalDir          = "runtime/uploads"
	DefaultLocalURLPrefix    = "/uploads"
	DefaultMinioEndpoint     = "127.0.0.1:9000"
	DefaultMinioBucket       = "admintpl"
	DefaultRustFSEndpoint    = "127.0.0.1:9000"
	DefaultRustFSBucket      = "admintpl"
	DefaultRustFSRegion      = "us-east-1"
	DefaultOssEndpoint       = "oss-cn-hangzhou.aliyuncs.com"
	DefaultOssBucket         = "admintpl"
	DefaultContentType       = "application/octet-stream"
	DefaultListLimit         = 1000
	DefaultPresignExpiry     = 15 * time.Minute
	defaultTimeoutSeconds    = 30
	defaultMaxIdleConns      = 100
	objectKeyRandomByteCount = 16
)

// 文件存储配置项（config/config.yaml -> s3.*）
const (
	// KeyEnabled 是否启用文件存储（关闭时 New/Default 返回 ErrDisabled）
	KeyEnabled = "s3.enabled"
	// KeyDriver 存储驱动：local / minio / rustfs / cos / oss
	KeyDriver = "s3.driver"
	// KeyPublicBaseURL 对象访问地址的公共前缀（CDN / 反向代理域名），为空时各后端使用自身默认值
	KeyPublicBaseURL = "s3.public-base-url"

	// KeyLocalDir 本地目录（driver=local 时文件写入该目录）
	KeyLocalDir = "s3.local.dir"
	// KeyLocalURLPrefix 本地目录对外访问前缀（配合静态文件路由 / 反向代理使用）
	KeyLocalURLPrefix = "s3.local.url-prefix"

	// KeyMinioEndpoint MinIO 服务地址（host:port 或带 scheme 的 URL）
	KeyMinioEndpoint = "s3.minio.endpoint"
	// KeyMinioAccessKey MinIO Access Key
	KeyMinioAccessKey = "s3.minio.access-key"
	// KeyMinioSecretKey MinIO Secret Key（可用 ENC(...) 密文）
	KeyMinioSecretKey = "s3.minio.secret-key"
	// KeyMinioBucket MinIO 存储桶
	KeyMinioBucket = "s3.minio.bucket"
	// KeyMinioRegion MinIO 区域（留空自动向服务端查询）
	KeyMinioRegion = "s3.minio.region"
	// KeyMinioSecure 是否使用 HTTPS
	KeyMinioSecure = "s3.minio.secure"
	// KeyMinioPathStyle 是否强制路径风格寻址（endpoint 为 IP / 无泛域名解析时需开启）
	KeyMinioPathStyle = "s3.minio.path-style"

	// KeyRustFSEndpoint RustFS 服务地址（host:port 或带 scheme 的 URL）
	KeyRustFSEndpoint = "s3.rustfs.endpoint"
	// KeyRustFSAccessKey RustFS Access Key
	KeyRustFSAccessKey = "s3.rustfs.access-key"
	// KeyRustFSSecretKey RustFS Secret Key（可用 ENC(...) 密文）
	KeyRustFSSecretKey = "s3.rustfs.secret-key"
	// KeyRustFSBucket RustFS 存储桶
	KeyRustFSBucket = "s3.rustfs.bucket"
	// KeyRustFSRegion RustFS 区域（S3 兼容实现通常固定 us-east-1）
	KeyRustFSRegion = "s3.rustfs.region"
	// KeyRustFSSecure 是否使用 HTTPS
	KeyRustFSSecure = "s3.rustfs.secure"
	// KeyRustFSPathStyle 是否强制路径风格寻址（RustFS 默认 true）
	KeyRustFSPathStyle = "s3.rustfs.path-style"

	// KeyCosBucketURL 腾讯云 COS 存储桶访问域名（https://<bucket>-<appid>.cos.<region>.myqcloud.com）
	KeyCosBucketURL = "s3.cos.bucket-url"
	// KeyCosBucket 存储桶名（bucket-url 留空时与 region 一起拼接访问域名，形如 <bucket>-<appid>）
	KeyCosBucket = "s3.cos.bucket"
	// KeyCosRegion 存储桶地域（如 ap-guangzhou，bucket-url 留空时使用）
	KeyCosRegion = "s3.cos.region"
	// KeyCosSecretID 腾讯云 SecretId
	KeyCosSecretID = "s3.cos.secret-id"
	// KeyCosSecretKey 腾讯云 SecretKey（可用 ENC(...) 密文）
	KeyCosSecretKey = "s3.cos.secret-key"
	// KeyCosDomain 自定义访问域名（CDN / 自有域名），为空时使用 bucket-url
	KeyCosDomain = "s3.cos.domain"

	// KeyOssEndpoint 阿里云 OSS 访问域名（如 oss-cn-hangzhou.aliyuncs.com）
	KeyOssEndpoint = "s3.oss.endpoint"
	// KeyOssAccessKeyID 阿里云 AccessKeyId
	KeyOssAccessKeyID = "s3.oss.access-key-id"
	// KeyOssAccessKeySecret 阿里云 AccessKeySecret（可用 ENC(...) 密文）
	KeyOssAccessKeySecret = "s3.oss.access-key-secret"
	// KeyOssBucket 存储桶
	KeyOssBucket = "s3.oss.bucket"
	// KeyOssDomain 自定义访问域名（CDN / 自有域名），为空时使用 endpoint + bucket
	KeyOssDomain = "s3.oss.domain"
)

// 文件存储错误（业务代码用 errors.Is 判断）
var (
	// ErrDisabled 文件存储未启用（s3.enabled=false）
	ErrDisabled = errors.New("文件存储未启用（s3.enabled / S3_ENABLED）")
	// ErrNotFound 文件不存在
	ErrNotFound = errors.New("文件不存在")
	// ErrUnsupported 当前后端不支持该操作（如本地目录无法生成签名 URL）
	ErrUnsupported = errors.New("当前文件存储后端不支持该操作")
	// ErrInvalidKey 非法的文件键（空、绝对路径、包含 .. 等）
	ErrInvalidKey = errors.New("非法的文件键")
)

// Object 存储对象元信息
type Object struct {
	Key         string    `json:"key"`         // 文件键（相对路径，如 2024/06/01/xxx.png）
	Size        int64     `json:"size"`        // 文件大小（字节）
	ContentType string    `json:"contentType"` // 内容类型
	ETag        string    `json:"etag"`        // 对象 ETag（本地目录为空）
	ModTime     time.Time `json:"modTime"`     // 最后修改时间
}

// Storage 文件存储接口：所有后端（本地目录 / MinIO / RustFS / COS / OSS）统一实现该接口。
//
// 约定：
//   - key 为相对路径（使用 / 分隔），由调用方保证唯一性（可用 GenerateKey 生成）；
//   - size < 0 表示长度未知（流式上传），COS 等后端会退化为分块上传；
//   - 文件不存在时返回包装了 ErrNotFound 的错误，能力不支持时返回包装了 ErrUnsupported 的错误。
type Storage interface {
	// Kind 后端类型（local / minio / rustfs / cos / oss）
	Kind() string
	// Put 上传（覆盖）对象，返回对象元信息
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error)
	// Get 读取对象内容，调用方负责 Close
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete 删除对象（不存在不报错）
	Delete(ctx context.Context, key string) error
	// Exists 判断对象是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// Stat 读取对象元信息
	Stat(ctx context.Context, key string) (*Object, error)
	// List 列出指定前缀下的对象（limit<=0 时使用默认上限）
	List(ctx context.Context, prefix string, limit int) ([]Object, error)
	// URL 返回对象的公开访问地址（不做签名，需桶/目录本身可公开访问）
	URL(key string) string
	// PresignGet 生成带签名的下载地址（本地目录后端返回 ErrUnsupported）
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
	// PresignPut 生成带签名的上传地址（本地目录后端返回 ErrUnsupported）
	PresignPut(ctx context.Context, key string, expiry time.Duration) (string, error)
	// Close 释放后端资源
	Close() error
}

// LocalOptions 本地目录后端配置（driver=local）
type LocalOptions struct {
	Dir       string // 本地目录（文件根目录，不存在会自动创建）
	URLPrefix string // 对外访问前缀（如 /uploads，配合静态文件路由或反向代理）
}

// S3Options S3 兼容后端配置（driver=minio / rustfs）
type S3Options struct {
	Endpoint  string // 服务地址，支持 host:port 或带 scheme 的 URL
	AccessKey string // Access Key（与 SecretKey 同时为空表示匿名访问）
	SecretKey string // Secret Key
	Bucket    string // 存储桶
	Region    string // 区域（留空由服务端决定）
	Secure    bool   // 是否使用 HTTPS（endpoint 带 scheme 时以 scheme 为准）
	PathStyle bool   // 是否强制路径风格寻址
}

// CosOptions 腾讯云 COS 配置（driver=cos）
type CosOptions struct {
	BucketURL string // 存储桶访问域名（优先）
	Bucket    string // 存储桶名（bucket-url 留空时与 Region 拼接访问域名）
	Region    string // 地域（如 ap-guangzhou）
	SecretID  string // SecretId
	SecretKey string // SecretKey
	Domain    string // 自定义访问域名（CDN）
}

// OssOptions 阿里云 OSS 配置（driver=oss）
type OssOptions struct {
	Endpoint        string // 访问域名（如 oss-cn-hangzhou.aliyuncs.com）
	AccessKeyID     string // AccessKeyId
	AccessKeySecret string // AccessKeySecret
	Bucket          string // 存储桶
	Domain          string // 自定义访问域名（CDN）
}

// Options 文件存储配置
type Options struct {
	Enabled       bool         // 是否启用（仅 New/Default 校验，Open 不校验）
	Driver        string       // 存储驱动
	PublicBaseURL string       // 对象访问地址的公共前缀
	Local         LocalOptions // 本地目录配置
	Minio         S3Options    // MinIO 配置
	RustFS        S3Options    // RustFS 配置
	Cos           CosOptions   // 腾讯云 COS 配置
	Oss           OssOptions   // 阿里云 OSS 配置
}

// LoadOptions 从配置文件（s3.*）读取文件存储配置
// @return: Options 文件存储配置
func LoadOptions() Options {
	return Options{
		Enabled:       config.Bool(KeyEnabled),
		Driver:        NormalizeDriver(config.String(KeyDriver)),
		PublicBaseURL: strings.TrimSpace(config.String(KeyPublicBaseURL)),
		Local: LocalOptions{
			Dir:       stringValue(KeyLocalDir, DefaultLocalDir),
			URLPrefix: stringValue(KeyLocalURLPrefix, DefaultLocalURLPrefix),
		},
		Minio: S3Options{
			Endpoint:  stringValue(KeyMinioEndpoint, DefaultMinioEndpoint),
			AccessKey: config.String(KeyMinioAccessKey),
			SecretKey: config.String(KeyMinioSecretKey),
			Bucket:    stringValue(KeyMinioBucket, DefaultMinioBucket),
			Region:    strings.TrimSpace(config.String(KeyMinioRegion)),
			Secure:    boolValue(KeyMinioSecure, false),
			PathStyle: boolValue(KeyMinioPathStyle, false),
		},
		RustFS: S3Options{
			Endpoint:  stringValue(KeyRustFSEndpoint, DefaultRustFSEndpoint),
			AccessKey: config.String(KeyRustFSAccessKey),
			SecretKey: config.String(KeyRustFSSecretKey),
			Bucket:    stringValue(KeyRustFSBucket, DefaultRustFSBucket),
			Region:    stringValue(KeyRustFSRegion, DefaultRustFSRegion),
			Secure:    boolValue(KeyRustFSSecure, false),
			PathStyle: boolValue(KeyRustFSPathStyle, true),
		},
		Cos: CosOptions{
			BucketURL: strings.TrimSpace(config.String(KeyCosBucketURL)),
			Bucket:    strings.TrimSpace(config.String(KeyCosBucket)),
			Region:    strings.TrimSpace(config.String(KeyCosRegion)),
			SecretID:  config.String(KeyCosSecretID),
			SecretKey: config.String(KeyCosSecretKey),
			Domain:    strings.TrimSpace(config.String(KeyCosDomain)),
		},
		Oss: OssOptions{
			Endpoint:        stringValue(KeyOssEndpoint, DefaultOssEndpoint),
			AccessKeyID:     config.String(KeyOssAccessKeyID),
			AccessKeySecret: config.String(KeyOssAccessKeySecret),
			Bucket:          stringValue(KeyOssBucket, DefaultOssBucket),
			Domain:          strings.TrimSpace(config.String(KeyOssDomain)),
		},
	}
}

// Open 按配置创建文件存储实例（不校验 s3.enabled，便于测试与自定义场景）
// @param: opts Options 文件存储配置
// @return: Storage, error
func Open(opts Options) (Storage, error) {
	switch NormalizeDriver(opts.Driver) {
	case DriverMinio:
		return newS3Compatible(DriverMinio, opts.Minio, opts.PublicBaseURL)
	case DriverRustFS:
		return newS3Compatible(DriverRustFS, opts.RustFS, opts.PublicBaseURL)
	case DriverCos:
		return newCos(opts)
	case DriverOss:
		return newOss(opts)
	default:
		return newLocal(opts)
	}
}

// New 从配置文件（s3.*）初始化文件存储实例
// @return: Storage, error s3.enabled=false 时返回 ErrDisabled
func New() (Storage, error) {
	opts := LoadOptions()
	if !opts.Enabled {
		return nil, ErrDisabled
	}
	st, err := Open(opts)
	if err != nil {
		return nil, err
	}
	logrus.Infof("文件存储已启用：driver=%s", st.Kind())
	return st, nil
}

var (
	defaultMu      sync.Mutex
	defaultStorage Storage
)

// Default 返回进程内共享的文件存储实例（懒加载，重复调用复用同一实例）
// @return: Storage, error
func Default() (Storage, error) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultStorage != nil {
		return defaultStorage, nil
	}
	st, err := New()
	if err != nil {
		return nil, err
	}
	defaultStorage = st
	return st, nil
}

// Reset 释放并清空共享实例（测试或运行期切换配置后重新加载用）
func Reset() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultStorage != nil {
		if err := defaultStorage.Close(); err != nil {
			logrus.Warnf("关闭文件存储失败: %s", err)
		}
	}
	defaultStorage = nil
}

// LocalDirectory 若后端为本地目录，返回磁盘根目录（便于注册静态文件路由）
// @param: st Storage 文件存储实例
// @return: string 目录, bool 是否为本地目录后端
func LocalDirectory(st Storage) (string, bool) {
	type localDir interface {
		LocalDir() string
	}
	if d, ok := st.(localDir); ok {
		return d.LocalDir(), true
	}
	return "", false
}

// NormalizeDriver 归一化存储驱动名（未知值回退为默认驱动并告警）
// @param: driver string 驱动名
// @return: string 归一化后的驱动名
func NormalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", DriverLocal, "file", "filesystem", "dir", "local-dir", "localfile":
		return DriverLocal
	case DriverMinio:
		return DriverMinio
	case DriverRustFS, "rust-fs", "rust_fs":
		return DriverRustFS
	case DriverCos, "tencent", "qcloud", "tencent-cos":
		return DriverCos
	case DriverOss, "aliyun", "alibaba", "ali-oss":
		return DriverOss
	default:
		logrus.Warnf("未知的文件存储驱动 %q，已回退为 %s（可选：local / minio / rustfs / cos / oss）", driver, DefaultDriver)
		return DefaultDriver
	}
}

// stringValue 读取字符串配置，空值使用默认值
func stringValue(key, def string) string {
	if value := strings.TrimSpace(config.String(key)); value != "" {
		return value
	}
	return def
}

// boolValue 读取布尔配置，未配置时使用默认值（区分「未配置」与「显式 false」）
func boolValue(key string, def bool) bool {
	if !config.IsSet(key) {
		return def
	}
	return config.Bool(key)
}

// pickContentType 推断内容类型：显式传入 > 扩展名推断 > 默认二进制流
func pickContentType(given, key string) string {
	if ct := strings.TrimSpace(given); ct != "" {
		return ct
	}
	return contentTypeByExtension(key)
}

// expiryOrDefault 签名有效期（<=0 使用默认值）
func expiryOrDefault(expiry time.Duration) time.Duration {
	if expiry <= 0 {
		return DefaultPresignExpiry
	}
	return expiry
}

// normalizeListLimit 列表条数上限归一化（单次最多 1000 条，与各对象存储上限一致）
func normalizeListLimit(limit int) int {
	if limit <= 0 {
		return DefaultListLimit
	}
	if limit > DefaultListLimit {
		return DefaultListLimit
	}
	return limit
}

// wrapNotFound 包装「文件不存在」错误，保留原始错误信息
func wrapNotFound(err error) error {
	if err == nil {
		return ErrNotFound
	}
	return fmt.Errorf("%w: %s", ErrNotFound, err.Error())
}

// cleanPrefix 归一化列举前缀（空表示列举全部，末尾不带 /）
func cleanPrefix(prefix string) string {
	prefix = strings.ReplaceAll(strings.TrimSpace(prefix), "\\", "/")
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return path.Clean(prefix)
}
