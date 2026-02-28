<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useUserStore } from "@/stores/user";
import CypButton from "@/components/common/CypButton.vue";
import CypInput from "@/components/common/CypInput.vue";
import CypFooter from "@/components/common/CypFooter.vue";
import CypDialog from "@/components/common/CypDialog.vue";
import logoCypRegistry from "@/assets/logo-cyp-registry.svg";

const router = useRouter();
// 说明：公开注册功能已关闭，此视图仅作为占位以避免构建错误；
// 实际路由已在 router 中重定向到 Login，不再调用 userStore.register。
const userStore = useUserStore();

const form = ref({
  username: "",
  email: "",
  password: "",
  confirmPassword: "",
});

const errors = ref<Record<string, string>>({});

// 注册成功提示框（替代浏览器 alert，遵循界面规范3.3/3.4节）
const showSuccessDialog = ref(false);
// 注册失败提示框
const showErrorDialog = ref(false);
const errorMessage = ref("");

function handleSuccessDialogClose() {
  showSuccessDialog.value = false;
  router.push("/login");
}

function handleErrorDialogClose() {
  showErrorDialog.value = false;
  errorMessage.value = "";
}

async function handleRegister() {
  // 公开注册已关闭：表单点击只弹出提示，不再校验/提交
  errorMessage.value = "当前系统已关闭公开注册，请联系管理员获取访问账号。";
  showErrorDialog.value = true;
}

function navigateToLogin() {
  router.push("/login");
}
</script>

<template>
  <div class="register-page">
    <div class="register-container">
      <div class="register-card">
        <div class="register-header">
          <div class="register-logo">
            <img
              :src="logoCypRegistry"
              alt="CYP-Registry Logo"
              width="48"
              height="48"
            />
          </div>
          <h1 class="register-title">创建账户</h1>
          <p class="register-subtitle">注册 CYP-Registry 镜像仓库管理平台</p>
        </div>

        <form class="register-form" @submit.prevent="handleRegister">
          <div class="form-group">
            <label class="form-label">用户名</label>
            <CypInput
              v-model="form.username"
              type="text"
              placeholder="请输入用户名"
              :error="errors.username"
            />
          </div>

          <div class="form-group">
            <label class="form-label">邮箱</label>
            <CypInput
              v-model="form.email"
              type="email"
              placeholder="请输入邮箱"
              :error="errors.email"
            />
          </div>

          <div class="form-group">
            <label class="form-label">密码</label>
            <CypInput
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              autocomplete="new-password"
              :error="errors.password"
            />
          </div>

          <div class="form-group">
            <label class="form-label">确认密码</label>
            <CypInput
              v-model="form.confirmPassword"
              type="password"
              placeholder="请再次输入密码"
              autocomplete="new-password"
              :error="errors.confirmPassword"
            />
          </div>

          <CypButton
            type="primary"
            size="large"
            block
            :loading="userStore.isLoading"
            @click="handleRegister"
          >
            注册
          </CypButton>
        </form>

        <div class="register-footer">
          <span>已有账号？</span>
          <a href="javascript:void(0)" @click="navigateToLogin">立即登录</a>
        </div>
      </div>

      <div class="register-bg">
        <div class="bg-content">
          <h2>加入我们</h2>
          <p>创建账户，开始使用企业级容器镜像管理解决方案。</p>
          <ul class="feature-list">
            <li>
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                />
              </svg>
              无限项目创建
            </li>
            <li>
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"
                />
              </svg>
              访问令牌与项目权限控制
            </li>
            <!-- 原“安全漏洞扫描”卖点已移除，避免出现漏洞相关表述 -->
          </ul>
        </div>
      </div>

      <!-- 注册成功系统框 -->
      <CypDialog
        v-model="showSuccessDialog"
        title="注册成功"
        width="420px"
        @close="handleSuccessDialogClose"
      >
        <p>注册成功，请使用新账号登录。</p>
        <template #footer>
          <CypButton @click="handleSuccessDialogClose"> 稍后再说 </CypButton>
          <CypButton type="primary" @click="handleSuccessDialogClose">
            去登录
          </CypButton>
        </template>
      </CypDialog>

      <!-- 注册失败系统框 -->
      <CypDialog
        v-model="showErrorDialog"
        title="注册失败"
        width="420px"
        @close="handleErrorDialogClose"
      >
        <p>{{ errorMessage }}</p>
        <template #footer>
          <CypButton type="primary" @click="handleErrorDialogClose">
            知道了
          </CypButton>
        </template>
      </CypDialog>
    </div>

    <!-- 统一底部信息 -->
    <CypFooter />
  </div>
</template>

<style lang="scss" scoped>
.register-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 24px;
}

.register-container {
  flex: 1;
  display: flex;
  width: 100%;
  max-width: 1000px;
  margin: 0 auto;
  background: white;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.register-card {
  flex: 1;
  padding: 48px;
  display: flex;
  flex-direction: column;
}

.register-header {
  text-align: center;
  margin-bottom: 32px;
}

.register-logo {
  display: flex;
  justify-content: center;
  margin-bottom: 16px;
}

.register-title {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 8px;
}

.register-subtitle {
  font-size: 14px;
  color: #64748b;
  margin: 0;
}

.register-form {
  flex: 1;
}

.form-group {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  margin-bottom: 8px;
}

.register-footer {
  text-align: center;
  font-size: 14px;
  color: #64748b;
  margin-top: 24px;

  a {
    color: #6366f1;
    font-weight: 500;
    margin-left: 4px;
    cursor: pointer;

    &:hover {
      text-decoration: underline;
    }
  }
}

.register-bg {
  flex: 1;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
}

.bg-content {
  color: white;
  text-align: center;

  h2 {
    font-size: 28px;
    font-weight: 600;
    margin: 0 0 16px;
  }

  p {
    font-size: 15px;
    opacity: 0.9;
    line-height: 1.7;
    margin: 0 0 32px;
  }
}

.feature-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 16px;
  text-align: left;

  li {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 14px;
  }

  svg {
    flex-shrink: 0;
    opacity: 0.9;
  }
}

@media (max-width: 768px) {
  .register-container {
    flex-direction: column;
  }

  .register-bg {
    display: none;
  }

  .register-card {
    padding: 32px;
  }
}
</style>
