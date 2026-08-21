package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
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

	// If Cloudflare R2 is enabled, upload directly to R2 bucket
	if s.isR2 && s.s3Client != nil {
		_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(s.cfg.R2BucketName),
			Key:         aws.String(safeName),
			Body:        src,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upload file to Cloudflare R2: %w", err)
		}

		publicBase := strings.TrimRight(s.cfg.R2PublicURL, "/")
		var fileURL string
		if publicBase != "" {
			fileURL = fmt.Sprintf("%s/%s", publicBase, safeName)
		} else {
			fileURL = fmt.Sprintf("https://%s.%s.r2.cloudflarestorage.com/%s", s.cfg.R2BucketName, s.cfg.R2AccountID, safeName)
		}

		return &UploadResult{
			FileName: file.Filename,
			FileURL:  fileURL,
			FileSize: file.Size,
			FileType: ext,
		}, nil
	}

	// Fallback: Local storage
	destPath := filepath.Join(s.uploadDir, safeName)
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	size, err := io.Copy(dst, src)
	if err != nil {
		return nil, err
	}

	fileURL := fmt.Sprintf("/uploads/%s", safeName)
	return &UploadResult{
		FileName: file.Filename,
		FileURL:  fileURL,
		FileSize: size,
		FileType: ext,
	}, nil
}

