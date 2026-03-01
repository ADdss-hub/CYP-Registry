// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// BlobExists 检查Blob是否存在（带摘要验证）
func (r *Registry) BlobExists(ctx context.Context, project, digest string) (bool, int64, error) {
	exists, err := r.CheckBlob(ctx, project, digest)
	if err != nil || !exists {
		return exists, 0, err
	}

	size, err := r.GetBlobSize(ctx, project, digest)
	if err != nil {
		return false, 0, err
	}

	return true, size, nil
}

// ParseContentRange 解析Content-Range头
// 格式: bytes start-end/total
func ParseContentRange(contentRange string) (start, end, total int64, err error) {
	// 格式: bytes start-end/total
	parts := strings.SplitN(contentRange, " ", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("invalid content-range format: %s", contentRange)
	}

	rangePart := parts[1]
	parts = strings.SplitN(rangePart, "/", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("invalid content-range format: %s", contentRange)
	}

	rangeBytes := parts[0]
	total, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid total in content-range: %w", err)
	}

	parts = strings.SplitN(rangeBytes, "-", 2)
	if len(parts) != 2 {
		return 0, 0, 0, fmt.Errorf("invalid range in content-range: %s", rangeBytes)
	}

	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid start in content-range: %w", err)
	}

	end, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid end in content-range: %w", err)
	}

	return start, end, total, nil
}
