package provider

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
)

// ensure interface is implemented
var _ Provider = (*S3Provider)(nil)

type s3FileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (f *s3FileInfo) Name() string       { return f.name }
func (f *s3FileInfo) Size() int64        { return f.size }
func (f *s3FileInfo) IsDir() bool        { return f.isDir }
func (f *s3FileInfo) ModTime() time.Time { return f.modTime }

type S3Provider struct {
	client *s3.Client
	bucket string
	prefix string
	uploader *manager.Uploader
}

// NewS3Provider creates a new S3Provider.
// bucket is the S3 bucket name.
func NewS3Provider(ctx context.Context, bucket string, prefix string) (*S3Provider, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
    uploader := manager.NewUploader(client)

	return &S3Provider{
		client:   client,
		bucket:   bucket,
		prefix:   prefix,
        uploader: uploader,
	}, nil
}

// buildKey constructs the full S3 key based on the provider's prefix
func (p *S3Provider) buildKey(subPath string) string {
	subPath = strings.TrimPrefix(subPath, "/")
	if p.prefix == "" {
		return subPath
	}
	// Avoid double slashes
	key := path.Join(p.prefix, subPath)
	return strings.TrimPrefix(key, "/")
}

// Stat returns the FileInfo for the given path.
func (p *S3Provider) Stat(ctx context.Context, pth string) (FileInfo, error) {
	key := p.buildKey(pth)

	// exact match
	headOut, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})

	if err == nil {
		var modTime time.Time
		if headOut.LastModified != nil {
			modTime = *headOut.LastModified
		}
		var size int64
		if headOut.ContentLength != nil {
			size = *headOut.ContentLength
		}

		return &s3FileInfo{
			name:    path.Base(key),
			size:    size,
			isDir:   strings.HasSuffix(key, "/"),
			modTime: modTime,
		}, nil
	}

	// maybe a directory? Let's check prefix
	dirPrefix := key + "/"
	if key == "" {
		dirPrefix = ""
	}

	listOut, err := p.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(p.bucket),
		Prefix:  aws.String(dirPrefix),
		MaxKeys: aws.Int32(1),
	})

	if err != nil {
		return nil, fmt.Errorf("stat failed for %q: %w", pth, err)
	}

	// if objects exist, treat as directory
	// listOut.Contents actually isn't 100% full proof if there are no contents
	// but there are CommonPrefixes.
	
	if len(listOut.Contents) > 0 {
		return &s3FileInfo{
			name:  path.Base(key),
			isDir: true,
		}, nil
	}
	
	if len(listOut.CommonPrefixes) > 0 {
		return &s3FileInfo{
			name:  path.Base(key),
			isDir: true,
		}, nil
	}

	return nil, fmt.Errorf("file not found: %s", pth)
}

// List returns the contents of the given directory.
func (p *S3Provider) List(ctx context.Context, pth string) ([]FileInfo, error) {
	dirPrefix := p.buildKey(pth)
	if dirPrefix != "" && !strings.HasSuffix(dirPrefix, "/") {
		dirPrefix += "/"
	}

	var infos []FileInfo
	var continuationToken *string

	for {
		out, err := p.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(p.bucket),
			Prefix:            aws.String(dirPrefix),
			Delimiter:         aws.String("/"),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list %q: %w", pth, err)
		}

		// Add common prefixes as directories
		for _, cp := range out.CommonPrefixes {
			name := strings.TrimPrefix(*cp.Prefix, dirPrefix)
			name = strings.TrimSuffix(name, "/")
			infos = append(infos, &s3FileInfo{
				name:  name,
				isDir: true,
			})
		}

		// Add objects as files (or explicit directories if they end in /)
		for _, obj := range out.Contents {
			name := strings.TrimPrefix(*obj.Key, dirPrefix)
			if name == "" { // sometimes the dir itself is in the results
				continue
			}
			isDir := strings.HasSuffix(name, "/")
			if isDir {
				name = strings.TrimSuffix(name, "/")
			}

			var modTime time.Time
			if obj.LastModified != nil {
				modTime = *obj.LastModified
			}
			var size int64
			if obj.Size != nil {
				size = *obj.Size
			}

			infos = append(infos, &s3FileInfo{
				name:    name,
				size:    size,
				isDir:   isDir,
				modTime: modTime,
			})
		}

		if out.IsTruncated != nil && *out.IsTruncated {
			continuationToken = out.NextContinuationToken
		} else {
			break
		}
	}

	return infos, nil
}

// OpenRead opens a file for streaming reads.
func (p *S3Provider) OpenRead(ctx context.Context, pth string) (io.ReadCloser, error) {
	key := p.buildKey(pth)
	out, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open read %q: %w", pth, err)
	}
	return out.Body, nil
}

// OpenWrite opens a file for streaming writes.
func (p *S3Provider) OpenWrite(ctx context.Context, pth string, metadata FileInfo) (io.WriteCloser, error) {
	key := p.buildKey(pth)

	// Check if this is just a directory placeholder we need to create
	if metadata != nil && metadata.IsDir() {
		// S3 doesn't have true directories, but writing a 0-byte object ending in '/' simulates it
		if !strings.HasSuffix(key, "/") {
			key += "/"
		}
		
		_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(p.bucket),
			Key:    aws.String(key),
			Body:   strings.NewReader(""),
		})
		
		if err != nil {
			return nil, fmt.Errorf("failed to write directory placeholder: %w", err)
		}
		
		// Return a dummy writer since we're done
		return &dummyWriter{}, nil
	}

	// Standard file upload
	pr, pw := io.Pipe()

	errChan := make(chan error, 1)

	go func() {
		_, err := p.uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket: aws.String(p.bucket),
			Key:    aws.String(key),
			Body:   pr,
		})
		pr.CloseWithError(err)
		errChan <- err
	}()

	return &asyncS3Writer{
		pw:      pw,
		errChan: errChan,
	}, nil
}

type asyncS3Writer struct {
	pw      *io.PipeWriter
	errChan <-chan error
}

func (w *asyncS3Writer) Write(p []byte) (n int, err error) {
	return w.pw.Write(p)
}

func (w *asyncS3Writer) Close() error {
	if err := w.pw.Close(); err != nil {
		return err
	}
	// Wait for upload to complete
	if err := <-w.errChan; err != nil {
		return fmt.Errorf("s3 upload failed: %w", err)
	}
	return nil
}

type dummyWriter struct{}

func (w *dummyWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (w *dummyWriter) Close() error {
	return nil
}
