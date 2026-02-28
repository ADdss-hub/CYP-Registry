<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { userApi } from '@/services/user'
import CypButton from '@/components/common/CypButton.vue'
import CypInput from '@/components/common/CypInput.vue'
import CypFooter from '@/components/common/CypFooter.vue'
import CypDialog from '@/components/common/CypDialog.vue'
import logoCypRegistry from '@/assets/logo-cyp-registry.svg'
import { copyToClipboard } from '@/utils/clipboard'

const userStore = useUserStore()

const form = ref({
  username: '',
  password: '',
})

// “记住用户名”配置
const rememberUsername = ref(false)
const REMEMBER_USERNAME_FLAG_KEY = 'cyp-remember-username'
const REMEMBER_USERNAME_VALUE_KEY = 'cyp-remember-username-value'

const errors = ref<Record<string, string>>({})
const loginError = ref<string>('')

const isLoading = computed(() => userStore.isLoading)

// 默认管理员一次性提示
const showDefaultAdminDialog = ref(false)
const defaultAdminUsername = ref('')
const defaultAdminPassword = ref('')

function validate(): boolean {
  errors.value = {}
  loginError.value = ''

  if (!form.value.username) {
    errors.value.username = '请输入用户名'
  }

  if (!form.value.password) {
    errors.value.password = '请输入密码'
  } else if (form.value.password.length < 6) {
    errors.value.password = '密码长度至少6位'
  }

  return Object.keys(errors.value).length === 0
}

async function handleLogin() {
  if (!validate()) return

  loginError.value = ''
  try {
    await userStore.login({
      username: form.value.username,
      password: form.value.password,
    })

    // 登录成功后，根据“记住用户名”选项同步到本机浏览器
    try {
      if (rememberUsername.value && form.value.username) {
        localStorage.setItem(REMEMBER_USERNAME_FLAG_KEY, '1')
        localStorage.setItem(REMEMBER_USERNAME_VALUE_KEY, form.value.username)
      } else {
        localStorage.removeItem(REMEMBER_USERNAME_FLAG_KEY)
        localStorage.removeItem(REMEMBER_USERNAME_VALUE_KEY)
      }
    } catch {
      // 在部分受限环境下可能无法访问 localStorage，忽略此错误不影响登录流程
    }
  } catch (err: any) {
    // 显示错误消息
    loginError.value = err?.message || err?.payload?.message || '登录失败，请检查用户名和密码'
    console.error('登录失败:', err)
  }
}

// 首次加载时：
// 1. 先尝试从本机浏览器恢复“记住用户名”的设置
// 2. 再尝试获取一次性默认管理员账号信息，用于在登录页弹窗提示并复制保存。
onMounted(async () => {
  // 恢复“记住用户名”
  try {
    const flag = localStorage.getItem(REMEMBER_USERNAME_FLAG_KEY)
    const savedUsername = localStorage.getItem(REMEMBER_USERNAME_VALUE_KEY)
    if (flag === '1' && savedUsername) {
      rememberUsername.value = true
      form.value.username = savedUsername
    }
  } catch {
    // 忽略 localStorage 相关错误
  }

  // 默认管理员一次性提示
  try {
    const creds = await userApi.getDefaultAdminOnce()
    if (creds && creds.username && creds.password) {
      defaultAdminUsername.value = creds.username
      defaultAdminPassword.value = creds.password
      // 仅在当前用户名为空时才预填默认管理员用户名，避免覆盖已记住的用户名
      if (!form.value.username) {
        form.value.username = creds.username
      }
      showDefaultAdminDialog.value = true
    }
  } catch {
    // 接口不存在或已被读取时静默忽略，不影响正常登录流程
  }
})

