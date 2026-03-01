// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/cyp-registry/registry/src/pkg/response"
	"github.com/google/uuid"
)

// APIError API错误响应
type APIError struct {
	Errors []ErrorDetail `json:"errors"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CheckBlob 检查Blob是否存在
// GET /v2/<name>/blobs/<digest>
func (r *Registry) CheckBlob(ctx context.Context, project, digest string) (bool, error) {
	// 验证摘要格式
	algorithm, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return false, err
	}

	// 存储路径使用完整摘要（包含算法前缀），与 CompleteBlobUpload/MountBlob/DeleteBlob 保持一致
	fullDigest := fmt.Sprintf("%s:%s", algorithm, hexDigest)
	path := BuildBlobPath(project, fullDigest)
	return r.storage.Exists(ctx, path)
}

// GetBlob 获取Blob
// GET /v2/<name>/blobs/<digest>
func (r *Registry) GetBlob(ctx context.Context, project, digest string) (io.Reader, int64, error) {
	// 验证摘要格式
	algorithm, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return nil, 0, err
	}

	// 存储路径使用完整摘要（包含算法前缀）
	fullDigest := fmt.Sprintf("%s:%s", algorithm, hexDigest)
	path := BuildBlobPath(project, fullDigest)

	reader, size, err := r.storage.Get(ctx, path)
	if err != nil {
		// 统一向上游暴露为 ErrBlobNotFound，方便控制器返回 404
		return nil, 0, ErrBlobNotFound
	}
	return reader, size, nil
}

// GetBlobSize 获取Blob大小
func (r *Registry) GetBlobSize(ctx context.Context, project, digest string) (int64, error) {
	algorithm, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return 0, err
	}

	// 存储路径使用完整摘要（包含算法前缀）
	fullDigest := fmt.Sprintf("%s:%s", algorithm, hexDigest)
	path := BuildBlobPath(project, fullDigest)
	size, _, err := r.storage.Stat(ctx, path)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return 0, ErrBlobNotFound
		}
		return 0, err
	}

	return size, nil
}

// InitiateBlobUpload 初始化Blob上传
// POST /v2/<name>/blobs/uploads/
func (r *Registry) InitiateBlobUpload(ctx context.Context, project string) (*UploadInfo, error) {
	uploadID := uuid.New().String()

	info := &UploadInfo{
		UUID:        uploadID,
		Project:     project,
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	return info, nil
}

// UploadBlobChunk 上传Blob分片
// PATCH /v2/<name>/blobs/uploads/<uuid>
func (r *Registry) UploadBlobChunk(ctx context.Context, project, uploadID string, offset int64, body io.Reader, size int64) (int64, error) {
	// 验证uploadID
	if uploadID == "" {
		return 0, ErrUploadNotFound
	}

	// 构建上传路径
	path := fmt.Sprintf("%s/uploads/%s", project, uploadID)

	// 检查是否已存在
	exists, err := r.storage.Exists(ctx, path)
	if err != nil {
		return 0, err
	}

	// 如果文件存在，获取当前大小作为offset
	if exists {
		currentSize, _, err := r.storage.Stat(ctx, path)
		if err != nil {
			return 0, err
		}
		// 追加模式：检查offset是否匹配
		if offset != 0 && offset != currentSize {
			return currentSize, fmt.Errorf("offset mismatch: expected %d, got %d", offset, currentSize)
		}
	}

	// 如果是追加，更新offset
	if exists {
		// 获取现有文件
		existing, _, err := r.storage.Get(ctx, path)
		if err != nil {
			return 0, err
		}

		// 将新数据追加到现有数据
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(existing); err != nil {
			return 0, err
		}

		// 读取新数据
		newData, err := io.ReadAll(body)
		if err != nil {
			return 0, err
		}

		// 合并数据
		buf.Write(newData)

		// 重新上传（删除旧数据失败时仅记录错误，由后续覆盖操作保证一致性）
		if err := r.storage.Delete(ctx, path); err != nil {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"append_blob","path":"%s","error":"failed to delete old blob: %v"}`, time.Now().Format(time.RFC3339), path, err)
		}
		if err := r.storage.Put(ctx, path, &buf, int64(buf.Len())); err != nil {
			return 0, err
		}

		return int64(buf.Len()), nil
	}

	// 首次上传
	if err := r.storage.Put(ctx, path, body, size); err != nil {
		return 0, err
	}

	return size, nil
}

