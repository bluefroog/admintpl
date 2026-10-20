package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// ossStorage 阿里云 OSS 后端
type ossStorage struct {
	client  *oss.Client
	bucket  *oss.Bucket
	name    string // 存储桶名
	baseURL string // 对象访问地址前缀（public-base-url > domain > endpoint/bucket）
}

// newOss 创建阿里云 OSS 后端
// @param: opts Options 文件存储配置
// @return: Storage, error
func newOss(opts Options) (Storage, error) {
	endpoint := strings.TrimSpace(opts.Oss.Endpoint)
	if endpoint == "" {
		return nil, errors.New("oss: 未配置 endpoint")
	}
	accessKeyID := strings.TrimSpace(opts.Oss.AccessKeyID)
	accessKeySecret := strings.TrimSpace(opts.Oss.AccessKeySecret)
	if accessKeyID == "" || accessKeySecret == "" {
		return nil, errors.New("oss: 未配置 access-key-id / access-key-secret")
	}
	bucketName := stringDefault(opts.Oss.Bucket, DefaultOssBucket)

	client, err := oss.New(normalizeOssEndpoint(endpoint), accessKeyID, accessKeySecret,
		oss.Timeout(int64(defaultTimeoutSeconds), int64(defaultTimeoutSeconds*3)),
	)
	if err != nil {
		return nil, fmt.Errorf("oss: 初始化客户端失败（endpoint=%s）: %w", endpoint, err)
	}
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("oss: 获取存储桶失败（%s）: %w", bucketName, err)
	}

	baseURL := strings.TrimSpace(opts.PublicBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(opts.Oss.Domain)
	}
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.%s", bucketName, strings.TrimSuffix(normalizeOssEndpoint(endpoint), "/"))
	} else if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + strings.TrimPrefix(baseURL, "//")
	}
	return &ossStorage{client: client, bucket: bucket, name: bucketName, baseURL: strings.TrimSuffix(baseURL, "/")}, nil
}

// Kind 后端类型
func (o *ossStorage) Kind() string { return DriverOss }

// Put 上传对象
func (o *ossStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("oss 上传失败: 内容为空（%s）", clean)
	}
	ct := pickContentType(contentType, clean)
	if err = o.bucket.PutObject(clean, reader, oss.ContentType(ct)); err != nil {
		return nil, fmt.Errorf("oss 上传失败（%s）: %w", clean, mapOssError(err))
	}
	return &Object{Key: clean, Size: size, ContentType: ct, ModTime: time.Now()}, nil
}

// Get 读取对象内容
func (o *ossStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	body, err := o.bucket.GetObject(clean)
	if err != nil {
		return nil, fmt.Errorf("oss 下载失败（%s）: %w", clean, mapOssError(err))
	}
	return body, nil
}

// Delete 删除对象（不存在不报错）
func (o *ossStorage) Delete(ctx context.Context, key string) error {
	clean, err := NormalizeKey(key)
	if err != nil {
		return err
	}
	if err = o.bucket.DeleteObject(clean); err != nil {
		if errors.Is(mapOssError(err), ErrNotFound) {
			return nil
		}
		return fmt.Errorf("oss 删除失败（%s）: %w", clean, mapOssError(err))
	}
	return nil
}

// Exists 判断对象是否存在
func (o *ossStorage) Exists(ctx context.Context, key string) (bool, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return false, err
	}
	exist, err := o.bucket.IsObjectExist(clean)
	if err != nil {
		return false, fmt.Errorf("oss 查询失败（%s）: %w", clean, mapOssError(err))
	}
	return exist, nil
}

// Stat 读取对象元信息
func (o *ossStorage) Stat(ctx context.Context, key string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	header, err := o.bucket.GetObjectDetailedMeta(clean)
	if err != nil {
		return nil, fmt.Errorf("oss 查询失败（%s）: %w", clean, mapOssError(err))
	}
	return objectFromHeader(clean, header, 0), nil
}

// List 列出指定前缀下的对象
func (o *ossStorage) List(ctx context.Context, prefix string, limit int) ([]Object, error) {
	prefix = cleanPrefix(prefix)
	limit = normalizeListLimit(limit)
	result, err := o.bucket.ListObjects(oss.Prefix(prefix), oss.MaxKeys(limit))
	if err != nil {
		return nil, fmt.Errorf("oss 列举失败（前缀 %q）: %w", prefix, mapOssError(err))
	}
	objects := make([]Object, 0, len(result.Objects))
	for _, item := range result.Objects {
		objects = append(objects, Object{
			Key:         item.Key,
			Size:        item.Size,
			ContentType: contentTypeByExtension(item.Key),
			ETag:        strings.Trim(item.ETag, `"`),
			ModTime:     item.LastModified,
		})
		if len(objects) >= limit {
			break
		}
	}
	return objects, nil
}

// URL 返回对象公开访问地址
func (o *ossStorage) URL(key string) string { return joinPublicURL(o.baseURL, key) }

// PresignGet 生成带签名的下载地址
func (o *ossStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return "", err
	}
	signed, err := o.bucket.SignURL(clean, oss.HTTPGet, int64(expiryOrDefault(expiry).Seconds()))
	if err != nil {
		return "", fmt.Errorf("oss 生成下载签名失败（%s）: %w", clean, err)
	}
	return signed, nil
}

// PresignPut 生成带签名的上传地址
func (o *ossStorage) PresignPut(ctx context.Context, key string, expiry time.Duration) (string, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return "", err
	}
	signed, err := o.bucket.SignURL(clean, oss.HTTPPut, int64(expiryOrDefault(expiry).Seconds()))
	if err != nil {
		return "", fmt.Errorf("oss 生成上传签名失败（%s）: %w", clean, err)
	}
	return signed, nil
}

// Close OSS 客户端无需显式关闭
func (o *ossStorage) Close() error { return nil }

// normalizeOssEndpoint endpoint 只保留 host:port / 域名（支持误填 https:// 前缀）
func normalizeOssEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if strings.Contains(endpoint, "://") {
		if idx := strings.Index(endpoint, "://"); idx >= 0 {
			endpoint = endpoint[idx+3:]
		}
	}
	return strings.TrimSuffix(endpoint, "/")
}

// stringDefault 字符串配置归一化（空值使用默认值）
func stringDefault(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return strings.TrimSpace(value)
}

// objectFromHeader 从 HTTP 响应头解析对象元信息（COS / OSS 的 Head 结果）
func objectFromHeader(key string, header http.Header, fallbackSize int64) *Object {
	object := &Object{
		Key:         key,
		Size:        fallbackSize,
		ContentType: pickContentType(header.Get("Content-Type"), key),
		ETag:        strings.Trim(header.Get("ETag"), `"`),
	}
	if value := header.Get("Content-Length"); value != "" {
		if size, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
			object.Size = size
		}
	}
	if value := header.Get("Last-Modified"); value != "" {
		if modTime, err := http.ParseTime(value); err == nil {
			object.ModTime = modTime
		}
	}
	return object
}

// mapOssError 把 OSS 错误转换为统一错误（对象不存在 -> ErrNotFound）
func mapOssError(err error) error {
	if err == nil {
		return nil
	}
	var serviceErr oss.ServiceError
	if errors.As(err, &serviceErr) && serviceErr.StatusCode == http.StatusNotFound {
		return wrapNotFound(err)
	}
	var serviceErrPtr *oss.ServiceError
	if errors.As(err, &serviceErrPtr) && serviceErrPtr != nil && serviceErrPtr.StatusCode == http.StatusNotFound {
		return wrapNotFound(err)
	}
	return err
}
