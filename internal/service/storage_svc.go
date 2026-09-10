package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tstech/backend/internal/config"
)

type StorageService struct {
	cfg       *config.Config
	uploadDir string
	s3Client  *s3.Client
	isR2      bool
}

func NewStorageService(cfg *config.Config) *StorageService {
	svc := &StorageService{
		cfg:       cfg,
		uploadDir: "./uploads",
	}

	// Check if Cloudflare R2 credentials are provided
	if cfg.R2AccountID != "" && cfg.R2AccessKeyID != "" && cfg.R2SecretAccessKey != "" && cfg.R2BucketName != "" {
		endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2AccountID)
		customCreds := credentials.NewStaticCredentialsProvider(cfg.R2AccessKeyID, cfg.R2SecretAccessKey, "")

		awsCfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
			awsConfig.WithCredentialsProvider(customCreds),
			awsConfig.WithRegion("auto"),
		)
		if err != nil {
			log.Printf("⚠️ Failed to load AWS config for R2: %v. Falling back to local storage.", err)
		} else {
			svc.s3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(endpoint)
			})
			svc.isR2 = true
			log.Printf("☁️ Cloudflare R2 Storage enabled (Bucket: %s)", cfg.R2BucketName)
			return svc
		}
	}

	// Fallback to local storage directory
	if err := os.MkdirAll(svc.uploadDir, 0755); err != nil {
		log.Printf("⚠️ Failed to create uploads dir: %v\n", err)
	}
	log.Println("📁 Using local storage for uploads (./uploads)")
	return svc
}

type UploadResult struct {
	FileName string `json:"file_name"`
	FileURL  string `json:"file_url"`
	FileSize int64  `json:"file_size"`
	FileType string `json:"file_type"`
}

func (s *StorageService) SaveFile(file *multipart.FileHeader) (*UploadResult, error) {
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	ext := filepath.Ext(file.Filename)
	safeName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 1. Always save to local storage (./uploads)
	if err := os.MkdirAll(s.uploadDir, 0755); err != nil {
		log.Printf("⚠️ Failed to ensure uploads dir: %v", err)
	}
	destPath := filepath.Join(s.uploadDir, safeName)
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create local file: %w", err)
	}

	size, err := io.Copy(dst, src)
	dst.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to save local file: %w", err)
	}

	// 2. If Cloudflare R2 is enabled, also upload copy to R2 bucket for backup
	if s.isR2 && s.s3Client != nil {
		if seeker, ok := src.(io.ReadSeeker); ok {
			_, _ = seeker.Seek(0, io.SeekStart)
			_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket:      aws.String(s.cfg.R2BucketName),
				Key:         aws.String(safeName),
				Body:        seeker,
				ContentType: aws.String(contentType),
			})
			if err != nil {
				log.Printf("⚠️ Cloudflare R2 backup upload failed: %v", err)
			}
		}
	}

	// Always return clean /uploads/ URL (which resolves correctly across dev, prod, and rewrites)
	fileURL := fmt.Sprintf("/uploads/%s", safeName)
	return &UploadResult{
		FileName: file.Filename,
		FileURL:  fileURL,
		FileSize: size,
		FileType: ext,
	}, nil
}

// GetFile retrieves a file from local storage or downloads from R2 if not present locally
func (s *StorageService) GetFile(filename string) (io.ReadCloser, string, error) {
	localPath := filepath.Join(s.uploadDir, filename)
	if f, err := os.Open(localPath); err == nil {
		return f, "", nil
	}

	// Fallback to R2 if available
	if s.isR2 && s.s3Client != nil {
		resp, err := s.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(s.cfg.R2BucketName),
			Key:    aws.String(filename),
		})
		if err == nil && resp.Body != nil {
			// Cache locally for instant subsequent loads
			_ = os.MkdirAll(s.uploadDir, 0755)
			if dst, err2 := os.Create(localPath); err2 == nil {
				b, _ := io.ReadAll(resp.Body)
				_, _ = dst.Write(b)
				_ = dst.Close()
				_ = resp.Body.Close()

				ct := ""
				if resp.ContentType != nil {
					ct = *resp.ContentType
				}
				if cachedF, err3 := os.Open(localPath); err3 == nil {
					return cachedF, ct, nil
				}
			}
			ct := ""
			if resp.ContentType != nil {
				ct = *resp.ContentType
			}
			return resp.Body, ct, nil
		}
	}

	return nil, "", os.ErrNotExist
}

