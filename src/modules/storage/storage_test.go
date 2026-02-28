// Package storage_test 存储模块测试
package storage_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/cyp-registry/registry/src/modules/storage/driver"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestConfig 创建测试配置
func createTestConfig(t *testing.T, storagePath string) *config.Config {
	cfg := &config.Config{}
	cfg.Set("storage.local.path", storagePath)
	cfg.Set("storage.type", "local")
	return cfg
}

// TestLocalStorage_Put 测试文件上传
func TestLocalStorage_Put(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	content := []byte("hello world")

	// 测试上传文件
	err = store.Put(ctx, "test/blob", bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)

	// 验证文件存在
	exists, err := store.Exists(ctx, "test/blob")
	require.NoError(t, err)
	assert.True(t, exists)

	// 验证文件内容
	reader, size, err := store.Get(ctx, "test/blob")
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

// TestLocalStorage_Get 测试获取文件
func TestLocalStorage_Get(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	content := []byte("test content for get")

	// 先创建文件
	err = store.Put(ctx, "test/file", bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)

	// 测试获取文件
	reader, size, err := store.Get(ctx, "test/file")
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

// TestLocalStorage_Delete 测试删除文件
func TestLocalStorage_Delete(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	content := []byte("delete me")

	// 创建文件
	err = store.Put(ctx, "test/to_delete", bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)

	// 验证存在
	exists, err := store.Exists(ctx, "test/to_delete")
	require.NoError(t, err)
	assert.True(t, exists)

	// 删除文件
	err = store.Delete(ctx, "test/to_delete")
	require.NoError(t, err)

	// 验证不存在
	exists, err = store.Exists(ctx, "test/to_delete")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestLocalStorage_Exists 测试检查文件是否存在
func TestLocalStorage_Exists(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 测试不存在的文件
	exists, err := store.Exists(ctx, "test/nonexistent")
	require.NoError(t, err)
	assert.False(t, exists)

	// 创建文件
	content := []byte("exists test")
	err = store.Put(ctx, "test/exists", bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)

	// 测试存在的文件
	exists, err = store.Exists(ctx, "test/exists")
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestLocalStorage_Stat 测试获取文件信息
func TestLocalStorage_Stat(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	content := []byte("stat test content")

	// 创建文件
	err = store.Put(ctx, "test/stat", bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)

	// 获取文件信息
	size, modTime, err := store.Stat(ctx, "test/stat")
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)
	assert.NotEmpty(t, modTime)
}

// TestLocalStorage_List 测试列出文件
func TestLocalStorage_List(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 创建测试文件
	files := map[string][]byte{
		"test/dir/file1.txt": []byte("content1"),
		"test/dir/file2.txt": []byte("content2"),
		"test/dir/file3.txt": []byte("content3"),
	}

	for path, content := range files {
		err = store.Put(ctx, path, bytes.NewReader(content), int64(len(content)))
		require.NoError(t, err)
	}

	// 列出文件
	entries, err := store.List(ctx, "test/dir")
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

// TestLocalStorage_GetUsage 测试获取存储使用量
func TestLocalStorage_GetUsage(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 创建测试文件
	files := map[string][]byte{
		"usage/file1.txt": make([]byte, 100), // 100 bytes
		"usage/file2.txt": make([]byte, 200), // 200 bytes
		"usage/file3.txt": make([]byte, 300), // 300 bytes
	}

	totalSize := int64(600)
	for path, content := range files {
		err = store.Put(ctx, path, bytes.NewReader(content), int64(len(content)))
		require.NoError(t, err)
	}

	// 获取使用量
	size, count, err := store.GetUsage(ctx, "usage")
	require.NoError(t, err)
	assert.Equal(t, totalSize, size)
	assert.Equal(t, int64(3), count)
}

// TestLocalStorage_PathTraversal 防止路径遍历攻击
func TestLocalStorage_PathTraversal(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 测试路径遍历攻击
	testCases := []struct {
		name string
		path string
	}{
		{"parent dir", "../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"double dot", "test/../etc/passwd"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = store.Put(ctx, tc.path, bytes.NewReader([]byte("malicious")), 9)
			assert.Error(t, err)
			assert.ErrorIs(t, err, driver.ErrPermissionDenied)
		})
	}
}

// TestLocalStorage_Name 测试驱动名称
func TestLocalStorage_Name(t *testing.T) {
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	assert.Equal(t, "local", store.Name())
}

// TestLocalStorage_UnknownSize 测试上传未知大小的文件
func TestLocalStorage_UnknownSize(t *testing.T) {
	// 准备测试环境
	tempDir := t.TempDir()
	cfg := createTestConfig(t, tempDir)

	store, err := driver.NewLocalStorage(cfg)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	content := []byte("unknown size content")

	// 使用-1表示未知大小
	err = store.Put(ctx, "test/unknown_size", bytes.NewReader(content), -1)
	require.NoError(t, err)

	// 验证文件内容
	reader, size, err := store.Get(ctx, "test/unknown_size")
	require.NoError(t, err)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, data)
	assert.Equal(t, int64(len(content)), size)
}

// TestStorage_Interface 确保LocalStorage实现Storage接口
func TestStorage_Interface(t *testing.T) {
	// 此测试已移除，因为 storage.Storage 是包内接口
	// LocalStorage 在 driver 包中定义
}
