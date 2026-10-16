package s3

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/example/medusa-go-clone/internal/modules/file/domain/models"
	entityid "github.com/example/medusa-go-clone/internal/utils/id"
)

const Identifier = "s3"

type Config struct {
	FileURL              string
	AccessKeyID          string
	SecretAccessKey      string
	SessionToken         string
	AuthenticationMethod string
	Region               string
	Bucket               string
	Prefix               string
	Endpoint             string
	CacheControl         string
	DownloadFileDuration time.Duration
	ACL                  *string
	DisableACL           bool
	UsePathStyle         bool
}

type Provider struct {
	config    Config
	client    *awss3.Client
	presigner *awss3.PresignClient
	logger    *slog.Logger
}

func New(ctx context.Context, config Config, logger *slog.Logger) (*Provider, error) {
	method := config.AuthenticationMethod
	if method == "" {
		method = "access-key"
	}
	if method == "access-key" && (config.AccessKeyID == "" || config.SecretAccessKey == "") {
		return nil, fmt.Errorf("Access key ID and secret access key are required when using access key authentication")
	}
	loadOptions := []func(*awsconfig.LoadOptions) error{}
	if config.Region != "" {
		loadOptions = append(loadOptions, awsconfig.WithRegion(config.Region))
	}
	if method == "access-key" {
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, config.SessionToken)))
	} else if method != "s3-iam-role" {
		return nil, fmt.Errorf("unsupported S3 authentication method %q", method)
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}
	if config.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(config.Endpoint)
	}
	client := awss3.NewFromConfig(awsCfg, func(options *awss3.Options) { options.UsePathStyle = config.UsePathStyle })
	if config.CacheControl == "" {
		config.CacheControl = "public, max-age=31536000"
	}
	if config.DownloadFileDuration <= 0 {
		config.DownloadFileDuration = time.Hour
	}
	config.AuthenticationMethod = method
	if logger == nil {
		logger = slog.Default()
	}
	return &Provider{config: config, client: client, presigner: awss3.NewPresignClient(client), logger: logger}, nil
}

func (*Provider) Identifier() string { return Identifier }

func (p *Provider) Upload(ctx context.Context, input models.UploadInput) (models.File, error) {
	if input.Filename == "" {
		return models.File{}, fmt.Errorf("No filename provided")
	}
	clean := sanitizeFilePath(input.Filename)
	if clean == "" {
		return models.File{}, fmt.Errorf("Invalid filename: %s", input.Filename)
	}
	dir, filename := path.Split(clean)
	ext := path.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	key := p.config.Prefix
	if dir != "" {
		key += dir
	}
	key += base + "-" + entityid.New("") + ext
	acl := p.resolveACL(input.Access, true)
	metadata := map[string]string{"original-filename": encodeURIComponent(input.Filename)}
	_, err := p.client.PutObject(ctx, &awss3.PutObjectInput{Bucket: aws.String(p.config.Bucket), Key: aws.String(key), Body: bytes.NewReader(input.Content), ContentType: aws.String(input.MIMEType), CacheControl: aws.String(p.config.CacheControl), Metadata: metadata, ACL: acl})
	if err != nil {
		p.logger.Error("S3 upload failed", "error", err)
		return models.File{}, err
	}
	return models.File{ID: key, URL: p.publicURL(key)}, nil
}

func (p *Provider) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	objects := make([]types.ObjectIdentifier, 0, len(ids))
	for _, id := range ids {
		objects = append(objects, types.ObjectIdentifier{Key: aws.String(id)})
	}
	_, err := p.client.DeleteObjects(ctx, &awss3.DeleteObjectsInput{Bucket: aws.String(p.config.Bucket), Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)}})
	if err != nil {
		p.logger.Error("S3 delete failed", "error", err)
	}
	// The Medusa S3 provider logs and swallows delete errors.
	return nil
}

func (p *Provider) PresignedDownloadURL(ctx context.Context, key string) (string, error) {
	result, err := p.presigner.PresignGetObject(ctx, &awss3.GetObjectInput{Bucket: aws.String(p.config.Bucket), Key: aws.String(key)}, func(options *awss3.PresignOptions) { options.Expires = p.config.DownloadFileDuration })
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func (p *Provider) PresignedUploadURL(ctx context.Context, filename, mimeType, access string) (models.File, error) {
	if filename == "" {
		return models.File{}, fmt.Errorf("No filename provided")
	}
	clean := sanitizeFilePath(filename)
	if clean == "" {
		return models.File{}, fmt.Errorf("Invalid filename: %s", filename)
	}
	key := p.config.Prefix + clean
	input := &awss3.PutObjectInput{Bucket: aws.String(p.config.Bucket), Key: aws.String(key), ContentType: aws.String(mimeType)}
	if access != "" {
		input.ACL = p.resolveACL(access, false)
	}
	result, err := p.presigner.PresignPutObject(ctx, input, func(options *awss3.PresignOptions) { options.Expires = time.Hour })
	if err != nil {
		return models.File{}, err
	}
	return models.File{ID: key, URL: result.URL}, nil
}

func (p *Provider) resolveACL(access string, serverUpload bool) types.ObjectCannedACL {
	if p.config.DisableACL {
		return ""
	}
	if p.config.ACL != nil {
		return types.ObjectCannedACL(*p.config.ACL)
	}
	if !serverUpload && access == "" {
		return ""
	}
	value := types.ObjectCannedACLPrivate
	if access == "public" {
		value = types.ObjectCannedACLPublicRead
	}
	return value
}

func (p *Provider) publicURL(key string) string {
	parts := strings.Split(key, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return p.config.FileURL + "/" + strings.Join(parts, "/")
}

func sanitizeFilePath(filename string) string {
	clean := strings.TrimLeft(strings.ReplaceAll(filename, "\\", "/"), "/")
	clean = path.Clean(clean)
	parts := strings.Split(clean, "/")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != "." && part != ".." {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "/")
}

func encodeURIComponent(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}
