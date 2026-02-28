// Package registry Docker Registry API模块
// 实现Docker Registry HTTP API V2规范
package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cyp-registry/registry/src/modules/storage"
)

// ErrInvalidDigest 无效的摘要格式
var ErrInvalidDigest = errors.New("registry: invalid digest")

// ErrManifestNotFound Manifest不存在
var ErrManifestNotFound = errors.New("registry: manifest not found")

// ErrBlobNotFound Blob不存在
var ErrBlobNotFound = errors.New("registry: blob not found")

// ErrUploadNotFound 上传不存在
var ErrUploadNotFound = errors.New("registry: upload not found")

// ErrInvalidContentType 无效的内容类型
var ErrInvalidContentType = errors.New("registry: invalid content type")

// ErrImmutableTag 不可覆盖的历史版本标签（例如 v1.0.0）
var ErrImmutableTag = errors.New("registry: immutable tag cannot be overwritten")

// MediaType 定义OCI/Docker媒体类型
const (
	MediaTypeDocker2Manifest          = "application/vnd.docker.distribution.manifest.v2+json"
	MediaTypeDocker2ManifestList      = "application/vnd.docker.distribution.manifest.list.v2+json"
	MediaTypeOCIManifest              = "application/vnd.oci.image.manifest.v1+json"
	MediaTypeOCIManifestIndex         = "application/vnd.oci.image.index.v1+json"
	MediaTypeDocker2ImageLayer        = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	MediaTypeDocker2ImageLayerNonDist = "application/vnd.docker.image.rootfs.diff.tar"
	MediaTypeOCIImageLayer            = "application/vnd.oci.image.layer.v1.tar+gzip"
	MediaTypeOCIImageLayerNonDist     = "application/vnd.oci.image.layer.v1.tar"
	MediaTypeDocker2ImageConfig       = "application/vnd.docker.container.image.v1+json"
	MediaTypeOCIImageConfig           = "application/vnd.oci.image.config.v1+json"
)

// Registry Docker Registry服务
type Registry struct {
	storage storage.Storage

	// tagIndex:
	// 为了修复部分环境下 /v2/<name>/tags/list 无法正确列出标签的问题，
	// 在内存中维护当前进程内的 tag 索引（project -> set(tags)）。
	// 仍保留基于底层存储的 List 逻辑作为兜底。
	tagIndex map[string]map[string]struct{}
	mu       sync.RWMutex
}

// NewRegistry 创建Registry服务实例
func NewRegistry(store storage.Storage) *Registry {
	return &Registry{
		storage:  store,
		tagIndex: make(map[string]map[string]struct{}),
	}
}

// versionTagRegexp 用于识别“历史版本号”标签（不可覆盖），例如：
// v1.0.0、1.2.3、v2.3.4-beta、1.0.0-20260227 等。
var versionTagRegexp = regexp.MustCompile(`^(v)?\d+\.\d+\.\d+([._-][0-9A-Za-z]+)*$`)

// isImmutableTag 判断给定 tag 是否为“历史版本号”标签，若是则禁止覆盖。
// 例如：v1.0.0、1.2.3、v2.3.4-beta。
// 像 stable、prod、latest、dev 等“当前版本”标签则允许多次更新。
func isImmutableTag(tag string) bool {
	return versionTagRegexp.MatchString(tag)
}

// addTag 将 tag 写入内存索引
func (r *Registry) addTag(project, tag string) {
	if project == "" || tag == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.tagIndex[project]
	if !ok {
		m = make(map[string]struct{})
		r.tagIndex[project] = m
	}
	m[tag] = struct{}{}
}

// removeTag 从内存索引中移除指定 tag
func (r *Registry) removeTag(project, tag string) {
	if project == "" || tag == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.tagIndex[project]; ok {
		delete(m, tag)
		if len(m) == 0 {
			delete(r.tagIndex, project)
		}
	}
}

