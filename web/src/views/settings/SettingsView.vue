<script setup lang="ts">
import { ref, watch } from "vue";
import { useUserStore } from "@/stores/user";
import { useThemeStore } from "@/stores/theme";
import { useNotificationStore } from "@/stores/notification";
import CypButton from "@/components/common/CypButton.vue";
import CypInput from "@/components/common/CypInput.vue";
import CypSelect from "@/components/common/CypSelect.vue";
import CypSwitch from "@/components/common/CypSwitch.vue";
import CypDialog from "@/components/common/CypDialog.vue";
import type { AccessToken, NotificationSettings } from "@/types";
import { copyToClipboard } from "@/utils/clipboard";
import { adminApi } from "@/services/admin";
import type { SystemConfig, UpdateSystemConfigRequest } from "@/services/admin";

const userStore = useUserStore();
const themeStore = useThemeStore();
const notificationStore = useNotificationStore();

const activeSection = ref("profile");

// 个人资料表单
const profileForm = ref({
  username: userStore.user?.username || "",
  email: userStore.user?.email || "",
  bio: userStore.user?.bio || "",
  avatar: userStore.user?.avatar || "",
});

// 监听 userStore.user 的变化，同步更新 profileForm
watch(
  () => userStore.user,
  (newUser) => {
    if (newUser) {
      profileForm.value.username = newUser.username || "";
      profileForm.value.email = newUser.email || "";
      profileForm.value.bio = newUser.bio || "";
      // 只有当新头像URL与当前不同时才更新，避免覆盖用户正在编辑的内容
      if (newUser.avatar && newUser.avatar !== profileForm.value.avatar) {
        profileForm.value.avatar = newUser.avatar;
      }
    }
  },
  { deep: true, immediate: true },
);

// 安全设置表单
const securityForm = ref({
  currentPassword: "",
  newPassword: "",
  confirmPassword: "",
});

// 通知设置（默认开启 Webhook 通知，频率为实时）
const notificationSettings = ref<{
  emailEnabled: boolean;
  scanCompleted: boolean;
  securityAlerts: boolean;
  webhookNotifications: boolean;
  digest: NotificationSettings["digest"];
  notificationEmail: string;
}>({
  emailEnabled: true,
  scanCompleted: true,
  securityAlerts: true,
  webhookNotifications: true,
  digest: "realtime",
  notificationEmail: "",
});

// 偏好设置
const language = ref("zh-CN");
const timezone = ref("Asia/Shanghai");

// 系统配置（仅管理员）
const systemConfig = ref<SystemConfig>({
  https: {
    enabled: false,
    ssl_certificate_path: "",
    ssl_certificate_key_path: "",
    ssl_protocols: ["TLSv1.2", "TLSv1.3"],
    http_redirect: true,
  },
  cors: {
    allowed_origins: [],
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
    allowed_headers: ["Authorization", "Content-Type", "X-Requested-With"],
  },
  rate_limit: {
    enabled: true,
    requests_per_second: 100,
    burst: 200,
  },
});
const systemConfigLoaded = ref(false);

// 访问令牌
const tokens = ref<AccessToken[]>([]);
const tokensLoaded = ref(false);

// 对话框状态
const showCreateTokenDialog = ref(false);
const showDeleteConfirmDialog = ref(false);
const selectedToken = ref<AccessToken | null>(null);
const showTokenDialog = ref(false);
const createdToken = ref("");
const createdTokenType = ref("");

// 头像编辑系统框（文件上传方式，遵循《全平台通用界面开发设计规范》3.4节）
const showAvatarDialog = ref(false);
const avatarFile = ref<File | null>(null);
const avatarLoadFailed = ref(false);

// 通用提示框（信息/错误），临时代替全局 Toast，遵循3.3节“提示框组件”文案规范
const showMessageDialog = ref(false);
const messageDialogTitle = ref("");
const messageDialogContent = ref("");

function openMessageDialog(title: string, content: string) {
  messageDialogTitle.value = title;
  messageDialogContent.value = content;
  showMessageDialog.value = true;
}

const newTokenForm = ref<{
  name: string;
  expiresAt: string;
  scopes: string[];
}>({
  name: "",
  expiresAt: "",
  scopes: [] as string[],
});

const scopeOptions = [
  { value: "read", label: "读取" },
  { value: "write", label: "写入" },
  { value: "delete", label: "删除" },
  { value: "admin", label: "管理" },
];

const scopeLabelMap: Record<string, string> = scopeOptions.reduce(
  (acc, item) => {
    acc[item.value] = item.label;
    return acc;
  },
  {} as Record<string, string>,
);

const expiryOptions = [
  { value: "", label: "永不过期" },
  { value: "7", label: "7天" },
  { value: "30", label: "30天" },
  { value: "90", label: "90天" },
  { value: "365", label: "1年" },
];

// 访问令牌列表按需加载：仅在首次进入“访问令牌”页签时从后端拉取
async function loadTokensIfNeeded() {
  if (tokensLoaded.value) return;
  try {
    const list = await userStore.listPAT();
    tokens.value = list;
    tokensLoaded.value = true;
  } catch (err: any) {
    openMessageDialog("加载失败", err?.message || "加载访问令牌列表失败");
  }
}

