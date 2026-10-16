package local

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/medusa-go-clone/internal/modules/file/domain/models"
)

type Provider struct {
	publicDir, privateDir string
	backendURL            string
}

func New(publicDir, privateDir, backendURL string) *Provider {
	if publicDir == "" {
		publicDir = filepath.Join("static")
	}
	if privateDir == "" {
		privateDir = publicDir
	}
	if backendURL == "" {
		backendURL = "http://localhost:8080/static"
	}
	return &Provider{publicDir: publicDir, privateDir: privateDir, backendURL: backendURL}
}
func (p *Provider) resolve(key string) (string, error) {
	base := p.publicDir
	if strings.HasPrefix(filepath.Base(key), "private-") {
		base = p.privateDir
	}
	root, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(key)))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid file key: %s", key)
	}
	return path, nil
}
func (p *Provider) key(name, access string) string {
	dir := filepath.ToSlash(filepath.Dir(name))
	base := filepath.Base(name)
	prefix := ""
	if access != "public" {
		prefix = "private-"
	}
	file := prefix + fmt.Sprintf("%d-%s", time.Now().UnixMilli(), base)
	if dir == "." {
		return file
	}
	return dir + "/" + file
}
func (p *Provider) Upload(ctx context.Context, in models.UploadInput) (models.File, error) {
	if err := ctx.Err(); err != nil {
		return models.File{}, err
	}
	if in.Filename == "" {
		return models.File{}, fmt.Errorf("no filename provided")
	}
	key := p.key(in.Filename, in.Access)
	path, err := p.resolve(key)
	if err != nil {
		return models.File{}, err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return models.File{}, err
	}
	if err = os.WriteFile(path, in.Content, 0644); err != nil {
		return models.File{}, err
	}
	return models.File{ID: key, URL: p.fileURL(key)}, nil
}
func (p *Provider) Delete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		path, err := p.resolve(id)
		if err != nil {
			return err
		}
		if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
func (p *Provider) PresignedDownloadURL(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := p.resolve(id)
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(path); err != nil {
		return "", fmt.Errorf("file with key %s not found", id)
	}
	return p.fileURL(id), nil
}
func (p *Provider) PresignedUploadURL(ctx context.Context, name, mimeType, access string) (models.File, error) {
	if err := ctx.Err(); err != nil {
		return models.File{}, err
	}
	if name == "" {
		return models.File{}, fmt.Errorf("no filename provided")
	}
	return models.File{ID: name, URL: "/admin/uploads"}, nil
}
func (p *Provider) fileURL(key string) string {
	u, err := url.Parse(p.backendURL)
	if err != nil {
		return p.backendURL + "/" + key
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + filepath.ToSlash(key)
	return u.String()
}