// getTagsFromIndex 从内存索引中获取指定仓库的所有 tag
func (r *Registry) getTagsFromIndex(project string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.tagIndex[project]
	if !ok || len(m) == 0 {
		return nil
	}
	tags := make([]string, 0, len(m))
	for t := range m {
		tags = append(tags, t)
	}
	return tags
}

// Manifest Docker镜像Manifest结构
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType,omitempty"`
	Config        LayerInfo         `json:"config"`
	Layers        []LayerInfo       `json:"layers"`
	Subject       *Descriptor       `json:"subject,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// ManifestIndex Docker镜像列表/OCI索引
type ManifestIndex struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType,omitempty"`
	Manifests     []Descriptor      `json:"manifests"`
	Annotations   map[string]string `json:"annotations,omitempty"`
}

// Descriptor 描述符（引用其他资源的结构）
type Descriptor struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

// LayerInfo 图层信息（简化版Descriptor）
type LayerInfo struct {
	Descriptor
}

// BlobInfo Blob信息
type BlobInfo struct {
	Digest       string    `json:"digest"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"contentType"`
	CreatedAt    time.Time `json:"createdAt"`
	UploadUUID   string    `json:"uploadUUID,omitempty"`
	UploadOffset int64     `json:"uploadOffset,omitempty"`
}

// UploadInfo 上传会话信息
type UploadInfo struct {
	UUID        string
	Project     string
	Reference   string
	Digest      string
	Size        int64
	StartedAt   time.Time
	LastUpdated time.Time
}

// UploadState 上传状态存储
type UploadState struct {
	Uploads map[string]*UploadInfo
}

// NewUploadState 创建新的上传状态
func NewUploadState() *UploadState {
	return &UploadState{
		Uploads: make(map[string]*UploadInfo),
	}
}

// ParseDigest 解析摘要字符串
// 格式: algorithm:hex
func ParseDigest(digest string) (algorithm, hexDigest string, err error) {
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return "", "", ErrInvalidDigest
	}

	algorithm = parts[0]
	hexDigest = parts[1]

	// 验证算法
	if algorithm != "sha256" && algorithm != "sha512" {
		return "", "", fmt.Errorf("%w: unsupported algorithm %s", ErrInvalidDigest, algorithm)
	}

	// 验证十六进制长度
	expectedLen := 64 // sha256
	if algorithm == "sha512" {
		expectedLen = 128
	}

	if len(hexDigest) != expectedLen {
		return "", "", fmt.Errorf("%w: invalid digest length", ErrInvalidDigest)
	}

	// 验证十六进制字符
	if _, err := hex.DecodeString(hexDigest); err != nil {
		return "", "", fmt.Errorf("%w: invalid digest characters", ErrInvalidDigest)
	}

	return algorithm, hexDigest, nil
}

// CalculateDigest 计算数据的SHA256摘要
func CalculateDigest(reader io.Reader) (string, int64, error) {
	hash := sha256.New()

	// 读取并计算哈希
	size, err := io.Copy(hash, reader)
	if err != nil {
		return "", 0, fmt.Errorf("failed to calculate digest: %w", err)
	}

	// 生成摘要
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	return digest, size, nil
}

// BuildBlobPath 构建Blob存储路径
// 路径格式: <project>/blobs/sha256/<digest>
func BuildBlobPath(project, digest string) string {
	return fmt.Sprintf("%s/blobs/sha256/%s", project, digest)
}

// BuildManifestPath 构建Manifest存储路径
// 路径格式: <project>/manifests/<reference>
func BuildManifestPath(project, reference string) string {
	return fmt.Sprintf("%s/manifests/%s", project, reference)
}

// ParseReference 解析仓库引用
// 支持: tag (latest) 或 digest (sha256:xxx)
func ParseReference(ref string) (isDigest bool, reference string) {
	if strings.HasPrefix(ref, "sha256:") {
		return true, ref
	}
	return false, ref
}