// 加载通知设置（进入“通知设置”页签时从后端拉取）
async function loadNotificationSettings() {
  try {
    const settings = await userStore.getNotificationSettings();
    notificationSettings.value = {
      emailEnabled: settings.email_enabled,
      scanCompleted: settings.scan_completed,
      securityAlerts: settings.security_alerts,
      webhookNotifications: settings.webhook_notifications,
      digest: settings.digest,
      notificationEmail: settings.notification_email || "",
    };
  } catch (err: any) {
    openMessageDialog("加载失败", err?.message || "加载通知设置失败");
  }
}

// 加载系统配置（进入"系统配置"页签时从后端拉取）
async function loadSystemConfig() {
  if (!userStore.isAdmin) return;
  if (systemConfigLoaded.value) return;
  try {
    const config = await adminApi.getSystemConfig();
    systemConfig.value = config;
    systemConfigLoaded.value = true;
  } catch (err: any) {
    openMessageDialog("加载失败", err?.message || "加载系统配置失败");
  }
}

watch(
  () => activeSection.value,
  (section) => {
    if (section === "tokens") {
      loadTokensIfNeeded();
    }
    if (section === "notifications") {
      loadNotificationSettings();
    }
    if (section === "system" && userStore.isAdmin) {
      loadSystemConfig();
    }
  },
  { immediate: true },
);

