package common

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/colin-404/logx"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Error definitions
var (
	gridFSBucket          *gridfs.Bucket
	ErrImageNotFound      = errors.New("image not found")
	ErrInvalidImageType   = errors.New("invalid image type")
	ErrImageAlreadyExists = errors.New("image already exists")
)

// ImageMeta represents image metadata structure
type ImageMeta struct {
	ImageID      string             `bson:"imageID" json:"imageID"`                       // Image ID
	GridFSID     primitive.ObjectID `bson:"gridfsID" json:"gridfsID"`                     // Internal GridFS ID
	OriginalName string             `bson:"originalName" json:"originalName"`             // Original filename
	ContentType  string             `bson:"contentType" json:"contentType"`               // MIME type
	Size         int64              `bson:"size" json:"size"`                             // File size in bytes
	Checksum     string             `bson:"checksum" json:"checksum"`                     // SHA256 checksum
	Tags         []string           `bson:"tags,omitempty" json:"tags,omitempty"`         // Tags
	Metadata     map[string]any     `bson:"metadata,omitempty" json:"metadata,omitempty"` // Custom metadata
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`                   // Creation time
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`                   // Update time
}

// ImageStore manages image storage operations
type ImageStore struct {
	bucket         *gridfs.Bucket
	metaCollection *mongo.Collection
}

// NewImageStore creates a new image store manager
func NewImageStore() *ImageStore {
	return &ImageStore{
		bucket:         gridFSBucket,
		metaCollection: GetCollection("imageMetadata"),
	}
}

// isValidImageType checks if the content type is a valid image type
func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg", "image/jpg", "image/png", "image/gif",
		"image/webp", "image/bmp", "image/tiff", "image/svg+xml",
	}
	for _, validType := range validTypes {
		if contentType == validType {
			return true
		}
	}
	return false
}

// StoreImageFromFile stores an image from a file path
func (is *ImageStore) StoreImageFromFile(ctx context.Context, imagePath string, tags []string, metadata map[string]any) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return is.StoreImageFromReader(ctx, file, filepath.Base(imagePath), tags, metadata)
}

// StoreImageFromReader stores an image from an io.Reader
func (is *ImageStore) StoreImageFromReader(ctx context.Context, reader io.Reader, filename string, tags []string, metadata map[string]any) (string, error) {
	// Read file content for checksum calculation and type detection
	imageData, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Detect content type
	contentType := http.DetectContentType(imageData)
	if !isValidImageType(contentType) {
		return "", ErrInvalidImageType
	}

	// Calculate checksum
	checksum := fmt.Sprintf("%x", sha256.Sum256(imageData))

	// Check if image with same checksum already exists
	existing, err := is.GetImageByChecksum(ctx, checksum)
	if err == nil && existing != nil {
		return existing.ImageID, ErrImageAlreadyExists
	}

	// Generate unique ID
	imageID := xid.New().String()
	gridfsID := primitive.NewObjectID()

	// Store to GridFS
	uploadStream, err := is.bucket.OpenUploadStreamWithID(gridfsID, filename)
	if err != nil {
		return "", fmt.Errorf("failed to open upload stream: %w", err)
	}
	defer uploadStream.Close()

	_, err = uploadStream.Write(imageData)
	if err != nil {
		return "", fmt.Errorf("failed to write to GridFS: %w", err)
	}

	// Save metadata
	now := time.Now()
	imageMeta := ImageMeta{
		ImageID:      imageID,
		GridFSID:     gridfsID,
		OriginalName: filename,
		ContentType:  contentType,
		Size:         int64(len(imageData)),
		Checksum:     checksum,
		Tags:         tags,
		Metadata:     metadata,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = is.metaCollection.InsertOne(ctx, imageMeta)
	if err != nil {
		// If metadata save fails, cleanup GridFS file
		_ = is.bucket.Delete(gridfsID)
		return "", fmt.Errorf("failed to save metadata: %w", err)
	}

	logx.Infof("Successfully stored image: %s (size: %d bytes)", imageID, imageMeta.Size)
	return imageID, nil
}

// GetImageMeta retrieves image metadata by ID
func (is *ImageStore) GetImageMeta(ctx context.Context, imageID string) (*ImageMeta, error) {
	var imageMeta ImageMeta
	err := is.metaCollection.FindOne(ctx, bson.M{"imageID": imageID}).Decode(&imageMeta)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("failed to get image metadata: %w", err)
	}
	return &imageMeta, nil
}

// GetImageByChecksum retrieves image metadata by checksum
func (is *ImageStore) GetImageByChecksum(ctx context.Context, checksum string) (*ImageMeta, error) {
	var imageMeta ImageMeta
	err := is.metaCollection.FindOne(ctx, bson.M{"checksum": checksum}).Decode(&imageMeta)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("failed to get image by checksum: %w", err)
	}
	return &imageMeta, nil
}

