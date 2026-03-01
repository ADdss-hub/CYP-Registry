package service

import (
	"testing"

	"github.com/cyp-registry/registry/src/pkg/config"
)

// TestNewService_Smoke 仅做一次构造函数的冒烟测试，确保基础依赖注入不 panic。
func TestNewService_Smoke(t *testing.T) {
	jwtCfg := &config.JWTConfig{}
	patCfg := &config.PATConfig{}

	svc := NewService(jwtCfg, patCfg, 10)
	if svc == nil {
		t.Fatalf("expected non-nil Service instance")
	}
}