async function handleCopyDefaultAdmin() {
  if (!defaultAdminUsername.value && !defaultAdminPassword.value) return
  const text = `默认管理员用户名：${defaultAdminUsername.value}\n默认管理员密码：${defaultAdminPassword.value}`
  try {
    await copyToClipboard(text)
    loginError.value = '默认管理员账号信息已复制，请妥善保存并尽快修改密码。'
    showDefaultAdminDialog.value = false
  } catch {
    loginError.value = '无法访问剪贴板，请手动复制弹窗中的用户名和密码。'
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-container">
      <div class="login-card">
        <div class="login-header">
          <div class="login-logo">
            <img :src="logoCypRegistry" alt="CYP-Registry Logo" width="48" height="48" />
          </div>
          <h1 class="login-title">欢迎回来</h1>
          <p class="login-subtitle">登录 CYP-Registry 镜像仓库管理平台</p>
        </div>

        <!-- 默认管理员一次性提示弹窗 -->
        <CypDialog
          v-model="showDefaultAdminDialog"
          title="默认管理员账号已生成"
          width="480px"
        >
          <p>系统检测到这是首次部署，已自动创建一个默认管理员账号，请立即复制并妥善保存：</p>
          <div class="default-admin-box">
            <div class="field">
              <span class="label">用户名（6-10 位英文+数字）：</span>
              <span class="value monospace">{{ defaultAdminUsername }}</span>
            </div>
            <div class="field">
              <span class="label">密码（10-15 位英文+数字+符号）：</span>
              <span class="value monospace">{{ defaultAdminPassword }}</span>
            </div>
            <p class="tip">
              请使用上述账号登录后，立即前往「系统设置 → 账户安全」修改密码，并根据需要创建个人账号或访问令牌。
            </p>
          </div>
          <template #footer>
            <CypButton @click="showDefaultAdminDialog = false">稍后手动复制</CypButton>
            <CypButton type="primary" style="margin-left: 8px" @click="handleCopyDefaultAdmin">
              一键复制用户名和密码
            </CypButton>
          </template>
        </CypDialog>

        <form class="login-form" @submit.prevent="handleLogin">
          <div v-if="loginError" class="login-error">
            {{ loginError }}
          </div>

          <div class="form-group">
            <label class="form-label">用户名</label>
            <CypInput
              v-model="form.username"
              type="text"
              placeholder="请输入用户名"
              :error="errors.username"
              @keyup.enter="handleLogin"
            />
          </div>

          <div class="form-group">
            <label class="form-label">密码</label>
            <CypInput
              v-model="form.password"
              type="password"
              placeholder="请输入密码"
              autocomplete="current-password"
              :error="errors.password"
              @keyup.enter="handleLogin"
            />
          </div>

          <div class="remember-row">
            <label class="remember-label">
              <input v-model="rememberUsername" type="checkbox" class="remember-checkbox" />
              <span>记住用户名（仅保存在本机浏览器中）</span>
            </label>
          </div>

          <CypButton
            type="primary"
            size="large"
            block
            :loading="isLoading"
            @click="handleLogin"
          >
            登录
          </CypButton>
        </form>

        <!-- 取消公开注册：不再显示注册链接 -->
      </div>

      <div class="login-bg">
        <div class="bg-content">
          <h2>安全可靠的容器镜像仓库</h2>
          <p>企业级容器镜像管理解决方案，提供完整的镜像存储和自动化工作流支持。</p>
          <ul class="feature-list">
            <li>
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
              </svg>
              Docker Registry API 完整兼容
            </li>
            <!-- 原“集成 Trivy 漏洞扫描”卖点已移除 -->
            <li>
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
              </svg>
              Webhook 事件通知
            </li>
            <li>
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
              </svg>
              RBAC 权限控制
            </li>
          </ul>
        </div>
      </div>
    </div>

    <!-- 统一底部信息 -->
    <CypFooter />
  </div>
</template>

<style lang="scss" scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 24px;
}

.login-container {
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

.login-card {
  flex: 1;
  padding: 48px;
  display: flex;
  flex-direction: column;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-logo {
  display: flex;
  justify-content: center;
  margin-bottom: 16px;
}

.login-title {
  font-size: 24px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 8px;
}

.login-subtitle {
  font-size: 14px;
  color: #64748b;
  margin: 0;
}

.login-form {
  flex: 1;
}

.login-error {
  padding: 12px 16px;
  margin-bottom: 24px;
  background: #fee;
  border: 1px solid #fcc;
  border-radius: 8px;
  color: #c33;
  font-size: 14px;
  text-align: center;
}

.form-group {
  margin-bottom: 24px;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  margin-bottom: 8px;
}

.hint-box {
  margin-bottom: 16px;
  padding: 10px 12px;
  border-radius: 8px;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 12px;
  line-height: 1.5;
}

.remember-row {
  margin-bottom: 16px;
}

.remember-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #4b5563;
}

.remember-checkbox {
  width: 14px;
  height: 14px;
  cursor: pointer;
}

.login-footer {
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

.login-bg {
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

// 响应式
@media (max-width: 768px) {
  .login-container {
    flex-direction: column;
  }

  .login-bg {
    display: none;
  }

  .login-card {
    padding: 32px;
  }
}
</style>

