package provider

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"
)

type localFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (l *localFileInfo) Name() string       { return l.name }
func (l *localFileInfo) Size() int64        { return l.size }
func (l *localFileInfo) IsDir() bool        { return l.isDir }
func (l *localFileInfo) ModTime() time.Time { return l.modTime }

// uid/gid/mode methods for basic localFileInfo so it trivially satisfies UnixFileInfo if needed,
// but usually we'll return a unixFileInfo.
func (l *localFileInfo) UID() uint32       { return 0 }
func (l *localFileInfo) GID() uint32       { return 0 }
func (l *localFileInfo) Mode() os.FileMode { return 0 }

// LocalProvider implements the Provider interface for posix-compliant local filesystems.
type LocalProvider struct {
	basePath string
	mapper   *MetadataMapper
}

// NewLocalProvider creates a new LocalProvider rooted at basePath.
// If basePath is empty, it acts upon absolute or relative paths directly.
func NewLocalProvider(basePath string) *LocalProvider {
	return &LocalProvider{
		basePath: basePath,
		mapper:   NewMetadataMapper(), // default empty mapper
	}
}

// WithMetadataMapper adds a metadata mapper to the provider
func (p *LocalProvider) WithMetadataMapper(mapper *MetadataMapper) *LocalProvider {
	p.mapper = mapper
	return p
}

func (p *LocalProvider) resolve(path string) string {
	if p.basePath == "" {
		return path
	}
	// To prevent traversing outside base, we could add checks here later
	return filepath.Join(p.basePath, filepath.Clean(path))
}

func (p *LocalProvider) Stat(ctx context.Context, path string) (FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := p.resolve(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return WrapOSFileInfo(info), nil
}

func (p *LocalProvider) List(ctx context.Context, path string) ([]FileInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := p.resolve(path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var infos []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // skip files that disappeared between ReadDir and Info
		}
		infos = append(infos, WrapOSFileInfo(info))
	}
	return infos, nil
}

func (p *LocalProvider) OpenRead(ctx context.Context, path string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := p.resolve(path)
	return os.Open(fullPath)
}

func (p *LocalProvider) OpenWrite(ctx context.Context, path string, metadata FileInfo) (io.WriteCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	fullPath := p.resolve(path)

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, err
	}

	mode := os.FileMode(0644)
	if uInfo, ok := metadata.(UnixFileInfo); ok && uInfo.Mode() != 0 {
		mode = uInfo.Mode()
	}

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return nil, err
	}

	return &localWriteCloser{
		File:     file,
		fullPath: fullPath,
		metadata: metadata,
		mapper:   p.mapper,
	}, nil
}

// localWriteCloser wraps an os.File and applies metadata (such as timestamps) upon close.
// This is necessary because writing to the file updates its mtime.
type localWriteCloser struct {
	*os.File
	fullPath string
	metadata FileInfo
	mapper   *MetadataMapper
}

func (l *localWriteCloser) Close() error {
	err := l.File.Close()
	if err != nil {
		return err
	}

	// Apply any ownership and permissions mapped via mapper
	if l.mapper != nil && l.metadata != nil {
		// Ignore metadata application errors for now during sync (permissions issues, etc)
		_ = ApplyMetadata(l.fullPath, l.metadata, l.mapper)
	}

	if l.metadata != nil && !l.metadata.ModTime().IsZero() {
		// Ignore errors on applying timestamp
		_ = os.Chtimes(l.fullPath, time.Now(), l.metadata.ModTime())
	}

	return nil
}
