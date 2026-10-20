package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNormalizeKey(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"普通键", "a/b.png", "a/b.png", false},
		{"去掉首部斜杠", "/a/b.png", "a/b.png", false},
		{"反斜杠转斜杠", `a\b.png`, "a/b.png", false},
		{"清理多余分隔符", "a//b/./c.png", "a/b/c.png", false},
		{"空键", "  ", "", true},
		{"仅斜杠", "///", "", true},
		{"存在穿越", "../etc/passwd", "", true},
		{"中间穿越", "a/../../b", "", true},
		{"点目录", ".", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeKey(c.in)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidKey) {
					t.Fatalf("NormalizeKey(%q) 应返回 ErrInvalidKey，实际 err=%v got=%q", c.in, err, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeKey(%q) 不应报错: %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("NormalizeKey(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestGenerateKey(t *testing.T) {
	pattern := regexp.MustCompile(`^\d{4}/\d{2}/\d{2}/[0-9a-f]{32}\.png$`)
	key := GenerateKey("头像.PNG")
	if !pattern.MatchString(key) {
		t.Fatalf("GenerateKey 结果格式不正确: %q", key)
	}
	if GenerateKey("avatar.png") == key {
		t.Fatal("GenerateKey 每次调用应生成不同键")
	}
	// 非法扩展名一律丢弃，避免生成奇怪的对象键
	for _, name := range []string{"a.tar.gz?x=1", "noext", "../../evil.png/../../"} {
		got := GenerateKey(name)
		if strings.Contains(got, "..") {
			t.Fatalf("GenerateKey(%q) 生成了不安全的键: %q", name, got)
		}
		if !regexp.MustCompile(`^\d{4}/\d{2}/\d{2}/[0-9a-f]{32}(\.[a-z0-9]{1,10})?$`).MatchString(got) {
			t.Fatalf("GenerateKey(%q) 结果格式不正确: %q", name, got)
		}
	}
}

func TestJoinPublicURL(t *testing.T) {
	cases := []struct {
		base string
		key  string
		want string
	}{
		{"/uploads", "2024/06/a b.png", "/uploads/2024/06/a%20b.png"},
		{"https://cdn.example.com/", "a/b.png", "https://cdn.example.com/a/b.png"},
		{"", "a/b.png", "/a/b.png"},
		{"/uploads", "../evil", ""},
	}
	for _, c := range cases {
		if got := joinPublicURL(c.base, c.key); got != c.want {
			t.Fatalf("joinPublicURL(%q, %q) = %q, want %q", c.base, c.key, got, c.want)
		}
	}
}

func newTestLocal(t *testing.T) Storage {
	t.Helper()
	st, err := Open(Options{Driver: DriverLocal, Local: LocalOptions{Dir: t.TempDir()}})
	if err != nil {
		t.Fatalf("初始化本地存储失败: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestLocalStorageCRUD(t *testing.T) {
	ctx := context.Background()
	st := newTestLocal(t)
	if st.Kind() != DriverLocal {
		t.Fatalf("Kind() = %q, want %q", st.Kind(), DriverLocal)
	}

	content := []byte("hello admintpl")
	obj, err := st.Put(ctx, "/docs/a.txt", bytes.NewReader(content), int64(len(content)), "text/plain")
	if err != nil {
		t.Fatalf("Put 失败: %v", err)
	}
	if obj.Key != "docs/a.txt" || obj.Size != int64(len(content)) || obj.ContentType != "text/plain" {
		t.Fatalf("Put 返回元信息不正确: %+v", obj)
	}

	exist, err := st.Exists(ctx, "docs/a.txt")
	if err != nil || !exist {
		t.Fatalf("Exists = %v, err = %v，期望 true", exist, err)
	}

	stat, err := st.Stat(ctx, "docs/a.txt")
	if err != nil {
		t.Fatalf("Stat 失败: %v", err)
	}
	if stat.Size != int64(len(content)) {
		t.Fatalf("Stat.Size = %d, want %d", stat.Size, len(content))
	}

	body, err := st.Get(ctx, "docs/a.txt")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	got, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("Get 内容不正确: %q, err = %v", got, err)
	}

	list, err := st.List(ctx, "docs", 10)
	if err != nil || len(list) != 1 || list[0].Key != "docs/a.txt" {
		t.Fatalf("List = %+v, err = %v", list, err)
	}

	if url := st.URL("docs/a.txt"); url != "/uploads/docs/a.txt" {
		t.Fatalf("URL = %q, want %q", url, "/uploads/docs/a.txt")
	}

	if _, err = st.PresignGet(ctx, "docs/a.txt", time.Minute); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("本地目录 PresignGet 应返回 ErrUnsupported，实际 %v", err)
	}
	if _, err = st.PresignPut(ctx, "docs/a.txt", time.Minute); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("本地目录 PresignPut 应返回 ErrUnsupported，实际 %v", err)
	}

	if err = st.Delete(ctx, "docs/a.txt"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	// 删除不存在的文件不报错
	if err = st.Delete(ctx, "docs/a.txt"); err != nil {
		t.Fatalf("重复 Delete 不应报错: %v", err)
	}
	if _, err = st.Stat(ctx, "docs/a.txt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("删除后 Stat 应返回 ErrNotFound，实际 %v", err)
	}
	if _, err = st.Get(ctx, "docs/a.txt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("删除后 Get 应返回 ErrNotFound，实际 %v", err)
	}
	if exist, err = st.Exists(ctx, "docs/a.txt"); err != nil || exist {
		t.Fatalf("删除后 Exists = %v, err = %v，期望 false", exist, err)
	}
}

func TestLocalStorageSizeMismatch(t *testing.T) {
	st := newTestLocal(t)
	if _, err := st.Put(context.Background(), "a.txt", strings.NewReader("abc"), 10, ""); err == nil {
		t.Fatal("声明大小与实际不一致时应报错")
	}
}

func TestLocalStorageRejectsEscape(t *testing.T) {
	st := newTestLocal(t)
	if _, err := st.Put(context.Background(), "../evil.txt", strings.NewReader("x"), 1, ""); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("非法键应返回 ErrInvalidKey，实际 %v", err)
	}
}

func TestLocalDirectory(t *testing.T) {
	st := newTestLocal(t)
	dir, ok := LocalDirectory(st)
	if !ok || dir == "" {
		t.Fatalf("LocalDirectory = %q, %v，期望返回本地目录", dir, ok)
	}
	if _, ok = LocalDirectory(&fakeStorage{}); ok {
		t.Fatal("非本地后端不应返回目录")
	}
}

func TestLocalPublicBaseURLOverride(t *testing.T) {
	st, err := Open(Options{
		Driver:        DriverLocal,
		PublicBaseURL: "https://cdn.example.com/static",
		Local:         LocalOptions{Dir: t.TempDir()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if url := st.URL("a/b.png"); url != "https://cdn.example.com/static/a/b.png" {
		t.Fatalf("URL = %q", url)
	}
}

func TestNormalizeDriver(t *testing.T) {
	cases := map[string]string{
		"":          DriverLocal,
		"local":     DriverLocal,
		"FILE":      DriverLocal,
		"minio":     DriverMinio,
		"rustfs":    DriverRustFS,
		"rust-fs":   DriverRustFS,
		"cos":       DriverCos,
		"qcloud":    DriverCos,
		"oss":       DriverOss,
		"aliyun":    DriverOss,
		"unknown-x": DriverLocal,
	}
	for in, want := range cases {
		if got := NormalizeDriver(in); got != want {
			t.Fatalf("NormalizeDriver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOpenValidatesDriverConfig(t *testing.T) {
	cases := []struct {
		name   string
		opts   Options
		wantIn string
	}{
		{
			name:   "minio 只配了 access-key",
			opts:   Options{Driver: DriverMinio, Minio: S3Options{Endpoint: "127.0.0.1:9000", AccessKey: "ak"}},
			wantIn: "access-key 与 secret-key",
		},
		{
			name:   "minio 未配置 endpoint",
			opts:   Options{Driver: DriverMinio, Minio: S3Options{}},
			wantIn: "endpoint",
		},
		{
			name:   "cos 缺少密钥",
			opts:   Options{Driver: DriverCos, Cos: CosOptions{BucketURL: "https://demo-125.cos.ap-guangzhou.myqcloud.com"}},
			wantIn: "secret-id",
		},
		{
			name:   "cos 缺少访问域名",
			opts:   Options{Driver: DriverCos, Cos: CosOptions{SecretID: "id", SecretKey: "key"}},
			wantIn: "bucket-url",
		},
		{
			name:   "oss 缺少密钥",
			opts:   Options{Driver: DriverOss, Oss: OssOptions{Endpoint: "oss-cn-hangzhou.aliyuncs.com"}},
			wantIn: "access-key-id",
		},
		{
			name:   "oss endpoint 非法",
			opts:   Options{Driver: DriverOss, Oss: OssOptions{Endpoint: "", AccessKeyID: "id", AccessKeySecret: "key"}},
			wantIn: "endpoint",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st, err := Open(c.opts)
			if err == nil {
				_ = st.Close()
				t.Fatalf("期望报错（包含 %q），实际初始化成功", c.wantIn)
			}
			if !strings.Contains(err.Error(), c.wantIn) {
				t.Fatalf("错误信息应包含 %q，实际 %v", c.wantIn, err)
			}
		})
	}
}

func TestOpenS3Compatible(t *testing.T) {
	opts := Options{
		Driver: DriverRustFS,
		RustFS: S3Options{
			Endpoint:  "http://127.0.0.1:9000",
			AccessKey: "ak",
			SecretKey: "sk",
			Bucket:    "files",
			Region:    DefaultRustFSRegion,
			PathStyle: true,
		},
	}
	st, err := Open(opts)
	if err != nil {
		t.Fatalf("初始化 RustFS 后端失败: %v", err)
	}
	if st.Kind() != DriverRustFS {
		t.Fatalf("Kind = %q", st.Kind())
	}
	if url := st.URL("a/b.png"); url != "http://127.0.0.1:9000/files/a/b.png" {
		t.Fatalf("URL = %q", url)
	}

	ctx := context.Background()
	signed, err := st.PresignGet(ctx, "a/b.png", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet 失败: %v", err)
	}
	if !strings.Contains(signed, "X-Amz-Signature") || !strings.Contains(signed, "/files/a/b.png") {
		t.Fatalf("签名地址不正确: %s", signed)
	}
	signedPut, err := st.PresignPut(ctx, "a/b.png", time.Minute)
	if err != nil || signedPut == "" {
		t.Fatalf("PresignPut 失败: %v", err)
	}
}

func TestOpenOss(t *testing.T) {
	st, err := Open(Options{
		Driver: DriverOss,
		Oss: OssOptions{
			Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
			AccessKeyID:     "id",
			AccessKeySecret: "secret",
			Bucket:          "files",
		},
	})
	if err != nil {
		t.Fatalf("初始化 OSS 后端失败: %v", err)
	}
	if st.Kind() != DriverOss {
		t.Fatalf("Kind = %q", st.Kind())
	}
	if url := st.URL("a/b.png"); url != "https://files.oss-cn-hangzhou.aliyuncs.com/a/b.png" {
		t.Fatalf("URL = %q", url)
	}
	signed, err := st.PresignGet(context.Background(), "a/b.png", time.Minute)
	if err != nil {
		t.Fatalf("PresignGet 失败: %v", err)
	}
	if !strings.Contains(signed, "Signature=") {
		t.Fatalf("签名地址不正确: %s", signed)
	}
}

func TestOpenCos(t *testing.T) {
	st, err := Open(Options{
		Driver: DriverCos,
		Cos: CosOptions{
			Bucket:    "demo-1250000000",
			Region:    "ap-guangzhou",
			SecretID:  "id",
			SecretKey: "secret",
		},
	})
	if err != nil {
		t.Fatalf("初始化 COS 后端失败: %v", err)
	}
	if st.Kind() != DriverCos {
		t.Fatalf("Kind = %q", st.Kind())
	}
	want := "https://demo-1250000000.cos.ap-guangzhou.myqcloud.com/a/b.png"
	if url := st.URL("a/b.png"); url != want {
		t.Fatalf("URL = %q, want %q", url, want)
	}
}

func TestLoadOptionsDefaults(t *testing.T) {
	// 测试环境下没有 config/config.yaml，读取到的应是全部默认值
	opts := LoadOptions()
	if opts.Driver != DriverLocal {
		t.Fatalf("默认 driver = %q, want %q", opts.Driver, DriverLocal)
	}
	if opts.Local.Dir != DefaultLocalDir || opts.Local.URLPrefix != DefaultLocalURLPrefix {
		t.Fatalf("本地目录默认值不正确: %+v", opts.Local)
	}
	if opts.Minio.Endpoint != DefaultMinioEndpoint || opts.Minio.Bucket != DefaultMinioBucket {
		t.Fatalf("MinIO 默认值不正确: %+v", opts.Minio)
	}
	if opts.RustFS.Region != DefaultRustFSRegion || !opts.RustFS.PathStyle {
		t.Fatalf("RustFS 默认值不正确: %+v", opts.RustFS)
	}
	if opts.Oss.Endpoint != DefaultOssEndpoint || opts.Oss.Bucket != DefaultOssBucket {
		t.Fatalf("OSS 默认值不正确: %+v", opts.Oss)
	}
}

func TestNewDisabled(t *testing.T) {
	st, err := New()
	if !errors.Is(err, ErrDisabled) {
		if st != nil {
			_ = st.Close()
		}
		t.Fatalf("s3.enabled=false 时 New() 应返回 ErrDisabled，实际 %v", err)
	}
}

func TestObjectFromHeader(t *testing.T) {
	header := http.Header{}
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", "12")
	header.Set("ETag", `"abc"`)
	header.Set("Last-Modified", "Wed, 01 May 2024 10:00:00 GMT")
	object := objectFromHeader("a/b.txt", header, 0)
	if object.Size != 12 || object.ContentType != "text/plain" || object.ETag != "abc" {
		t.Fatalf("解析结果不正确: %+v", object)
	}
	if object.ModTime.IsZero() {
		t.Fatal("Last-Modified 未解析")
	}
}

func TestExpiryOrDefault(t *testing.T) {
	if expiryOrDefault(0) != DefaultPresignExpiry {
		t.Fatal("expiry<=0 应使用默认有效期")
	}
	if expiryOrDefault(time.Minute) != time.Minute {
		t.Fatal("显式有效期不应被覆盖")
	}
}

// fakeStorage 用于验证非本地后端不返回目录
type fakeStorage struct{}

func (f *fakeStorage) Kind() string { return DriverMinio }
func (f *fakeStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error) {
	return nil, ErrUnsupported
}
func (f *fakeStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, ErrUnsupported
}
func (f *fakeStorage) Delete(ctx context.Context, key string) error { return ErrUnsupported }
func (f *fakeStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, ErrUnsupported
}
func (f *fakeStorage) Stat(ctx context.Context, key string) (*Object, error) {
	return nil, ErrUnsupported
}
func (f *fakeStorage) List(ctx context.Context, prefix string, limit int) ([]Object, error) {
	return nil, ErrUnsupported
}
func (f *fakeStorage) URL(key string) string { return "" }
func (f *fakeStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", ErrUnsupported
}
func (f *fakeStorage) PresignPut(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", ErrUnsupported
}
func (f *fakeStorage) Close() error { return nil }
