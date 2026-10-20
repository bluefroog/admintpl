package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// s3Compatible MinIO / RustFS 通用后端（两者均兼容 S3 协议，差别在默认寻址方式与默认 region）
type s3Compatible struct {
	driver  string // local / minio / rustfs
	client  *minio.Client
	bucket  string
	baseURL string // 对象访问地址前缀，空则使用 {scheme}://{endpoint}/{bucket}
	scheme  string // http / https
	host    string // 服务地址（host:port）
}

// newS3Compatible 创建 S3 兼容后端（MinIO / RustFS）
// @param: driver string 后端类型
// @param: opts S3Options S3 兼容配置
// @param: publicBaseURL string 对象访问地址公共前缀
// @return: Storage, error
func newS3Compatible(driver string, opts S3Options, publicBaseURL string) (Storage, error) {
	host, scheme, err := parseEndpoint(opts.Endpoint, opts.Secure)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", driver, err)
	}
	accessKey := strings.TrimSpace(opts.AccessKey)
	secretKey := strings.TrimSpace(opts.SecretKey)
	if (accessKey == "") != (secretKey == "") {
		return nil, fmt.Errorf("%s: access-key 与 secret-key 必须同时配置", driver)
	}
	bucket := strings.TrimSpace(opts.Bucket)
	if bucket == "" {
		bucket = DefaultMinioBucket
	}

	lookup := minio.BucketLookupAuto
	if opts.PathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       scheme == "https",
		Region:       strings.TrimSpace(opts.Region),
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: 初始化客户端失败（endpoint=%s）: %w", driver, host, err)
	}
	return &s3Compatible{
		driver:  driver,
		client:  client,
		bucket:  bucket,
		baseURL: strings.TrimSpace(publicBaseURL),
		scheme:  scheme,
		host:    host,
	}, nil
}

// Kind 后端类型
func (s *s3Compatible) Kind() string { return s.driver }

// Put 上传对象
func (s *s3Compatible) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("%s 上传失败: 内容为空（%s）", s.driver, clean)
	}
	ct := pickContentType(contentType, clean)
	info, err := s.client.PutObject(ctx, s.bucket, clean, reader, size, minio.PutObjectOptions{ContentType: ct})
	if err != nil {
		return nil, fmt.Errorf("%s 上传失败（%s）: %w", s.driver, clean, err)
	}
	return &Object{
		Key:         clean,
		Size:        info.Size,
		ContentType: ct,
		ETag:        info.ETag,
		ModTime:     info.LastModified,
	}, nil
}

// Get 读取对象内容
func (s *s3Compatible) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	object, err := s.client.GetObject(ctx, s.bucket, clean, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s 下载失败（%s）: %w", s.driver, clean, mapMinioError(err))
	}
	// GetObject 是延迟请求，Stat 才能真正确认对象是否存在
	if _, err = object.Stat(); err != nil {
		_ = object.Close()
		return nil, fmt.Errorf("%s 下载失败（%s）: %w", s.driver, clean, mapMinioError(err))
	}
	return object, nil
}

// Delete 删除对象（不存在不报错）
func (s *s3Compatible) Delete(ctx context.Context, key string) error {
	clean, err := NormalizeKey(key)
	if err != nil {
		return err
	}
	if err = s.client.RemoveObject(ctx, s.bucket, clean, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("%s 删除失败（%s）: %w", s.driver, clean, err)
	}
	return nil
}

// Exists 判断对象是否存在
func (s *s3Compatible) Exists(ctx context.Context, key string) (bool, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return false, err
	}
	_, err = s.client.StatObject(ctx, s.bucket, clean, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	if errors.Is(mapMinioError(err), ErrNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("%s 查询失败（%s）: %w", s.driver, clean, err)
}

// Stat 读取对象元信息
func (s *s3Compatible) Stat(ctx context.Context, key string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	info, err := s.client.StatObject(ctx, s.bucket, clean, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%s 查询失败（%s）: %w", s.driver, clean, mapMinioError(err))
	}
	return &Object{
		Key:         clean,
		Size:        info.Size,
		ContentType: pickContentType(info.ContentType, clean),
		ETag:        info.ETag,
		ModTime:     info.LastModified,
	}, nil
}

// List 列出指定前缀下的对象
func (s *s3Compatible) List(ctx context.Context, prefix string, limit int) ([]Object, error) {
	prefix = cleanPrefix(prefix)
	limit = normalizeListLimit(limit)

	// 提前结束遍历时取消内部协程，避免列表协程悬挂
	listCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	objects := make([]Object, 0, limit)
	for info := range s.client.ListObjects(listCtx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
		MaxKeys:   limit,
	}) {
		if info.Err != nil {
			return nil, fmt.Errorf("%s 列举失败（前缀 %q）: %w", s.driver, prefix, info.Err)
		}
		objects = append(objects, Object{
			Key:         info.Key,
			Size:        info.Size,
			ContentType: pickContentType(info.ContentType, info.Key),
			ETag:        info.ETag,
			ModTime:     info.LastModified,
		})
		if len(objects) >= limit {
			break
		}
	}
	return objects, nil
}

// URL 返回对象公开访问地址
func (s *s3Compatible) URL(key string) string {
	if s.baseURL != "" {
		return joinPublicURL(s.baseURL, key)
	}
	return joinPublicURL(fmt.Sprintf("%s://%s/%s", s.scheme, s.host, s.bucket), key)
}

// PresignGet 生成带签名的下载地址
func (s *s3Compatible) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return "", err
	}
	signed, err := s.client.PresignedGetObject(ctx, s.bucket, clean, expiryOrDefault(expiry), url.Values{})
	if err != nil {
		return "", fmt.Errorf("%s 生成下载签名失败（%s）: %w", s.driver, clean, err)
	}
	return signed.String(), nil
}

// PresignPut 生成带签名的上传地址
func (s *s3Compatible) PresignPut(ctx context.Context, key string, expiry time.Duration) (string, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return "", err
	}
	signed, err := s.client.PresignedPutObject(ctx, s.bucket, clean, expiryOrDefault(expiry))
	if err != nil {
		return "", fmt.Errorf("%s 生成上传签名失败（%s）: %w", s.driver, clean, err)
	}
	return signed.String(), nil
}

// Close S3 客户端无需显式关闭
func (s *s3Compatible) Close() error { return nil }

// parseEndpoint 解析服务地址，返回 host:port 与 scheme
// @param: endpoint string 地址（host:port 或带 scheme 的 URL）
// @param: secure bool 未指定 scheme 时是否使用 https
// @return: string host:port, string scheme, error
func parseEndpoint(endpoint string, secure bool) (string, string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", "", errors.New("未配置 endpoint")
	}
	scheme := "http"
	if secure {
		scheme = "https"
	}
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Host == "" {
			return "", "", fmt.Errorf("endpoint 非法（%s）", endpoint)
		}
		return parsed.Host, parsed.Scheme, nil
	}
	return strings.TrimSuffix(endpoint, "/"), scheme, nil
}

// mapMinioError 把 S3 错误码转换为统一错误（对象不存在 -> ErrNotFound）
func mapMinioError(err error) error {
	if err == nil {
		return nil
	}
	resp := minio.ToErrorResponse(err)
	switch resp.Code {
	case "NoSuchKey", "NoSuchObject", "NoSuchBucket", "NotFound":
		return wrapNotFound(err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return wrapNotFound(err)
	}
	return err
}
