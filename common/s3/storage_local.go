package s3

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// localStorage 本地目录后端：对象键直接映射为本地目录下的文件路径
//
// 说明：
//   - 目录由 s3.local.dir 指定，不存在会自动创建；
//   - 对外访问地址 = s3.public-base-url（优先）或 s3.local.url-prefix + /文件键，
//     需要由静态文件路由（如 gin 的 router.Static）或反向代理（nginx）把该前缀映射到目录；
//   - 写入先落临时文件再 rename，避免并发读到半截文件；
//   - 本地目录没有签名机制，PresignPut/PresignGet 返回 ErrUnsupported，直接使用 URL() 即可。
type localStorage struct {
	dir     string // 文件根目录（绝对路径）
	baseURL string // 对外访问前缀
}

// newLocal 创建本地目录后端
// @param: opts Options 文件存储配置
// @return: Storage, error
func newLocal(opts Options) (Storage, error) {
	dir := strings.TrimSpace(opts.Local.Dir)
	if dir == "" {
		dir = DefaultLocalDir
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("本地存储目录非法（%s）: %w", dir, err)
	}
	if err = os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("创建本地存储目录失败（%s）: %w", abs, err)
	}

	baseURL := strings.TrimSpace(opts.PublicBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(opts.Local.URLPrefix)
		if baseURL == "" {
			baseURL = DefaultLocalURLPrefix
		}
		if !strings.HasPrefix(baseURL, "/") {
			baseURL = "/" + baseURL
		}
	}
	return &localStorage{dir: abs, baseURL: baseURL}, nil
}

// Kind 后端类型
func (l *localStorage) Kind() string { return DriverLocal }

// LocalDir 本地磁盘根目录（对象键相对于该目录）
func (l *localStorage) LocalDir() string { return l.dir }

// Put 写入对象
func (l *localStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, fmt.Errorf("本地存储写入失败: 内容为空（%s）", clean)
	}
	full, err := l.filePath(clean)
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	dir := filepath.Dir(full)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录失败（%s）: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	written, err := io.Copy(tmp, reader)
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil && size >= 0 && written != size {
		err = fmt.Errorf("写入字节数与声明大小不一致（声明 %d，实际 %d）", size, written)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("写入文件失败（%s）: %w", clean, err)
	}
	if err = os.Rename(tmp.Name(), full); err != nil {
		_ = os.Remove(tmp.Name())
		return nil, fmt.Errorf("保存文件失败（%s）: %w", clean, err)
	}

	info, err := os.Stat(full)
	if err != nil {
		return nil, fmt.Errorf("读取文件信息失败（%s）: %w", clean, err)
	}
	return &Object{
		Key:         clean,
		Size:        info.Size(),
		ContentType: pickContentType(contentType, clean),
		ModTime:     info.ModTime(),
	}, nil
}

// Get 读取对象内容
func (l *localStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	full, err := l.filePath(clean)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, wrapNotFound(err)
		}
		return nil, fmt.Errorf("读取文件失败（%s）: %w", clean, err)
	}
	return file, nil
}

// Delete 删除对象（不存在不报错）
func (l *localStorage) Delete(ctx context.Context, key string) error {
	clean, err := NormalizeKey(key)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	full, err := l.filePath(clean)
	if err != nil {
		return err
	}
	if err = os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败（%s）: %w", clean, err)
	}
	return nil
}

// Exists 判断对象是否存在
func (l *localStorage) Exists(ctx context.Context, key string) (bool, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return false, err
	}
	if err = ctx.Err(); err != nil {
		return false, err
	}
	full, err := l.filePath(clean)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("读取文件信息失败（%s）: %w", clean, err)
	}
	return !info.IsDir(), nil
}

// Stat 读取对象元信息
func (l *localStorage) Stat(ctx context.Context, key string) (*Object, error) {
	clean, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	full, err := l.filePath(clean)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, wrapNotFound(err)
		}
		return nil, fmt.Errorf("读取文件信息失败（%s）: %w", clean, err)
	}
	if info.IsDir() {
		return nil, wrapNotFound(fmt.Errorf("%s 是目录", clean))
	}
	return &Object{
		Key:         clean,
		Size:        info.Size(),
		ContentType: contentTypeByExtension(clean),
		ModTime:     info.ModTime(),
	}, nil
}

// List 列出指定前缀下的对象
func (l *localStorage) List(ctx context.Context, prefix string, limit int) ([]Object, error) {
	prefix = cleanPrefix(prefix)
	limit = normalizeListLimit(limit)
	objects := make([]Object, 0, limit)
	err := filepath.WalkDir(l.dir, func(fullPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(l.dir, fullPath)
		if err != nil {
			return err
		}
		// 跳过上传过程中的临时文件（正常流程会被 rename，残留的临时文件不应出现在列表中）
		if strings.HasPrefix(filepath.Base(rel), ".upload-") {
			return nil
		}
		clean, err := NormalizeKey(rel)
		if err != nil {
			return nil
		}
		if prefix != "" && !strings.HasPrefix(clean, prefix) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		objects = append(objects, Object{
			Key:         clean,
			Size:        info.Size(),
			ContentType: contentTypeByExtension(clean),
			ModTime:     info.ModTime(),
		})
		if len(objects) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("列举文件失败（前缀 %q）: %w", prefix, err)
	}
	return objects, nil
}

// URL 返回对象公开访问地址
func (l *localStorage) URL(key string) string { return joinPublicURL(l.baseURL, key) }

// PresignGet 本地目录不支持签名 URL
func (l *localStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", fmt.Errorf("%w: 本地目录存储请直接使用 URL()", ErrUnsupported)
}

// PresignPut 本地目录不支持签名 URL
func (l *localStorage) PresignPut(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return "", fmt.Errorf("%w: 本地目录存储请直接使用 Put()", ErrUnsupported)
}

// Close 本地目录无外部连接需要释放
func (l *localStorage) Close() error { return nil }

// filePath 拼接对象键对应的本地文件路径，并校验不会逃出根目录
func (l *localStorage) filePath(key string) (string, error) {
	full := filepath.Join(l.dir, filepath.FromSlash(key))
	rel, err := filepath.Rel(l.dir, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %s", ErrInvalidKey, key)
	}
	return full, nil
}
