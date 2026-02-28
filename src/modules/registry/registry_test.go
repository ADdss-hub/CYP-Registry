// Package registry_test Registry模块测试
package registry_test

import (
	"bytes"
	"testing"

	"github.com/cyp-registry/registry/src/modules/registry"
	"github.com/cyp-registry/registry/src/modules/storage/driver"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestStorage 创建测试用本地存储
func createTestStorage(t *testing.T) *driver.LocalStorage {
	tempDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Set("storage.local.path", tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)

	return store
}

// TestCreateTestStorage 确保测试辅助函数可正常工作
func TestCreateTestStorage(t *testing.T) {
	store := createTestStorage(t)
	require.NotNil(t, store)
}

// TestBuildBlobPath 测试构建Blob路径
func TestBuildBlobPath(t *testing.T) {
	path := registry.BuildBlobPath("myproject", "sha256:abc123")
	expected := "myproject/blobs/sha256/sha256:abc123"

	assert.Equal(t, expected, path)
}

// TestBuildManifestPath 测试构建Manifest路径
func TestBuildManifestPath(t *testing.T) {
	path := registry.BuildManifestPath("myproject", "latest")
	expected := "myproject/manifests/latest"

	assert.Equal(t, expected, path)
}

// TestParseDigest 测试解析摘要
func TestParseDigest(t *testing.T) {
	testCases := []struct {
		name     string
		digest   string
		wantAlgo string
		wantHex  string
		wantErr  bool
	}{
		{
			name:     "valid sha256",
			digest:   "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantAlgo: "sha256",
			wantHex:  "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			wantErr:  false,
		},
		{
			name:    "invalid format",
			digest:  "invalid",
			wantErr: true,
		},
		{
			name:    "unsupported algorithm",
			digest:  "md5:abc123",
			wantErr: true,
		},
		{
			name:    "invalid length",
			digest:  "sha256:abc",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			algo, hex, err := registry.ParseDigest(tc.digest)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantAlgo, algo)
				assert.Equal(t, tc.wantHex, hex)
			}
		})
	}
}

// TestParseReference 测试解析引用
func TestParseReference(t *testing.T) {
	testCases := []struct {
		name       string
		ref        string
		wantDigest bool
		wantRef    string
	}{
		{
			name:       "tag",
			ref:        "latest",
			wantDigest: false,
			wantRef:    "latest",
		},
		{
			name:       "digest",
			ref:        "sha256:abc123",
			wantDigest: true,
			wantRef:    "sha256:abc123",
		},
		{
			name:       "another tag",
			ref:        "v1.0.0",
			wantDigest: false,
			wantRef:    "v1.0.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isDigest, ref := registry.ParseReference(tc.ref)

			assert.Equal(t, tc.wantDigest, isDigest)
			assert.Equal(t, tc.wantRef, ref)
		})
	}
}

// TestParseContentRange 测试解析Content-Range头
func TestParseContentRange(t *testing.T) {
	testCases := []struct {
		name         string
		contentRange string
		wantStart    int64
		wantEnd      int64
		wantTotal    int64
		wantErr      bool
	}{
		{
			name:         "valid range",
			contentRange: "bytes 0-99/1000",
			wantStart:    0,
			wantEnd:      99,
			wantTotal:    1000,
			wantErr:      false,
		},
		{
			name:         "middle range",
			contentRange: "bytes 500-999/10000",
			wantStart:    500,
			wantEnd:      999,
			wantTotal:    10000,
			wantErr:      false,
		},
		{
			name:         "invalid format",
			contentRange: "invalid",
			wantErr:      true,
		},
		{
			name:         "missing slash",
			contentRange: "bytes 0-99",
			wantErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start, end, total, err := registry.ParseContentRange(tc.contentRange)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantStart, start)
				assert.Equal(t, tc.wantEnd, end)
				assert.Equal(t, tc.wantTotal, total)
			}
		})
	}
}

// TestCalculateDigest 测试计算摘要
func TestCalculateDigest(t *testing.T) {
	data := []byte("hello world")

	digest, size, err := registry.CalculateDigest(bytes.NewReader(data))

	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), size)
	assert.Contains(t, digest, "sha256:")
	assert.Len(t, digest, 71) // "sha256:" + 64 hex chars
}

// TestManifestStructure 测试Manifest结构
func TestManifestStructure(t *testing.T) {
	manifest := &registry.Manifest{
		SchemaVersion: 2,
		MediaType:     registry.MediaTypeDocker2Manifest,
		Config: registry.LayerInfo{
			Descriptor: registry.Descriptor{
				MediaType: registry.MediaTypeDocker2ImageConfig,
				Digest:    "sha256:config123",
				Size:      1024,
			},
		},
		Layers: []registry.LayerInfo{
			{
				Descriptor: registry.Descriptor{
					MediaType: registry.MediaTypeDocker2ImageLayer,
					Digest:    "sha256:layer123",
					Size:      10240,
				},
			},
		},
	}

	assert.Equal(t, 2, manifest.SchemaVersion)
	assert.Equal(t, registry.MediaTypeDocker2Manifest, manifest.MediaType)
	assert.Len(t, manifest.Layers, 1)
	assert.Equal(t, "sha256:config123", manifest.Config.Digest)
}

// TestMediaTypeConstants 测试媒体类型常量
func TestMediaTypeConstants(t *testing.T) {
	assert.NotEmpty(t, registry.MediaTypeDocker2Manifest)
	assert.NotEmpty(t, registry.MediaTypeDocker2ManifestList)
	assert.NotEmpty(t, registry.MediaTypeOCIManifest)
	assert.NotEmpty(t, registry.MediaTypeOCIManifestIndex)
	assert.NotEmpty(t, registry.MediaTypeDocker2ImageLayer)
	assert.NotEmpty(t, registry.MediaTypeDocker2ImageConfig)
}

// TestRegistryErrors 测试错误定义
func TestRegistryErrors(t *testing.T) {
	assert.Error(t, registry.ErrInvalidDigest)
	assert.Error(t, registry.ErrManifestNotFound)
	assert.Error(t, registry.ErrBlobNotFound)
	assert.Error(t, registry.ErrUploadNotFound)
	assert.Error(t, registry.ErrInvalidContentType)
}

// TestUploadInfo 测试上传信息结构
func TestUploadInfo(t *testing.T) {
	info := &registry.UploadInfo{
		UUID:      "test-uuid",
		Project:   "myproject",
		Reference: "latest",
		Digest:    "",
		Size:      0,
	}

	assert.Equal(t, "test-uuid", info.UUID)
	assert.Equal(t, "myproject", info.Project)
	assert.Equal(t, "latest", info.Reference)
	assert.Equal(t, int64(0), info.Size)
	assert.Empty(t, info.Digest)
}