// GetImageData retrieves image data stream by ID
func (is *ImageStore) GetImageData(ctx context.Context, imageID string) (io.ReadCloser, *ImageMeta, error) {
	// First get metadata
	imageMeta, err := is.GetImageMeta(ctx, imageID)
	if err != nil {
		return nil, nil, err
	}

	// Get file stream from GridFS
	downloadStream, err := is.bucket.OpenDownloadStream(imageMeta.GridFSID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open download stream: %w", err)
	}

	return downloadStream, imageMeta, nil
}

// DownloadImageToFile downloads image to a file
func (is *ImageStore) DownloadImageToFile(ctx context.Context, imageID, outputPath string) error {
	downloadStream, imageMeta, err := is.GetImageData(ctx, imageID)
	if err != nil {
		return err
	}
	defer downloadStream.Close()

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Copy data
	_, err = io.Copy(outputFile, downloadStream)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	logx.Infof("Successfully downloaded image %s to %s (size: %d bytes)", imageID, outputPath, imageMeta.Size)
	return nil
}

// DeleteImage deletes image and its metadata
func (is *ImageStore) DeleteImage(ctx context.Context, imageID string) error {
	// Get metadata
	imageMeta, err := is.GetImageMeta(ctx, imageID)
	if err != nil {
		return err
	}

	// Delete GridFS file
	err = is.bucket.Delete(imageMeta.GridFSID)
	if err != nil {
		return fmt.Errorf("failed to delete from GridFS: %w", err)
	}

	// Delete metadata
	_, err = is.metaCollection.DeleteOne(ctx, bson.M{"imageID": imageID})
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	logx.Infof("Successfully deleted image: %s", imageID)
	return nil
}

// ListImages lists images with pagination and filtering support
func (is *ImageStore) ListImages(ctx context.Context, tags []string, limit, offset int64) ([]*ImageMeta, error) {
	filter := bson.M{}
	if len(tags) > 0 {
		filter["tags"] = bson.M{"$in": tags}
	}

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
	}
	if offset > 0 {
		opts.SetSkip(offset)
	}
	opts.SetSort(bson.M{"createdAt": -1}) // Sort by creation time descending

	cursor, err := is.metaCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find images: %w", err)
	}
	defer cursor.Close(ctx)

	var images []*ImageMeta
	for cursor.Next(ctx) {
		var imageMeta ImageMeta
		if err := cursor.Decode(&imageMeta); err != nil {
			return nil, fmt.Errorf("failed to decode image metadata: %w", err)
		}
		images = append(images, &imageMeta)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return images, nil
}

// UpdateImageTags updates image tags
func (is *ImageStore) UpdateImageTags(ctx context.Context, imageID string, tags []string) error {
	update := bson.M{
		"$set": bson.M{
			"tags":      tags,
			"updatedAt": time.Now(),
		},
	}

	result, err := is.metaCollection.UpdateOne(ctx, bson.M{"imageID": imageID}, update)
	if err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrImageNotFound
	}

	return nil
}

// UpdateImageMetadata updates image custom metadata
func (is *ImageStore) UpdateImageMetadata(ctx context.Context, imageID string, metadata map[string]any) error {
	update := bson.M{
		"$set": bson.M{
			"metadata":  metadata,
			"updatedAt": time.Now(),
		},
	}

	result, err := is.metaCollection.UpdateOne(ctx, bson.M{"imageID": imageID}, update)
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrImageNotFound
	}

	return nil
}

// GetImageStats 获取图片存储统计信息
func (is *ImageStore) GetImageStats(ctx context.Context) (map[string]any, error) {
	// Count total images
	totalCount, err := is.metaCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count images: %w", err)
	}

	// Calculate total size
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":       nil,
			"totalSize": bson.M{"$sum": "$size"},
			"avgSize":   bson.M{"$avg": "$size"},
		}},
	}

	cursor, err := is.metaCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalSize int64   `bson:"totalSize"`
		AvgSize   float64 `bson:"avgSize"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
	}

	stats := map[string]any{
		"totalCount": totalCount,
		"totalSize":  result.TotalSize,
		"avgSize":    result.AvgSize,
	}

	return stats, nil
}

// CleanupOrphanedFiles cleans up orphaned files (files in GridFS but not in metadata)
func (is *ImageStore) CleanupOrphanedFiles(ctx context.Context) (int, error) {
	// Get all GridFS file IDs
	gridfsFiles, err := is.bucket.Find(bson.M{})
	if err != nil {
		return 0, fmt.Errorf("failed to list GridFS files: %w", err)
	}
	defer gridfsFiles.Close(ctx)

	var orphanedCount int
	for gridfsFiles.Next(ctx) {
		var file struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := gridfsFiles.Decode(&file); err != nil {
			continue
		}

		// Check if corresponding record exists in metadata
		count, err := is.metaCollection.CountDocuments(ctx, bson.M{"gridfsID": file.ID})
		if err != nil {
			continue
		}

		// If not in metadata, delete GridFS file
		if count == 0 {
			if err := is.bucket.Delete(file.ID); err != nil {
				logx.Errorf("Failed to delete orphaned file %s: %v", file.ID.Hex(), err)
			} else {
				orphanedCount++
				logx.Infof("Deleted orphaned file: %s", file.ID.Hex())
			}
		}
	}

	return orphanedCount, nil
}