// 保存个人资料
async function saveProfile() {
  try {
    await userStore.updateCurrentUser({
      nickname: profileForm.value.username,
      avatar: profileForm.value.avatar,
      bio: profileForm.value.bio,
    });
    await userStore.fetchCurrentUser();
    openMessageDialog("保存成功", "个人资料已保存");
    notificationStore.addNotification({
      source: "system",
      title: "个人资料已更新",
      message: "您的用户名、头像或个人简介等资料已成功保存",
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("保存失败", err?.message || "个人资料保存失败");
  }
}

// 更换头像（使用系统框代替浏览器 prompt）
function changeAvatar() {
  avatarFile.value = null;
  showAvatarDialog.value = true;
}

function onAvatarFileChange(e: Event) {
  const target = e.target as HTMLInputElement;
  const file = target.files?.[0] || null;
  avatarFile.value = file;
}

// 保存头像（在头像弹窗中直接上传文件并更新，避免用户额外再点"保存更改"）
async function saveAvatarFromDialog() {
  if (!avatarFile.value) {
    openMessageDialog("校验失败", "请选择要上传的头像图片文件");
    return;
  }
  try {
    avatarLoadFailed.value = false;
    const updated = await userStore.uploadAvatar(avatarFile.value);
    // 更新 profileForm 中的头像URL
    if (updated && updated.avatar) {
      profileForm.value.avatar = updated.avatar;
    }
    // 强制刷新用户信息，确保头像URL是最新的
    await userStore.fetchCurrentUser();
    // 再次更新 profileForm，确保使用最新的用户信息
    if (userStore.user?.avatar) {
      profileForm.value.avatar = userStore.user.avatar;
    }
    openMessageDialog("保存成功", "头像已更新");
    showAvatarDialog.value = false;
    // 清空文件选择
    avatarFile.value = null;
  } catch (err: any) {
    openMessageDialog("保存失败", err?.message || "头像更新失败");
  }
}

// 修改密码
function changePassword() {
  if (securityForm.value.newPassword !== securityForm.value.confirmPassword) {
    openMessageDialog("校验失败", "两次输入的密码不一致");
    return;
  }
  if (securityForm.value.newPassword.length < 8) {
    openMessageDialog("校验失败", "密码长度至少为8个字符");
    return;
  }
  console.log("修改密码:", securityForm.value);
  openMessageDialog("修改成功", "密码已修改");
  notificationStore.addNotification({
    source: "system",
    title: "密码已修改",
    message: "您的登录密码已更新，如非本人操作请尽快联系管理员",
    status: "success",
  });
  securityForm.value = {
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  };
}

// 保存通知设置（调用后端接口）
async function saveNotificationSettings() {
  try {
    // 简单邮箱格式校验（仅在配置了独立通知邮箱时校验）
    if (notificationSettings.value.notificationEmail) {
      const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!emailPattern.test(notificationSettings.value.notificationEmail)) {
        openMessageDialog("校验失败", "通知邮箱格式不正确，请检查后再保存");
        return;
      }
    }

    const payload: NotificationSettings = {
      email_enabled: notificationSettings.value.emailEnabled,
      scan_completed: notificationSettings.value.scanCompleted,
      security_alerts: notificationSettings.value.securityAlerts,
      webhook_notifications: notificationSettings.value.webhookNotifications,
      digest: notificationSettings.value.digest,
      notification_email: notificationSettings.value.notificationEmail,
    };
    await userStore.updateNotificationSettings(payload);
    openMessageDialog("保存成功", "通知设置已保存");
    notificationStore.addNotification({
      source: "system",
      title: "通知设置已更新",
      message: "您的通知偏好与邮箱设置已保存",
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("保存失败", err?.message || "通知设置保存失败");
  }
}

// 创建令牌
function openCreateTokenDialog() {
  newTokenForm.value = {
    name: "",
    expiresAt: "",
    scopes: ["read"],
  };
  showCreateTokenDialog.value = true;
}

async function handleCreateToken() {
  if (!newTokenForm.value.name) {
    openMessageDialog("校验失败", "请输入令牌名称");
    return;
  }

  const expireDays = newTokenForm.value.expiresAt
    ? Number(newTokenForm.value.expiresAt)
    : undefined;

  try {
    const result = await userStore.createPAT({
      name: newTokenForm.value.name,
      scopes: newTokenForm.value.scopes,
      expireInDays: expireDays,
    });

    tokens.value.unshift(result.accessToken);
    showCreateTokenDialog.value = false;

    // 显示新令牌（只显示一次）
    createdToken.value = result.token;
    createdTokenType.value = result.tokenType;
    showTokenDialog.value = true;

    const translatedScopes =
      newTokenForm.value.scopes.length > 0
        ? newTokenForm.value.scopes.map((s) => scopeLabelMap[s] || s).join("，")
        : "默认读取";

    notificationStore.addNotification({
      source: "system",
      title: "访问令牌已创建",
      message: `已创建访问令牌「${newTokenForm.value.name}」，权限范围：${translatedScopes}`,
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("创建失败", err?.message || "创建令牌失败");
  }
}

function openDeleteConfirmDialog(token: AccessToken) {
  selectedToken.value = token;
  showDeleteConfirmDialog.value = true;
}

async function handleDeleteToken() {
  if (!selectedToken.value) return;

  try {
    await userStore.revokePAT(selectedToken.value.id);
    tokens.value = tokens.value.filter(
      (t: AccessToken) => t.id !== selectedToken.value!.id,
    );
    showDeleteConfirmDialog.value = false;
    openMessageDialog("删除成功", "访问令牌已删除");
    notificationStore.addNotification({
      source: "system",
      title: "访问令牌已删除",
      message: `访问令牌「${selectedToken.value.name}」已被删除`,
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("删除失败", err?.message || "删除访问令牌失败");
  } finally {
    selectedToken.value = null;
  }
}

async function copyToken(token: string) {
  try {
    if (!token) {
      openMessageDialog("复制失败", "当前没有可复制的令牌值");
      return;
    }

    await copyToClipboard(token);

    openMessageDialog("复制成功", "令牌已复制到剪贴板");
    notificationStore.addNotification({
      source: "system",
      title: "访问令牌已复制",
      message: "访问令牌值已复制到剪贴板",
      status: "success",
    });
  } catch (err: any) {
    console.error("复制令牌到剪贴板失败", err);
    openMessageDialog(
      "复制失败",
      err?.message || "复制令牌到剪贴板失败，请尝试手动选择并复制",
    );
  }
}

// 将令牌的权限范围转换为中文展示文案
function formatTokenScopes(scopes?: string[]): string {
  if (!scopes || scopes.length === 0) {
    return "默认读取";
  }
  return scopes.map((s) => scopeLabelMap[s] || s).join("，");
}

// 获取完整的头像URL（添加时间戳防止缓存）
function getAvatarUrl(avatar: string): string {
  if (!avatar) return "";
  // 如果已经是完整URL（包含 http:// 或 https://），添加时间戳
  if (avatar.startsWith("http://") || avatar.startsWith("https://")) {
    const separator = avatar.includes("?") ? "&" : "?";
    return avatar + separator + "t=" + Date.now();
  }
  // 如果是相对路径，添加时间戳防止缓存
  if (avatar.startsWith("/")) {
    return avatar + "?t=" + Date.now();
  }
  // 其他情况直接返回，但添加时间戳
  return avatar + (avatar.includes("?") ? "&" : "?") + "t=" + Date.now();
}

// 处理头像加载错误
function handleAvatarError() {
  // 只标记失败，避免直接操作 DOM 样式导致后续成功加载也一直不显示
  avatarLoadFailed.value = true;
}

// 保存系统配置
async function saveSystemConfig() {
  if (!userStore.isAdmin) {
    openMessageDialog("权限不足", "仅管理员可以修改系统配置");
    return;
  }
  try {
    const updateReq: UpdateSystemConfigRequest = {
      cors: {
        allowed_origins: systemConfig.value.cors.allowed_origins,
        allowed_methods: systemConfig.value.cors.allowed_methods,
        allowed_headers: systemConfig.value.cors.allowed_headers,
      },
      rate_limit: {
        enabled: systemConfig.value.rate_limit.enabled,
        requests_per_second: systemConfig.value.rate_limit.requests_per_second,
        burst: systemConfig.value.rate_limit.burst,
      },
    };
    await adminApi.updateSystemConfig(updateReq);
    openMessageDialog("保存成功", "系统配置已保存");
    notificationStore.addNotification({
      source: "system",
      title: "系统配置已更新",
      message: "系统配置已成功保存，部分配置可能需要重启服务生效",
      status: "success",
    });
  } catch (err: any) {
    openMessageDialog("保存失败", err?.message || "系统配置保存失败");
  }
}

// 添加CORS来源
function addCorsOrigin() {
  const origin = prompt("请输入允许的来源（例如：https://example.com）");
  if (origin && origin.trim()) {
    if (!systemConfig.value.cors.allowed_origins.includes(origin.trim())) {
      systemConfig.value.cors.allowed_origins.push(origin.trim());
    }
  }
}

// 删除CORS来源
function removeCorsOrigin(index: number) {
  systemConfig.value.cors.allowed_origins.splice(index, 1);
}
</script>

<template>
  <div class="settings-page">
    <div class="page-header">
      <h2 class="page-title">系统设置</h2>
      <p class="page-subtitle">管理您的账户和系统配置</p>
    </div>

    <div class="settings-layout">
      <nav class="settings-nav">
        <button
          class="nav-item"
          :class="{ active: activeSection === 'profile' }"
          @click="activeSection = 'profile'"
        >
          <svg viewBox="0 0 24 24" width="18" height="18">
            <path
              fill="currentColor"
              d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"
            />
          </svg>
          个人资料
        </button>
        <button
          class="nav-item"
          :class="{ active: activeSection === 'security' }"
          @click="activeSection = 'security'"
        >
          <svg viewBox="0 0 24 24" width="18" height="18">
            <path
              fill="currentColor"
              d="M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4zm0 10.99h7c-.53 4.12-3.28 7.79-7 8.94V12H5V6.3l7-3.11v8.8z"
            />
          </svg>
          安全设置
        </button>
        <button
          class="nav-item"
          :class="{ active: activeSection === 'notifications' }"
          @click="activeSection = 'notifications'"
        >
          <svg viewBox="0 0 24 24" width="18" height="18">
            <path
              fill="currentColor"
              d="M12 22c1.1 0 2-.9 2-2h-4c0 1.1.89 2 2 2zm6-6v-5c0-3.07-1.64-5.64-4.5-6.32V4c0-.83-.67-1.5-1.5-1.5s-1.5.67-1.5 1.5v.68C7.63 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z"
            />
          </svg>
          通知设置
        </button>
        <button
          class="nav-item"
          :class="{ active: activeSection === 'tokens' }"
          @click="activeSection = 'tokens'"
        >
          <svg viewBox="0 0 24 24" width="18" height="18">
            <path
              fill="currentColor"
              d="M21 10h-8.35C11.83 7.67 9.61 6 7 6c-3.31 0-6 2.69-6 6s2.69 6 6 6c2.61 0 4.83-1.67 5.65-4H13l2 2 2-2 2 2 4-4.04L21 10zM7 15c-1.65 0-3-1.35-3-3s1.35-3 3-3 3 1.35 3 3-1.35 3-3 3z"
            />
          </svg>
          访问令牌
        </button>
        <button
          class="nav-item"
          :class="{ active: activeSection === 'appearance' }"
          @click="activeSection = 'appearance'"
        >
          <svg viewBox="0 0 24 24" width="18" height="18">
            <path
              fill="currentColor"
              d="M12 3c-4.97 0-9 4.03-9 9s4.03 9 9 9c.83 0 1.5-.67 1.5-1.5 0-.39-.15-.74-.39-1.01-.23-.26-.38-.61-.38-.99 0-.83.67-1.5 1.5-1.5H16c2.76 0 5-2.24 5-5 0-4.42-4.03-8-9-8zm-5.5 9c-.83 0-1.5-.67-1.5-1.5S5.67 9 6.5 9 8 9.67 8 10.5 7.33 12 6.5 12zm3-4C8.67 8 8 7.33 8 6.5S8.67 5 9.5 5s1.5.67 1.5 1.5S10.33 8 9.5 8zm5 0c-.83 0-1.5-.67-1.5-1.5S13.67 5 14.5 5s1.5.67 1.5 1.5S15.33 8 14.5 8zm3 4c-.83 0-1.5-.67-1.5-1.5S16.67 9 17.5 9s1.5.67 1.5 1.5-.67 1.5-1.5 1.5z"
            />
          </svg>
          外观设置
        </button>
        <button
          v-if="userStore.isAdmin"
          class="nav-item"
          :class="{ active: activeSection === 'system' }"
          @click="activeSection = 'system'"
        >
          <svg viewBox="0 0 24 24" width="18" height="18">
            <path
              fill="currentColor"
              d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.06-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94L14.4 2.81c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.3-.07.63-.07.94s.02.64.06.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"
            />
          </svg>
          系统配置
        </button>
      </nav>

      <div class="settings-content">
        <!-- 个人资料 -->
        <section v-if="activeSection === 'profile'" class="settings-section">
          <h2>个人资料</h2>
          <div class="profile-avatar">
            <div class="avatar-preview">
              <img
                v-if="
                  (profileForm.avatar || userStore.user?.avatar) &&
                  !avatarLoadFailed
                "
                :key="profileForm.avatar || userStore.user?.avatar"
                :src="
                  getAvatarUrl(
                    profileForm.avatar || userStore.user?.avatar || '',
                  )
                "
                alt="avatar"
                @error="handleAvatarError"
              />
              <span v-else>
                {{
                  (profileForm.username || userStore.user?.username || "U")
                    .charAt(0)
                    .toUpperCase()
                }}
              </span>
            </div>
            <CypButton size="small" @click="changeAvatar"> 更换头像 </CypButton>
          </div>
          <div class="form-group">
            <label>用户名</label>
            <CypInput v-model="profileForm.username" placeholder="用户名" />
          </div>
          <div class="form-group">
            <label>邮箱</label>
            <CypInput
              v-model="profileForm.email"
              type="email"
              placeholder="邮箱地址"
            />
          </div>
          <div class="form-group">
            <label>个人简介</label>
            <textarea
              v-model="profileForm.bio"
              class="textarea"
              placeholder="介绍一下自己"
              rows="3"
            />
          </div>
          <CypButton type="primary" @click="saveProfile"> 保存更改 </CypButton>
        </section>

        <!-- 安全设置 -->
        <section v-if="activeSection === 'security'" class="settings-section">
          <h2>安全设置</h2>
          <div class="form-group">
            <label>当前密码</label>
            <CypInput
              v-model="securityForm.currentPassword"
              type="password"
              placeholder="请输入当前密码"
              autocomplete="current-password"
            />
          </div>
          <div class="form-group">
            <label>新密码</label>
            <CypInput
              v-model="securityForm.newPassword"
              type="password"
              placeholder="请输入新密码（至少8个字符）"
              autocomplete="new-password"
            />
          </div>
          <div class="form-group">
            <label>确认新密码</label>
            <CypInput
              v-model="securityForm.confirmPassword"
              type="password"
              placeholder="请再次输入新密码"
              autocomplete="new-password"
            />
          </div>
          <CypButton type="primary" @click="changePassword">
            修改密码
          </CypButton>

          <div class="danger-zone">
            <h3>危险操作</h3>
            <p>删除账户将永久移除您的所有数据和访问权限。此操作无法撤销。</p>
            <CypButton type="danger"> 删除账户 </CypButton>
          </div>
        </section>

        <!-- 通知设置 -->
        <section
          v-if="activeSection === 'notifications'"
          class="settings-section"
        >
          <h2>通知设置</h2>
          <div class="toggle-group">
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">邮件通知</span>
                <span class="toggle-desc"
                  >接收系统发送的邮件通知，可配置单独的通知邮箱</span
                >
              </div>
              <CypSwitch v-model="notificationSettings.emailEnabled" />
            </div>
            <div class="form-group" style="margin-top: 8px">
              <label>通知邮箱（可选）</label>
              <CypInput
                v-model="notificationSettings.notificationEmail"
                type="email"
                placeholder="不填写则默认使用账户邮箱：{{ profileForm.email || userStore.user?.email }}"
              />
            </div>
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">扫描完成通知</span>
                <span class="toggle-desc">当安全扫描任务完成时发送通知</span>
              </div>
              <CypSwitch v-model="notificationSettings.scanCompleted" />
            </div>
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">安全告警</span>
                <span class="toggle-desc">当发现严重安全风险时立即通知</span>
              </div>
              <CypSwitch v-model="notificationSettings.securityAlerts" />
            </div>
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">Webhook通知</span>
                <span class="toggle-desc">通过Webhook发送事件通知</span>
              </div>
              <CypSwitch v-model="notificationSettings.webhookNotifications" />
            </div>
          </div>

          <div class="form-group" style="margin-top: 24px">
            <label>通知频率</label>
            <CypSelect
              v-model="notificationSettings.digest"
              :options="[
                { value: 'realtime', label: '实时' },
                { value: 'daily', label: '每日摘要' },
                { value: 'weekly', label: '每周摘要' },
              ]"
            />
          </div>

          <CypButton type="primary" @click="saveNotificationSettings">
            保存设置
          </CypButton>
        </section>

        <!-- 访问令牌 -->
        <section v-if="activeSection === 'tokens'" class="settings-section">
          <h2>访问令牌</h2>
          <p class="section-desc">
            创建和管理用于API访问的令牌。令牌具有与您的账户相同的权限。
          </p>
          <CypButton
            type="primary"
            data-testid="create-token-button"
            @click="openCreateTokenDialog"
          >
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              style="margin-right: 6px"
            >
              <path
                fill="currentColor"
                d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"
              />
            </svg>
            创建新令牌
          </CypButton>

          <div class="token-list">
            <div v-for="token in tokens" :key="token.id" class="token-item">
              <div class="token-info">
                <span class="token-name">{{ token.name }}</span>
                <span class="token-value"
                  >{{ token.token.substring(0, 12) }}...</span
                >
                <div class="token-meta">
                  <span>创建于 {{ token.createdAt }}</span>
                  <span v-if="token.lastUsedAt"
                    >，最后使用于 {{ token.lastUsedAt }}</span
                  >
                  <span v-if="token.expiresAt"
                    >，有效期至 {{ token.expiresAt }}</span
                  >
                  <span v-else>，有效期：不过期</span>
                  <div class="token-scopes">
                    权限范围：{{ formatTokenScopes(token.scopes) }}
                  </div>
                </div>
              </div>
              <div class="token-actions">
                <CypButton size="small" @click="copyToken(token.token)">
                  复制
                </CypButton>
                <CypButton
                  size="small"
                  type="danger"
                  @click="openDeleteConfirmDialog(token)"
                >
                  删除
                </CypButton>
              </div>
            </div>
          </div>
        </section>

        <!-- 外观设置 -->
        <section v-if="activeSection === 'appearance'" class="settings-section">
          <h2>外观设置</h2>
          <div class="theme-selector">
            <h3>主题</h3>
            <div class="theme-options">
              <label
                class="theme-option light"
                :class="{ selected: themeStore.theme === 'light' }"
              >
                <input
                  type="radio"
                  name="theme"
                  value="light"
                  :checked="themeStore.theme === 'light'"
                  @change="themeStore.setTheme('light')"
                />
                <span class="theme-preview light" />
                <span class="theme-label">浅色</span>
              </label>
              <label
                class="theme-option dark"
                :class="{ selected: themeStore.theme === 'dark' }"
              >
                <input
                  type="radio"
                  name="theme"
                  value="dark"
                  data-testid="dark-theme-option"
                  :checked="themeStore.theme === 'dark'"
                  @change="themeStore.setTheme('dark')"
                />
                <span class="theme-preview dark" />
                <span class="theme-label">深色</span>
              </label>
              <label
                class="theme-option auto"
                :class="{ selected: themeStore.theme === 'auto' }"
              >
                <input
                  type="radio"
                  name="theme"
                  value="auto"
                  :checked="themeStore.theme === 'auto'"
                  @change="themeStore.setTheme('auto')"
                />
                <span class="theme-preview auto" />
                <span class="theme-label">跟随系统</span>
              </label>
            </div>
          </div>

          <div class="form-group">
            <label>语言</label>
            <CypSelect
              v-model="language"
              :options="[
                { value: 'zh-CN', label: '简体中文' },
                { value: 'en-US', label: 'English' },
              ]"
            />
          </div>

          <div class="form-group">
            <label>时区</label>
            <CypSelect
              v-model="timezone"
              :options="[
                { value: 'Asia/Shanghai', label: 'Asia/Shanghai (UTC+8)' },
                { value: 'UTC', label: 'UTC' },
              ]"
            />
          </div>
        </section>

        <!-- 系统配置（仅管理员） -->
        <section
          v-if="activeSection === 'system' && userStore.isAdmin"
          class="settings-section"
        >
          <h2>系统配置</h2>
          <p class="section-desc">
            管理系统级别的配置，包括
            HTTPS/SSL、安全策略等。修改后可能需要重启服务才能生效。
          </p>

          <!-- HTTPS/SSL 配置 -->
          <div class="config-group">
            <h3>HTTPS/SSL 配置</h3>
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">启用 HTTPS</span>
                <span class="toggle-desc"
                  >通过反向代理（如 Nginx）启用 HTTPS 访问</span
                >
              </div>
              <CypSwitch v-model="systemConfig.https.enabled" />
            </div>
            <div v-if="systemConfig.https.enabled" class="form-group">
              <label>SSL 证书路径</label>
              <CypInput
                v-model="systemConfig.https.ssl_certificate_path"
                placeholder="/etc/ssl/certs/registry.example.com.crt"
              />
              <small class="form-hint"
                >SSL
                证书文件路径（在反向代理服务器上，仅展示，需手动配置Nginx）</small
              >
            </div>
            <div v-if="systemConfig.https.enabled" class="form-group">
              <label>SSL 私钥路径</label>
              <CypInput
                v-model="systemConfig.https.ssl_certificate_key_path"
                placeholder="/etc/ssl/private/registry.example.com.key"
              />
              <small class="form-hint"
                >SSL
                私钥文件路径（在反向代理服务器上，仅展示，需手动配置Nginx）</small
              >
            </div>
            <div v-if="systemConfig.https.enabled" class="form-group">
              <label>HTTP 自动重定向到 HTTPS</label>
              <CypSwitch v-model="systemConfig.https.http_redirect" />
              <small class="form-hint"
                >启用后，所有 HTTP 请求将自动重定向到
                HTTPS（需在Nginx配置）</small
              >
            </div>
          </div>

          <!-- CORS 配置 -->
          <div class="config-group">
            <h3>CORS 配置</h3>
            <div class="form-group">
              <label>允许的来源</label>
              <div class="cors-origins-list">
                <div
                  v-for="(origin, index) in systemConfig.cors.allowed_origins"
                  :key="index"
                  class="cors-origin-item"
                >
                  <span>{{ origin }}</span>
                  <CypButton
                    size="small"
                    type="danger"
                    @click="removeCorsOrigin(index)"
                  >
                    删除
                  </CypButton>
                </div>
                <CypButton size="small" @click="addCorsOrigin">
                  + 添加来源
                </CypButton>
              </div>
              <small class="form-hint"
                >配置允许跨域访问的来源，支持多个域名</small
              >
            </div>
          </div>

          <!-- 速率限制配置 -->
          <div class="config-group">
            <h3>速率限制</h3>
            <div class="toggle-item">
              <div class="toggle-info">
                <span class="toggle-label">启用速率限制</span>
                <span class="toggle-desc">限制 API 请求频率，防止滥用</span>
              </div>
              <CypSwitch v-model="systemConfig.rate_limit.enabled" />
            </div>
            <div v-if="systemConfig.rate_limit.enabled" class="form-group">
              <label>每秒请求数</label>
              <CypInput
                v-model.number="systemConfig.rate_limit.requests_per_second"
                type="number"
                placeholder="100"
              />
              <small class="form-hint">允许的每秒请求数上限</small>
            </div>
            <div v-if="systemConfig.rate_limit.enabled" class="form-group">
              <label>突发请求数</label>
              <CypInput
                v-model.number="systemConfig.rate_limit.burst"
                type="number"
                placeholder="200"
              />
              <small class="form-hint">允许的突发请求数上限</small>
            </div>
          </div>

          <div class="config-warning">
            <svg viewBox="0 0 24 24" width="20" height="20">
              <path
                fill="currentColor"
                d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
              />
            </svg>
            <div>
              <strong>注意：</strong
              >修改系统配置后，部分配置可能需要重启服务或反向代理才能生效。请谨慎操作。
            </div>
          </div>

          <CypButton type="primary" @click="saveSystemConfig">
            保存配置
          </CypButton>
        </section>
      </div>
    </div>

    <!-- 创建令牌对话框（系统框组件） -->
    <CypDialog
      v-model="showCreateTokenDialog"
      title="创建访问令牌"
      width="500px"
      @close="showCreateTokenDialog = false"
    >
      <div class="dialog-form">
        <div class="form-group">
          <label>令牌名称 *</label>
          <CypInput
            v-model="newTokenForm.name"
            placeholder="例如: Production API Token"
            data-testid="token-name-input"
          />
        </div>
        <div class="form-group">
          <label>有效期</label>
          <CypSelect
            v-model="newTokenForm.expiresAt"
            :options="expiryOptions"
            data-testid="expiry-select"
          />
        </div>
        <div class="form-group">
          <label>权限范围</label>
          <div class="scope-options">
            <label
              v-for="scope in scopeOptions"
              :key="scope.value"
              class="scope-option"
              :class="{ selected: newTokenForm.scopes.includes(scope.value) }"
            >
              <input
                type="checkbox"
                :checked="newTokenForm.scopes.includes(scope.value)"
                :data-testid="`scope-${scope.value}`"
                @change="
                  (e) => {
                    const checked = (e.target as HTMLInputElement).checked;
                    if (checked) {
                      newTokenForm.scopes.push(scope.value);
                    } else {
                      newTokenForm.scopes = newTokenForm.scopes.filter(
                        (s) => s !== scope.value,
                      );
                    }
                  }
                "
              />
              {{ scope.label }}
            </label>
          </div>
        </div>
      </div>
      <template #footer>
        <CypButton @click="showCreateTokenDialog = false"> 取消 </CypButton>
        <CypButton
          type="primary"
          data-testid="create-button"
          @click="handleCreateToken"
        >
          创建
        </CypButton>
      </template>
    </CypDialog>

    <!-- 删除确认对话框（确认提示框，遵循3.3节） -->
    <CypDialog
      v-model="showDeleteConfirmDialog"
      title="删除访问令牌"
      width="400px"
      @close="showDeleteConfirmDialog = false"
    >
      <div class="delete-confirm">
        <p>
          确定要删除令牌 "<strong>{{ selectedToken?.name }}</strong
          >" 吗？
        </p>
        <p class="warning">
          此操作无法撤销，使用此令牌的应用将立即失去访问权限。
        </p>
      </div>
      <template #footer>
        <CypButton @click="showDeleteConfirmDialog = false"> 取消 </CypButton>
        <CypButton type="danger" @click="handleDeleteToken">
          确认删除
        </CypButton>
      </template>
    </CypDialog>

    <!-- 新建令牌结果对话框（系统框规范） -->
    <CypDialog
      v-model="showTokenDialog"
      title="令牌已创建"
      width="520px"
      @close="showTokenDialog = false"
    >
      <div class="token-result">
        <p class="token-tip">
          只会在此处显示一次完整令牌值，请立即复制并妥善保管。
        </p>
        <div class="token-block">
          <div class="token-label">令牌值</div>
          <div class="token-value">
            {{ createdToken }}
          </div>
        </div>
        <div class="token-meta">
          <span>类型：{{ createdTokenType }}</span>
        </div>
      </div>
      <template #footer>
        <CypButton @click="copyToken(createdToken)"> 复制令牌值 </CypButton>
        <CypButton type="primary" @click="showTokenDialog = false">
          关闭
        </CypButton>
      </template>
    </CypDialog>

    <!-- 编辑头像系统框（替代浏览器原生 prompt，遵循3.4节“弹窗系统框”规范） -->
    <CypDialog
      v-model="showAvatarDialog"
      title="更换头像"
      width="420px"
      @close="showAvatarDialog = false"
    >
      <div class="dialog-form">
        <div class="form-group">
          <label>上传头像图片</label>
          <input type="file" accept="image/*" @change="onAvatarFileChange" />
        </div>
      </div>
      <template #footer>
        <CypButton @click="showAvatarDialog = false"> 取消 </CypButton>
        <CypButton type="primary" @click="saveAvatarFromDialog">
          保存
        </CypButton>
      </template>
    </CypDialog>

    <!-- 通用提示框（信息/错误提示，遵循3.3节文案与布局规范） -->
    <CypDialog
      v-model="showMessageDialog"
      :title="messageDialogTitle"
      width="360px"
      @close="showMessageDialog = false"
    >
      <p>{{ messageDialogContent }}</p>
      <template #footer>
        <CypButton type="primary" @click="showMessageDialog = false">
          知道了
        </CypButton>
      </template>
    </CypDialog>
  </div>
</template>

<style lang="scss" scoped>
.settings-page {
  max-width: 1000px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 24px;

  h2 {
    font-size: 28px;
    font-weight: 700;
    line-height: 1.3;
    color: var(--text-primary, #1e293b);
    margin: 0 0 4px;
  }

  p {
    font-size: 14px;
    color: #64748b;
    margin: 0;
  }
}

.settings-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 24px;
}

.settings-nav {
  background: white;
  border-radius: 12px;
  padding: 12px;
  height: fit-content;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px 16px;
  border: none;
  background: transparent;
  text-align: left;
  font-size: 14px;
  color: #64748b;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: #f8fafc;
    color: #1e293b;
  }

  &.active {
    background: #6366f1;
    color: white;
  }
}

.settings-content {
  background: white;
  border-radius: 12px;
  padding: 32px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.settings-section {
  h2 {
    font-size: 20px;
    font-weight: 600;
    color: #1e293b;
    margin: 0 0 24px;
  }
}

.section-desc {
  font-size: 14px;
  color: #64748b;
  margin: -16px 0 20px;
}

.form-group {
  margin-bottom: 20px;

  label {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: #374151;
    margin-bottom: 8px;
  }
}

.textarea {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background-color: #ffffff;
  font-size: 14px;
  color: var(--text-primary, #1e293b);
  font-family: inherit;
  resize: vertical;

  &:focus {
    outline: none;
    border-color: #6366f1;
  }
}

.profile-avatar {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.avatar-preview {
  width: 80px;
  height: 80px;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 32px;
  font-weight: 600;
  overflow: hidden;
  position: relative;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 50%;
    position: absolute;
    top: 0;
    left: 0;
  }

  span {
    position: relative;
    z-index: 1;
  }
}

.toggle-group {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toggle-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
}

.toggle-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.toggle-label {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
}

.toggle-desc {
  font-size: 13px;
  color: #64748b;
}

.danger-zone {
  margin-top: 48px;
  padding: 20px;
  background: #fef2f2;
  border-radius: 8px;
  border: 1px solid #fecaca;

  h3 {
    font-size: 16px;
    font-weight: 600;
    color: #dc2626;
    margin: 0 0 8px;
  }

  p {
    font-size: 14px;
    color: #991b1b;
    margin: 0 0 16px;
  }
}

.token-list {
  margin-top: 24px;
}

.token-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  margin-bottom: 12px;
}

.token-info {
  flex: 1;
}

.token-name {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  display: block;
}

.token-value {
  font-size: 12px;
  color: #64748b;
  font-family: monospace;
  margin-top: 4px;
  display: block;
}

.token-meta {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
}

.token-scopes {
  margin-top: 2px;
}

.token-actions {
  display: flex;
  gap: 8px;
}

.theme-selector {
  margin-bottom: 32px;

  h3 {
    font-size: 14px;
    font-weight: 600;
    color: #1e293b;
    margin: 0 0 16px;
  }
}

.theme-options {
  display: flex;
  gap: 16px;
}

.theme-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  cursor: pointer;

  input {
    display: none;
  }

  .theme-preview {
    width: 80px;
    height: 60px;
    border-radius: 8px;
    border: 2px solid transparent;
    transition: all 0.2s ease;

    &.light {
      background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
    }

    &.dark {
      background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
    }

    &.auto {
      background: linear-gradient(135deg, #f8fafc 50%, #1e293b 50%);
    }
  }

  .theme-label {
    font-size: 13px;
    color: #64748b;
  }

  &.selected .theme-preview {
    border-color: #6366f1;
  }

  &:hover .theme-preview {
    transform: scale(1.02);
  }
}

.scope-options {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.scope-option {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #374151;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: #f1f5f9;
  }

  &.selected {
    background: #eef2ff;
    border-color: #6366f1;
    color: #6366f1;
  }
}

.dialog-form {
  .form-group {
    margin-bottom: 20px;
    &:last-child {
      margin-bottom: 0;
    }
    label {
      display: block;
      font-size: 14px;
      font-weight: 500;
      color: #374151;
      margin-bottom: 8px;
    }
  }
}

.delete-confirm {
  p {
    margin: 0 0 8px;
    font-size: 14px;
    color: #374151;
  }
  .warning {
    color: #dc2626;
    font-size: 13px;
  }
}

.token-result {
  .token-tip {
    font-size: 13px;
    color: #64748b;
    margin: 0 0 16px;
  }

  .token-block {
    padding: 12px 14px;
    background: #0f172a;
    border-radius: 8px;
    border: 1px solid #1e293b;
    margin-bottom: 12px;

    .token-label {
      font-size: 12px;
      color: #94a3b8;
      margin-bottom: 4px;
    }

    .token-value {
      font-family:
        ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas,
        "Liberation Mono", "Courier New", monospace;
      font-size: 13px;
      color: #e5e7eb;
      word-break: break-all;
    }
  }

  .token-meta {
    font-size: 12px;
    color: #94a3b8;
  }
}

@media (max-width: 768px) {
  .settings-layout {
    grid-template-columns: 1fr;
  }

  .settings-nav {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding: 8px;
  }

  .nav-item {
    flex-shrink: 0;
    padding: 10px 12px;
  }

  .token-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .token-actions {
    width: 100%;
    justify-content: flex-end;
  }
}

.config-group {
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid #e2e8f0;

  &:last-child {
    border-bottom: none;
  }

  h3 {
    font-size: 16px;
    font-weight: 600;
    color: #1e293b;
    margin: 0 0 16px;
  }
}

.form-hint {
  display: block;
  font-size: 12px;
  color: #64748b;
  margin-top: 4px;
}

.cors-origins-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 8px;
}

.cors-origin-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #374151;
}

.config-warning {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  background: #fef3c7;
  border: 1px solid #fde68a;
  border-radius: 8px;
  margin-bottom: 24px;
  color: #92400e;

  svg {
    flex-shrink: 0;
    margin-top: 2px;
  }

  div {
    flex: 1;
    font-size: 13px;
    line-height: 1.5;

    strong {
      font-weight: 600;
    }
  }
}
</style>
