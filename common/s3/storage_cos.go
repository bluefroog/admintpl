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

	"github.com/tencentyun/cos-go-sdk-v5"
)

// cosStorage 腾讯云 COS 后端
type cosStorage struct {
	client  *cos.Client
	baseURL string // 对象访问地址前缀（public-base-url > domain > bucket-url）
}

// newCos 创建腾讯云 COS 后端
// @param: opts Options 文件存储配置
// @return: Storage, error
func newCos(opts Options) (Storage, error) {
	bucketURL := strings.TrimSpace(opts.Cos.BucketURL)
	if bucketURL == "" {
		bucket := strings.TrimSpace(opts.Cos.Bucket)
		region := strings.TrimSpace(opts.Cos.Region)
		if bucket == "" || region == "" {
			return nil, errors.New("cos: 未配置 bucket-url（或同时配置 bucket 与 region）")
		}
		bucketURL = fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region)
	}
	secretID := strings.TrimSpace(opts.Cos.SecretID)
	secretKey := strings.TrimSpace(opts.Cos.SecretKey)
	if secretID == "" || secretKey == "" {
		return nil, errors.New("cos: 未配置 secret-id / secret-key")
	}

	parsed, err := url.Parse(bucketURL)
	if err != nil || parsed.Host == "" {
		return nil, fmt.Errorf("cos: bucket-url 非法（%q）", bucketURL)
	}
	client := cos.NewClient(&cos.BaseURL{BucketURL: parsed}, &http.Client{
		Timeout: defaultTimeoutSeconds * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
			Transport: &http.Transport{
				MaxIdleConns:    defaultMaxIdleConns,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	})

	baseURL := strings.TrimSpace(opts.PublicBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(opts.Cos.Domain)
	}
	if baseURL == "" {
		baseURL = strings.TrimSuffix(bucketURL, "/")
	} else if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + strings.TrimPrefix(baseURL, "//")
	}
	return &cosStorage{client: client, baseURL: strings.TrimSuffix(baseURL, "/")}, nil
}

// Kind 后端类型
func (c *cosStorage) Kind() string { return DriverCos }

// Put 上传对象
func (c *cosStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("cos 上传失败: 内容为空（%s）", clean)
	}
	ct := pickContentType(contentType, clean)
	// COS SDK 只识别已知长度的 reader；未知长度时使用分块上传
	var body io.Reader = reader
	if size >= 0 {
		body = fixedLengthReader{Reader: reader, size: size}
	}
	resp, err := c.client.Object.Put(ctx, clean, body, &cos.ObjectPutOptions{
		ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: ct},
	})
	if err != nil {
		return nil, fmt.Errorf("cos 上传失败（%s）: %w", clean, mapCosError(err))
	}
	etag := ""
	if resp != nil && resp.Header != nil {
		etag = strings.Trim(resp.Header.Get("ETag"), `"`)
	}
	return &Object{Key: clean, Size: size, ContentType: ct, ETag: etag, ModTime: time.Now()}, nil
}

// Get 读取对象内容
func (c *cosStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Object.Get(ctx, clean, nil)
	if err != nil {
		return nil, fmt.Errorf("cos 下载失败（%s）: %w", clean, mapCosError(err))
	}
	return resp.Body, nil
}

// Delete 删除对象（不存在不报错）
func (c *cosStorage) Delete(ctx context.Context, key string) error {
	clean, err := NormalizeKey(key)
	if err != nil {
		return err
	}
	if _, err = c.client.Object.Delete(ctx, clean); err != nil {
		return fmt.Errorf("cos 删除失败（%s）: %w", clean, mapCosError(err))
	}
	return nil
}

// Exists 判断对象是否存在
func (c *cosStorage) Exists(ctx context.Context, key string) (bool, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return false, err
	}
	_, err = c.client.Object.Head(ctx, clean, nil)
	if err == nil {
		return true, nil
	}
	if errors.Is(mapCosError(err), ErrNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("cos 查询失败（%s）: %w", clean, err)
}

// Stat 读取对象元信息
func (c *cosStorage) Stat(ctx context.Context, key string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Object.Head(ctx, clean, nil)
	if err != nil {
		return nil, fmt.Errorf("cos 查询失败（%s）: %w", clean, mapCosError(err))
	}
	return objectFromHeader(clean, resp.Header, 0), nil
}

// List 列出指定前缀下的对象
func (c *cosStorage) List(ctx context.Context, prefix string, limit int) ([]Object, error) {
	prefix = cleanPrefix(prefix)
	limit = normalizeListLimit(limit)
	result, _, err := c.client.Bucket.Get(ctx, &cos.BucketGetOptions{Prefix: prefix, MaxKeys: limit})
	if err != nil {
		return nil, fmt.Errorf("cos 列举失败（前缀 %q）: %w", prefix, mapCosError(err))
	}
	objects := make([]Object, 0, len(result.Contents))
	for _, item := range result.Contents {
		object := Object{
			Key:         item.Key,
			Size:        item.Size,
			ContentType: contentTypeByExtension(item.Key),
			ETag:        strings.Trim(item.ETag, `"`),
		}
		if item.LastModified != "" {
			if modTime, err := time.Parse(time.RFC3339, item.LastModified); err == nil {
				object.ModTime = modTime
			}
		}
		objects = append(objects, object)
		if len(objects) >= limit {
			break
		}
	}
	return objects, nil
}

// URL 返回对象公开访问地址
func (c *cosStorage) URL(key string) string { return joinPublicURL(c.baseURL, key) }

// PresignGet 生成带签名的下载地址
func (c *cosStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return "", err
	}
	signed, err := c.client.Object.GetPresignedURL2(ctx, http.MethodGet, clean, expiryOrDefault(expiry), nil)
	if err != nil {
		return "", fmt.Errorf("cos 生成下载签名失败（%s）: %w", clean, err)
	}
	return signed.String(), nil
}

// PresignPut 生成带签名的上传地址
func (c *cosStorage) PresignPut(ctx context.Context, key string, expiry time.Duration) (string, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return "", err
	}
	signed, err := c.client.Object.GetPresignedURL2(ctx, http.MethodPut, clean, expiryOrDefault(expiry), nil)
	if err != nil {
		return "", fmt.Errorf("cos 生成上传签名失败（%s）: %w", clean, err)
	}
	return signed.String(), nil
}

// Close COS 客户端无需显式关闭
func (c *cosStorage) Close() error { return nil }

// fixedLengthReader 为未知类型的 reader 补充长度信息（COS SDK 依赖 FixedLengthReader 判断长度）
type fixedLengthReader struct {
	io.Reader
	size int64
}

// Size 返回内容长度
func (r fixedLengthReader) Size() int64 { return r.size }

// mapCosError 把 COS 错误转换为统一错误（对象不存在 -> ErrNotFound）
func mapCosError(err error) error {
	if err == nil {
		return nil
	}
	var resp *cos.ErrorResponse
	if errors.As(err, &resp) {
		if resp.Code == "NoSuchResource" || resp.Code == "NoSuchKey" {
			return wrapNotFound(err)
		}
		if resp.Response != nil && resp.Response.StatusCode == http.StatusNotFound {
			return wrapNotFound(err)
		}
	}
	return err
}