// CompleteBlobUpload 完成Blob上传
// PUT /v2/<name>/blobs/uploads/<uuid>?digest=<digest>
func (r *Registry) CompleteBlobUpload(ctx context.Context, project, uploadID, digest string, size int64) error {
	if uploadID == "" {
		return ErrUploadNotFound
	}

	// 验证摘要格式
	algorithm, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return err
	}

	// 获取上传的数据
	uploadPath := fmt.Sprintf("%s/uploads/%s", project, uploadID)
	reader, _, err := r.storage.Get(ctx, uploadPath)
	if err != nil {
		return err
	}

	// 计算实际摘要
	actualDigest, actualSize, err := CalculateDigest(reader)
	if err != nil {
		return err
	}

	// 验证摘要匹配
	if digest != actualDigest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", digest, actualDigest)
	}

	// 验证大小匹配
	if size > 0 && actualSize != size {
		return fmt.Errorf("size mismatch: expected %d, got %d", size, actualSize)
	}

	// 移动到最终位置
	finalPath := BuildBlobPath(project, algorithm+":"+hexDigest)

	// 重新读取数据
	reader2, _, err := r.storage.Get(ctx, uploadPath)
	if err != nil {
		return err
	}

	// 删除上传的临时文件（删除失败不应影响后续读操作）
	if err := r.storage.Delete(ctx, uploadPath); err != nil {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"complete_blob_upload","upload_path":"%s","error":"failed to delete upload temp blob: %v"}`, time.Now().Format(time.RFC3339), uploadPath, err)
	}

	// 保存到最终位置
	return r.storage.Put(ctx, finalPath, reader2, actualSize)
}

// CancelBlobUpload 取消Blob上传
// DELETE /v2/<name>/blobs/uploads/<uuid>
func (r *Registry) CancelBlobUpload(ctx context.Context, project, uploadID string) error {
	if uploadID == "" {
		return ErrUploadNotFound
	}

	path := fmt.Sprintf("%s/uploads/%s", project, uploadID)
	return r.storage.Delete(ctx, path)
}

// MountBlob 跨仓库挂载Blob
// POST /v2/<name>/blobs/uploads/?mount=<digest>&from=<source-project>
func (r *Registry) MountBlob(ctx context.Context, project, sourceProject, digest string) error {
	// 验证摘要格式
	algorithm, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return err
	}

	// 源路径
	sourcePath := BuildBlobPath(sourceProject, algorithm+":"+hexDigest)

	// 检查源Blob是否存在
	exists, err := r.storage.Exists(ctx, sourcePath)
	if err != nil {
		return err
	}

	if !exists {
		return ErrBlobNotFound
	}

	// 目标路径
	destPath := BuildBlobPath(project, algorithm+":"+hexDigest)

	// 获取源Blob
	reader, size, err := r.storage.Get(ctx, sourcePath)
	if err != nil {
		return err
	}

	// 复制到目标位置
	return r.storage.Put(ctx, destPath, reader, size)
}

// DeleteBlob 删除Blob
// DELETE /v2/<name>/blobs/<digest>
func (r *Registry) DeleteBlob(ctx context.Context, project, digest string) error {
	// 验证摘要格式
	algorithm, hexDigest, err := ParseDigest(digest)
	if err != nil {
		return err
	}

	path := BuildBlobPath(project, algorithm+":"+hexDigest)
	return r.storage.Delete(ctx, path)
}

// GetBlobUploadStatus 获取上传状态
func (r *Registry) GetBlobUploadStatus(ctx context.Context, project, uploadID string) (*UploadInfo, error) {
	if uploadID == "" {
		return nil, ErrUploadNotFound
	}

	path := fmt.Sprintf("%s/uploads/%s", project, uploadID)

	exists, err := r.storage.Exists(ctx, path)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrUploadNotFound
	}

	size, _, err := r.storage.Stat(ctx, path)
	if err != nil {
		return nil, err
	}

	return &UploadInfo{
		UUID:        uploadID,
		Project:     project,
		Size:        size,
		LastUpdated: time.Now(),
	}, nil
}
