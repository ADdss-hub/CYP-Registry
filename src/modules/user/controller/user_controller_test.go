package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	project_dto "github.com/cyp-registry/registry/src/modules/project/dto"
	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ==================== 测试辅助函数 ====================

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestRequest(method, path string, body interface{}) (*http.Request, *httptest.ResponseRecorder) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	return req, rec
}

// ==================== 参数校验测试 ====================

func TestRegisterRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     dto.RegisterRequest
		expectError bool
	}{
		{
			name: "有效请求",
			request: dto.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
				Nickname: "Test User",
			},
			expectError: false,
		},
		{
			name: "用户名太短",
			request: dto.RegisterRequest{
				Username: "ab", // 少于3个字符
				Email:    "test@example.com",
				Password: "password123",
			},
			expectError: true,
		},
		{
			name: "无效邮箱",
			request: dto.RegisterRequest{
				Username: "testuser",
				Email:    "invalid-email",
				Password: "password123",
			},
			expectError: true,
		},
		{
			name: "密码太短",
			request: dto.RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "short", // 少于8个字符
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/register", func(c *gin.Context) {
				var req dto.RegisterRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req, rec := createTestRequest("POST", "/register", tt.request)
			router.ServeHTTP(rec, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

func TestLoginRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     dto.LoginRequest
		expectError bool
	}{
		{
			name: "有效请求",
			request: dto.LoginRequest{
				Username: "testuser",
				Password: "password123",
			},
			expectError: false,
		},
		{
			name: "缺少用户名",
			request: dto.LoginRequest{
				Password: "password123",
			},
			expectError: true,
		},
		{
			name: "缺少密码",
			request: dto.LoginRequest{
				Username: "testuser",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/login", func(c *gin.Context) {
				var req dto.LoginRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req, rec := createTestRequest("POST", "/login", tt.request)
			router.ServeHTTP(rec, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

func TestRefreshTokenRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     dto.RefreshTokenRequest
		expectError bool
	}{
		{
			name: "有效请求",
			request: dto.RefreshTokenRequest{
				RefreshToken: "valid-refresh-token",
			},
			expectError: false,
		},
		{
			name: "空RefreshToken",
			request: dto.RefreshTokenRequest{
				RefreshToken: "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/refresh", func(c *gin.Context) {
				var req dto.RefreshTokenRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req, rec := createTestRequest("POST", "/refresh", tt.request)
			router.ServeHTTP(rec, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// ==================== UpdateUserRequest 测试 ====================

func TestUpdateUserRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     dto.UpdateUserRequest
		expectError bool
	}{
		{
			name: "有效请求",
			request: dto.UpdateUserRequest{
				Nickname: "New Nickname",
				Avatar:   "https://example.com/avatar.png",
				Bio:      "This is my bio",
			},
			expectError: false,
		},
		{
			name: "昵称太长",
			request: dto.UpdateUserRequest{
				Nickname: string(make([]byte, 129)), // 超过128字符
			},
			expectError: true,
		},
		{
			name: "无效头像URL",
			request: dto.UpdateUserRequest{
				Avatar: "not-a-url",
			},
			expectError: true,
		},
		{
			name: "简介太长",
			request: dto.UpdateUserRequest{
				Bio: string(make([]byte, 501)), // 超过500字符
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.PUT("/users/me", func(c *gin.Context) {
				var req dto.UpdateUserRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req, rec := createTestRequest("PUT", "/users/me", tt.request)
			router.ServeHTTP(rec, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// ==================== ChangePasswordRequest 测试 ====================

func TestChangePasswordRequest_Validation(t *testing.T) { // 修复缺少逗号的 composite literal
	tests := []struct {
		name        string
		request     dto.ChangePasswordRequest
		expectError bool
	}{
		{
			name: "有效请求",
			request: dto.ChangePasswordRequest{ // <- 修复缺少逗号
				OldPassword: "oldpassword123",
				NewPassword: "newpassword123",
			},
			expectError: false,
		},
		{
			name: "缺少旧密码",
			request: dto.ChangePasswordRequest{
				NewPassword: "newpassword123",
			},
			expectError: true,
		},
		{
			name: "新密码太短",
			request: dto.ChangePasswordRequest{
				OldPassword: "oldpassword123",
				NewPassword: "short",
			},
			expectError: true,
		},
		{
			name: "新密码太长",
			request: dto.ChangePasswordRequest{
				OldPassword: "oldpassword123",
				NewPassword: string(make([]byte, 21)), // 超过20字符
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.PUT("/users/me/password", func(c *gin.Context) {
				var req dto.ChangePasswordRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req, rec := createTestRequest("PUT", "/users/me/password", tt.request)
			router.ServeHTTP(rec, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// ==================== CreatePATRequest 测试 ====================

func TestCreatePATRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     dto.CreatePATRequest
		expectError bool
	}{
		{
			name: "有效请求",
			request: dto.CreatePATRequest{
				Name:   "My Token",
				Scopes: []string{"read", "write"},
			},
			expectError: false,
		},
		{
			name: "名称为空",
			request: dto.CreatePATRequest{
				Name:   "",
				Scopes: []string{"read"},
			},
			expectError: true,
		},
		{
			name: "名称太长",
			request: dto.CreatePATRequest{
				Name:   string(make([]byte, 129)),
				Scopes: []string{"read"},
			},
			expectError: true,
		},
		{
			name: "Scopes为空",
			request: dto.CreatePATRequest{
				Name:   "My Token",
				Scopes: []string{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/users/me/pat", func(c *gin.Context) {
				var req dto.CreatePATRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req, rec := createTestRequest("POST", "/users/me/pat", tt.request)
			router.ServeHTTP(rec, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// ==================== 项目请求测试 ====================

func TestCreateProjectRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     project_dto.CreateProjectRequest
		expectError bool
	}{
		{
			name: "有效请求",
			request: project_dto.CreateProjectRequest{
				Name:        "my-project",
				Description: "My project description",
				IsPublic:    false,
			},
			expectError: false,
		},
		{
			name: "名称太短",
			request: project_dto.CreateProjectRequest{
				Name:     "a", // 少于2个字符
				IsPublic: false,
			},
			expectError: true,
		},
		{
			name: "描述太长",
			request: project_dto.CreateProjectRequest{
				Name:        "my-project",
				Description: string(make([]byte, 2001)), // 超过 max=2000
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupTestRouter()
			router.POST("/projects", func(c *gin.Context) {
				var req project_dto.CreateProjectRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			req, rec := createTestRequest("POST", "/projects", tt.request)
			router.ServeHTTP(rec, req)

			if tt.expectError {
				assert.Equal(t, http.StatusBadRequest, rec.Code)
			} else {
				assert.Equal(t, http.StatusOK, rec.Code)
			}
		})
	}
}

// ==================== 响应格式测试 ====================

func TestLoginResponse_Format(t *testing.T) {
	response := dto.LoginResponse{
		User: dto.UserResponse{
			ID:       testUUID(),
			Username: "testuser",
			Email:    "test@example.com",
		},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    7200,
	}

	// 序列化测试
	data, err := json.Marshal(response)
	assert.NoError(t, err)

	// 反序列化测试
	var decoded dto.LoginResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	// 验证字段
	assert.Equal(t, response.AccessToken, decoded.AccessToken)
	assert.Equal(t, response.RefreshToken, decoded.RefreshToken)
	assert.Equal(t, response.TokenType, decoded.TokenType)
	assert.Equal(t, response.ExpiresIn, decoded.ExpiresIn)
	assert.Equal(t, response.User.Username, decoded.User.Username)
}

func TestUserResponse_Format(t *testing.T) {
	response := dto.UserResponse{
		ID:        testUUID(),
		Username:  "testuser",
		Email:     "test@example.com",
		Nickname:  "Test User",
		Avatar:    "https://example.com/avatar.png",
		Bio:       "Test bio",
		IsActive:  true,
		IsAdmin:   false,
		CreatedAt: "2024-01-01T00:00:00Z",
	}

	// 序列化测试
	data, err := json.Marshal(response)
	assert.NoError(t, err)

	// 反序列化测试
	var decoded dto.UserResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	// 验证字段
	assert.Equal(t, response.ID, decoded.ID)
	assert.Equal(t, response.Username, decoded.Username)
	assert.Equal(t, response.Email, decoded.Email)
	assert.Equal(t, response.Nickname, decoded.Nickname)
	assert.Equal(t, response.IsActive, decoded.IsActive)
	assert.Equal(t, response.IsAdmin, decoded.IsAdmin)
}

func TestPageResponse_Format(t *testing.T) {
	// 测试分页响应格式 - 使用简单的 map 结构
	response := map[string]interface{}{
		"list":       []string{"item1", "item2"},
		"total":      100,
		"page":       1,
		"page_size":  20,
		"total_page": 5,
	}

	// 序列化测试
	data, err := json.Marshal(response)
	assert.NoError(t, err)

	// 反序列化测试
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	// 验证字段
	assert.Equal(t, float64(100), decoded["total"])
	assert.Equal(t, float64(1), decoded["page"])
	assert.Equal(t, float64(20), decoded["page_size"])
	assert.Equal(t, float64(5), decoded["total_page"])
}

func TestDeleteResponse_Format(t *testing.T) {
	response := dto.DeleteResponse{
		Success: true,
		Message: "删除成功",
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var decoded dto.DeleteResponse
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.True(t, decoded.Success)
	assert.Equal(t, "删除成功", decoded.Message)
}

// ==================== 辅助函数 ====================

func testUUID() uuid.UUID {
	return uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
}

// PageResponse 用于测试分页响应格式
type PageResponse struct {
	List      interface{} `json:"list"`
	Total     int64       `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	TotalPage int         `json:"total_page"`
}
