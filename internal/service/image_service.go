package service

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"skintrader-go/internal/config"
)

type ImageService struct {
	uploadConfig config.UploadConfig
}

func NewImageService(cfg config.UploadConfig) *ImageService {
	return &ImageService{uploadConfig: cfg}
}

type ProcessedImage struct {
	OriginalPath  string
	ThumbnailPath string
	Filename      string
	Size          int64
	MimeType      string
	Width         int
	Height        int
}

// ProcessPostImage processes an uploaded post image (resize + thumbnail).
func (s *ImageService) ProcessPostImage(filePath string) (*ProcessedImage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		// Non-fatal, proceed without dimensions
		cfg = image.Config{}
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := "image/jpeg"
	switch ext {
	case ".png":
		mimeType = "image/png"
	case ".webp":
		mimeType = "image/webp"
	case ".heic", ".heif":
		mimeType = "image/heic"
	}

	// TODO: Use imaging library to resize and create thumbnail
	// For now, return the original path
	thumbnailPath := strings.TrimSuffix(filePath, ext) + "_thumb" + ext

	return &ProcessedImage{
		OriginalPath:  filePath,
		ThumbnailPath: thumbnailPath,
		Filename:      filepath.Base(filePath),
		Size:          info.Size(),
		MimeType:      mimeType,
		Width:         cfg.Width,
		Height:        cfg.Height,
	}, nil
}

// ProcessKYCImage processes a KYC document image.
func (s *ImageService) ProcessKYCImage(filePath string) (string, error) {
	// TODO: Resize KYC image to max 2048x2048
	return filePath, nil
}

// ProcessAvatar processes a profile avatar image.
func (s *ImageService) ProcessAvatar(filePath string) (string, error) {
	// TODO: Resize to 500x500
	return filePath, nil
}

// GenerateFilename creates a unique filename with the given extension.
func (s *ImageService) GenerateFilename(ext string) string {
	return uuid.New().String() + ext
}

// GetUploadPath returns the full upload path for a given subdirectory.
func (s *ImageService) GetUploadPath(subdir string) string {
	return filepath.Join(s.uploadConfig.Dir, subdir)
}

// EnsureUploadDir creates the upload directory if it doesn't exist.
func (s *ImageService) EnsureUploadDir(subdir string) error {
	dir := s.GetUploadPath(subdir)
	return os.MkdirAll(dir, 0755)
}

// CleanupFile removes a file from disk.
func (s *ImageService) CleanupFile(filePath string) error {
	if filePath == "" {
		return nil
	}
	return os.Remove(filePath)
}
