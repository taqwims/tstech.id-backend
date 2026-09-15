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
	Folder   string `json:"folder,omitempty"`
}

// sanitizeFolder cleans the folder path to prevent traversal and enforce forward slashes
func sanitizeFolder(folder string) string {
	f := strings.TrimSpace(folder)
	f = strings.ReplaceAll(f, "\\", "/")
	f = strings.Trim(f, "/")

	// Remove any traversal elements
	parts := strings.Split(f, "/")
	var cleanParts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "." && p != ".." {
			cleanParts = append(cleanParts, p)
		}
	}

	if len(cleanParts) == 0 {
		return "general"
	}
	return strings.Join(cleanParts, "/")
}

// SaveFile saves an uploaded file to storage (local and Cloudflare R2) in the specified folder
func (s *StorageService) SaveFile(file *multipart.FileHeader, folder ...string) (*UploadResult, error) {
	targetFolder := "general"
	if len(folder) > 0 && folder[0] != "" {
		targetFolder = sanitizeFolder(folder[0])
	}

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

	// Object key / relative path (e.g. "payments/1712345678_proof.jpg")
	objectKey := fmt.Sprintf("%s/%s", targetFolder, safeName)

	// 1. Always save to local storage (./uploads/<targetFolder>/<safeName>)
	targetLocalDir := filepath.Join(s.uploadDir, targetFolder)
	if err := os.MkdirAll(targetLocalDir, 0755); err != nil {
		log.Printf("⚠️ Failed to ensure local uploads folder: %v", err)
	}
	destPath := filepath.Join(targetLocalDir, safeName)
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create local file: %w", err)
	}

	size, err := io.Copy(dst, src)
	dst.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to save local file: %w", err)
	}

	// Default fallback to local path URL
	fileURL := fmt.Sprintf("/uploads/%s", objectKey)

	// 2. If Cloudflare R2 is enabled, upload to R2 bucket and use public R2 URL
	if s.isR2 && s.s3Client != nil {
		localSavedFile, err := os.Open(destPath)
		if err == nil {
			defer localSavedFile.Close()
			_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket:      aws.String(s.cfg.R2BucketName),
				Key:         aws.String(objectKey),
				Body:        localSavedFile,
				ContentType: aws.String(contentType),
			})
			if err != nil {
				log.Printf("⚠️ Cloudflare R2 upload failed: %v (using local storage fallback)", err)
			} else {
				log.Printf("☁️ Successfully uploaded to Cloudflare R2: %s", objectKey)
				if s.cfg.R2PublicURL != "" {
					fileURL = fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.R2PublicURL, "/"), objectKey)
				}
			}
		}
	}

	return &UploadResult{
		FileName: file.Filename,
		FileURL:  fileURL,
		FileSize: size,
		FileType: ext,
		Folder:   targetFolder,
	}, nil
}

// GetFile retrieves a file from local storage or downloads from R2 if not present locally
func (s *StorageService) GetFile(relPath string) (io.ReadCloser, string, error) {
	cleanRel := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(relPath)), "/")
	if cleanRel == "." || cleanRel == "" || strings.HasPrefix(cleanRel, "..") {
		return nil, "", os.ErrNotExist
	}

	// 1. Try local storage with exact relative path
	localPath := filepath.Join(s.uploadDir, filepath.FromSlash(cleanRel))
	if f, err := os.Open(localPath); err == nil {
		return f, "", nil
	}

	// 2. If R2 is active, try downloading from R2 with exact key
	if s.isR2 && s.s3Client != nil {
		resp, err := s.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: aws.String(s.cfg.R2BucketName),
			Key:    aws.String(cleanRel),
		})
		if err == nil && resp.Body != nil {
			// Cache locally for instant subsequent loads
			_ = os.MkdirAll(filepath.Dir(localPath), 0755)
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

	// 3. Fallback for legacy filenames without folder prefix
	if !strings.Contains(cleanRel, "/") {
		knownFolders := []string{"general", "payments", "projects", "cms", "media", "avatars"}
		for _, kf := range knownFolders {
			subPath := filepath.Join(s.uploadDir, kf, cleanRel)
			if f, err := os.Open(subPath); err == nil {
				return f, "", nil
			}

			if s.isR2 && s.s3Client != nil {
				r2Key := fmt.Sprintf("%s/%s", kf, cleanRel)
				resp, err := s.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
					Bucket: aws.String(s.cfg.R2BucketName),
					Key:    aws.String(r2Key),
				})
				if err == nil && resp.Body != nil {
					_ = os.MkdirAll(filepath.Dir(subPath), 0755)
					if dst, err2 := os.Create(subPath); err2 == nil {
						b, _ := io.ReadAll(resp.Body)
						_, _ = dst.Write(b)
						_ = dst.Close()
						_ = resp.Body.Close()
						if cachedF, err3 := os.Open(subPath); err3 == nil {
							ct := ""
							if resp.ContentType != nil {
								ct = *resp.ContentType
							}
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
		}
	}

	return nil, "", os.ErrNotExist
}

