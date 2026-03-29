package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wibus-wee/allinone/pkg/zcore/model"
	"github.com/wibus-wee/allinone/pkg/zgen/querier"
)

type AttachmentService interface {
	CreateAttachment(ctx context.Context, userID int32, filename string, contentType string, sizeBytes int64, data io.Reader) (*querier.Attachment, error)
	GetAttachment(ctx context.Context, userID int32, id uuid.UUID) (*querier.Attachment, error)
	DeleteAttachment(ctx context.Context, userID int32, id uuid.UUID) (*querier.Attachment, error)
	LinkAttachment(ctx context.Context, userID int32, attachmentID uuid.UUID, resourceType string, resourceID uuid.UUID) error
	UnlinkAttachment(ctx context.Context, userID int32, attachmentID uuid.UUID, resourceType string, resourceID uuid.UUID) error
	ListAttachmentsByResource(ctx context.Context, userID int32, resourceType string, resourceID uuid.UUID) ([]*querier.Attachment, error)
	ResolveStoragePath(item *querier.Attachment) string
	MaxSizeBytes() int64
}

var (
	ErrAttachmentNotFound = errors.New("attachment not found")
	ErrAttachmentTooLarge = errors.New("attachment too large")
)

type AttachmentConfig struct {
	StorageDir   string
	MaxSizeBytes int64
}

func LoadAttachmentConfig() AttachmentConfig {
	dir := strings.TrimSpace(os.Getenv("MYAPP_ATTACHMENTS_STORAGE_DIR"))
	if dir == "" {
		dir = "./data/attachments"
	}

	maxSizeMB := int64(25)
	if raw := strings.TrimSpace(os.Getenv("MYAPP_ATTACHMENTS_MAX_SIZE_MB")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			maxSizeMB = parsed
		}
	}

	return AttachmentConfig{
		StorageDir:   dir,
		MaxSizeBytes: maxSizeMB * 1024 * 1024,
	}
}

type attachmentService struct {
	model        model.ModelInterface
	storageDir   string
	maxSizeBytes int64
}

func NewAttachmentService(m model.ModelInterface, cfg AttachmentConfig) AttachmentService {
	return &attachmentService{
		model:        m,
		storageDir:   cfg.StorageDir,
		maxSizeBytes: cfg.MaxSizeBytes,
	}
}

func (s *attachmentService) MaxSizeBytes() int64 {
	return s.maxSizeBytes
}

func (s *attachmentService) CreateAttachment(ctx context.Context, userID int32, filename string, contentType string, sizeBytes int64, data io.Reader) (*querier.Attachment, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}
	if s.maxSizeBytes > 0 && sizeBytes > s.maxSizeBytes {
		return nil, ErrAttachmentTooLarge
	}

	id := uuid.New()
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) > 12 {
		ext = ""
	}
	relPath := filepath.Join(fmt.Sprintf("u%d", userID), id.String()+ext)
	fullPath := filepath.Join(s.storageDir, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return nil, err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(fullPath), "upload-*")
	if err != nil {
		return nil, err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = tempFile.Close()
	}()

	if _, err := io.Copy(tempFile, data); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}
	if err := os.Rename(tempPath, fullPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}

	item, err := s.model.CreateAttachment(ctx, querier.CreateAttachmentParams{
		ID:          id,
		UserID:      userID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		StoragePath: relPath,
	})
	if err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}

	return item, nil
}

func (s *attachmentService) GetAttachment(ctx context.Context, userID int32, id uuid.UUID) (*querier.Attachment, error) {
	item, err := s.model.GetAttachmentByID(ctx, querier.GetAttachmentByIDParams{
		ID:     id,
		UserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAttachmentNotFound
	}
	return item, err
}

func (s *attachmentService) DeleteAttachment(ctx context.Context, userID int32, id uuid.UUID) (*querier.Attachment, error) {
	item, err := s.model.DeleteAttachment(ctx, querier.DeleteAttachmentParams{
		ID:     id,
		UserID: userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}

	fullPath := s.ResolveStoragePath(item)
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return item, err
	}

	return item, nil
}

func (s *attachmentService) LinkAttachment(ctx context.Context, userID int32, attachmentID uuid.UUID, resourceType string, resourceID uuid.UUID) error {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return err
	}
	if _, err := s.GetAttachment(ctx, userID, attachmentID); err != nil {
		return err
	}
	return s.model.CreateAttachmentLink(ctx, querier.CreateAttachmentLinkParams{
		AttachmentID: attachmentID,
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}

func (s *attachmentService) UnlinkAttachment(ctx context.Context, userID int32, attachmentID uuid.UUID, resourceType string, resourceID uuid.UUID) error {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return err
	}
	if _, err := s.GetAttachment(ctx, userID, attachmentID); err != nil {
		return err
	}
	return s.model.DeleteAttachmentLink(ctx, querier.DeleteAttachmentLinkParams{
		AttachmentID: attachmentID,
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}

func (s *attachmentService) ListAttachmentsByResource(ctx context.Context, userID int32, resourceType string, resourceID uuid.UUID) ([]*querier.Attachment, error) {
	if err := s.model.EnsureUser(ctx, userID); err != nil {
		return nil, err
	}
	return s.model.ListAttachmentsByResource(ctx, querier.ListAttachmentsByResourceParams{
		UserID:       userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}

func (s *attachmentService) ResolveStoragePath(item *querier.Attachment) string {
	return filepath.Join(s.storageDir, item.StoragePath)
}
