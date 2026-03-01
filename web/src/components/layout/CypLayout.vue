<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useUserStore } from "@/stores/user";
import { useThemeStore } from "@/stores/theme";
import { useNotificationStore } from "@/stores/notification";
import CypFooter from "@/components/common/CypFooter.vue";
import logoCypRegistry from "@/assets/logo-cyp-registry.svg";

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();
const themeStore = useThemeStore();
const notificationStore = useNotificationStore();

const isCollapsed = ref(false);

// 菜单项：当前系统仅存在管理员用户，所有人都可以看到系统日志
const menuItems = [
  { path: "/dashboard", icon: "dashboard", label: "仪表盘" },
  { path: "/projects", icon: "project", label: "项目管理" },
  { path: "/webhooks", icon: "webhook", label: "Webhook" },
  { path: "/logs", icon: "logs", label: "系统日志" },
  { path: "/settings", icon: "settings", label: "系统设置" },
  { path: "/docs", icon: "docs", label: "API文档" },
];

const currentPath = computed(() => route.path);

const showNotificationPanel = ref(false);
const notifications = computed(() => notificationStore.items);
const unreadCount = computed(() => notificationStore.unreadCount);

// 初始化主题 & 通知
onMounted(() => {
  themeStore.initTheme();
  themeStore.setupSystemThemeListener();
  notificationStore.loadFromServer();
});

function toggleSidebar() {
  isCollapsed.value = !isCollapsed.value;
}

function toggleTheme() {
  themeStore.toggleTheme();
}

function handleLogout() {
  userStore.logout();
}

function navigateTo(path: string) {
  router.push(path);
}

function toggleNotificationCenter() {
  showNotificationPanel.value = !showNotificationPanel.value;
  if (showNotificationPanel.value) {
    notificationStore.markAllRead();
  }
}

// 获取主题图标
function getThemeIcon() {
  const theme = themeStore.theme;
  if (theme === "dark") {
    return "🌙";
  } else if (theme === "auto") {
    return "🖥️";
  }
  return "☀️";
}

// 获取主题提示文本
function getThemeTooltip() {
  const theme = themeStore.theme;
  if (theme === "dark") {
    return "切换到浅色模式";
  } else if (theme === "auto") {
    return "切换到深色模式";
  }
  return "切换到自动模式";
}

// 获取头像URL（添加时间戳防止缓存）
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
</script>

<template>
  <div class="cyp-layout" :class="{ 'sidebar-collapsed': isCollapsed }">
    <!-- 侧边栏 -->
    <aside class="cyp-layout__sidebar" data-testid="sidebar">
      <div class="cyp-layout__logo">
        <div class="logo-icon">
          <img
            :src="logoCypRegistry"
            alt="CYP-Registry Logo"
            width="32"
            height="32"
          />
        </div>
        <span v-if="!isCollapsed" class="logo-text">CYP-Registry</span>
      </div>

      <nav class="cyp-layout__nav">
        <ul class="nav-list">
          <li
            v-for="item in menuItems"
            :key="item.path"
            class="nav-item"
            :class="{ active: currentPath.startsWith(item.path) }"
            @click="navigateTo(item.path)"
          >
            <span class="nav-icon">
              <svg viewBox="0 0 24 24" width="20" height="20">
                <template v-if="item.icon === 'dashboard'">
                  <path
                    fill="currentColor"
                    d="M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z"
                  />
                </template>
                <template v-else-if="item.icon === 'project'">
                  <path
                    fill="currentColor"
                    d="M20 6h-8l-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm-1 12H5c-.55 0-1-.45-1-1V9h16v8c0 .55-.45 1-1 1z"
                  />
                </template>
                <template v-else-if="item.icon === 'webhook'">
                  <path
                    fill="currentColor"
                    d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"
                  />
                </template>
                <template v-else-if="item.icon === 'settings'">
                  <path
                    fill="currentColor"
                    d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"
                  />
                </template>
                <template v-else-if="item.icon === 'docs'">
                  <path
                    fill="currentColor"
                    d="M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z"
                  />
                </template>
                <template v-else-if="item.icon === 'logs'">
                  <path
                    fill="currentColor"
                    d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-5 14H7v-2h7v2zm3-4H7v-2h10v2zm0-4H7V7h10v2z"
                  />
                </template>
                <template v-else>
                  <rect fill="currentColor" width="24" height="24" rx="4" />
                </template>
              </svg>
            </span>
            <span v-if="!isCollapsed" class="nav-label">{{ item.label }}</span>
          </li>
        </ul>
      </nav>

      <div class="cyp-layout__sidebar-footer">
        <button
          class="collapse-btn"
          data-testid="sidebar-toggle"
          @click="toggleSidebar"
        >
          <svg
            viewBox="0 0 24 24"
            width="20"
            height="20"
            :style="{ transform: isCollapsed ? 'rotate(180deg)' : '' }"
          >
            <path
              fill="currentColor"
              d="M15.41 7.41L14 6l-6 6 6 6 1.41-1.41L10.83 12z"
            />
          </svg>
        </button>
      </div>
    </aside>

    <!-- 主内容区 -->
    <div class="cyp-layout__main">
      <!-- 顶部栏 -->
      <header class="cyp-layout__header">
        <div class="header-left">
          <h1 class="header-title">
            {{ route.meta.title || "仪表盘" }}
          </h1>
        </div>
        <div class="header-right">
          <button
            class="header-btn theme-btn"
            :title="getThemeTooltip()"
            @click="toggleTheme"
          >
            <span class="theme-icon">{{ getThemeIcon() }}</span>
          </button>

          <div class="notification-wrapper">
            <button
              class="header-btn notification-btn"
              title="系统通知"
              @click="toggleNotificationCenter"
            >
              <svg viewBox="0 0 24 24" width="20" height="20">
                <path
                  fill="currentColor"
                  d="M12 22c1.1 0 2-.9 2-2h-4a2 2 0 0 0 2 2zm6-6v-5c0-3.07-1.63-5.64-4.5-6.32V4a1.5 1.5 0 0 0-3 0v.68C8.63 5.36 7 7.92 7 11v5l-2 2v1h15v-1l-2-2z"
                />
              </svg>
              <span v-if="unreadCount > 0" class="notification-badge">
                {{ unreadCount > 9 ? "9+" : unreadCount }}
              </span>
            </button>

            <div v-if="showNotificationPanel" class="notification-panel">
              <div class="notification-panel__header">
                <span class="title">系统通知</span>
                <button
                  class="link-btn"
                  type="button"
                  @click="notificationStore.loadFromServer"
                >
                  刷新
                </button>
              </div>
              <div class="notification-panel__body">
                <div
                  v-if="notifications.length === 0"
                  class="notification-empty"
                >
                  暂无通知
                </div>
                <div
                  v-for="item in notifications"
                  :key="item.id"
                  class="notification-item"
                  :class="item.status"
                >
                  <div class="notification-item__main">
                    <div class="notification-item__title">
                      {{ item.title }}
                    </div>
                    <div class="notification-item__message">
                      {{ item.message }}
                    </div>
                  </div>
                  <div class="notification-item__meta">
                    <span class="time">{{ item.createdAt }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="user-dropdown">
            <div class="user-avatar">
              <img
                v-if="userStore.user?.avatar"
                :src="getAvatarUrl(userStore.user.avatar)"
                alt="avatar"
              />
              <span v-else>
                {{ userStore.user?.username?.charAt(0)?.toUpperCase() || "U" }}
              </span>
            </div>
            <span class="user-name">{{ userStore.user?.username }}</span>
            <button class="logout-btn" @click="handleLogout">退出</button>
          </div>
        </div>
      </header>

      <!-- 页面内容 -->
      <main class="cyp-layout__content" data-testid="main-content">
        <router-view />
      </main>

      <!-- 底部统一信息（依据《界面开发设计规范》4.1） -->
      <CypFooter />
    </div>
  </div>
</template>

<style lang="scss" scoped>
@use "@/assets/styles/variables.scss" as *;

.cyp-layout {
  display: flex;
  min-height: 100vh;
  background: var(--bg-primary, #f8fafc);

  &__sidebar {
    width: 240px;
    background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
    display: flex;
    flex-direction: column;
    transition: width 0.3s ease;
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 100;

    &.collapsed {
      width: 64px;

      .nav-label,
      .logo-text {
        display: none;
      }

      .nav-item {
        justify-content: center;
        padding: 12px;
      }
    }
  }

  &__logo {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 20px 16px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);

    .logo-icon {
      flex-shrink: 0;
    }

    .logo-text {
      font-size: 16px;
      font-weight: 600;
      color: white;
      white-space: nowrap;
    }
  }

  &__nav {
    flex: 1;
    padding: 16px 8px;
    overflow-y: auto;
  }

  .nav-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    border-radius: 8px;
    color: #94a3b8;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.05);
      color: white;
    }

    &.active {
      background: $primary-color;
      color: white;
    }
  }

  .nav-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
  }

  &__sidebar-footer {
    padding: 16px;
    border-top: 1px solid rgba(255, 255, 255, 0.1);
  }

  .collapse-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    padding: 8px;
    background: rgba(255, 255, 255, 0.05);
    border: none;
    border-radius: 6px;
    color: #94a3b8;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.1);
      color: white;
    }
  }

  &__main {
    flex: 1;
    margin-left: 240px;
    transition: margin-left 0.3s ease;
    display: flex;
    flex-direction: column;
  }

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 56px;
    padding: 0 24px;
    background: var(--bg-white, white);
    border-bottom: 1px solid var(--border-color, #e2e8f0);
    position: sticky;
    top: 0;
    z-index: 50;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .header-title {
    font-size: 22px;
    font-weight: 600;
    color: var(--text-primary, #1e293b);
    margin: 0;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .header-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    background: var(--bg-secondary, #f1f5f9);
    border: none;
    border-radius: 8px;
    color: var(--text-secondary, #64748b);
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      background: var(--bg-tertiary, #e2e8f0);
      color: var(--text-primary, #1e293b);
    }
  }

  .theme-btn {
    font-size: 18px;

    .theme-icon {
      line-height: 1;
    }
  }

  .user-dropdown {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .user-avatar {
    width: 36px;
    height: 36px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-weight: 600;
    font-size: 14px;
    overflow: hidden;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
      display: block;
    }
  }

  .user-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary, #1e293b);
  }

  .logout-btn {
    padding: 6px 12px;
    background: transparent;
    border: 1px solid var(--border-color, #e2e8f0);
    border-radius: 6px;
    color: var(--text-secondary, #64748b);
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s ease;

    &:hover {
      border-color: $danger-color;
      color: $danger-color;
    }
  }

  .notification-wrapper {
    position: relative;

    .notification-btn {
      position: relative;
    }

    .notification-badge {
      position: absolute;
      top: 4px;
      right: 4px;
      min-width: 16px;
      height: 16px;
      padding: 0 4px;
      border-radius: 999px;
      background: #ef4444;
      color: #fff;
      font-size: 10px;
      line-height: 16px;
      text-align: center;
      box-shadow: 0 0 0 1px #fff;
    }

    .notification-panel {
      position: absolute;
      top: 48px;
      right: 0;
      width: 360px;
      max-height: 420px;
      background: var(--bg-white, #ffffff);
      box-shadow: 0 12px 30px rgba(15, 23, 42, 0.25);
      border-radius: 12px;
      border: 1px solid var(--border-color, #e2e8f0);
      display: flex;
      flex-direction: column;
      overflow: hidden;
      z-index: 60;

      &__header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 10px 14px;
        border-bottom: 1px solid var(--border-color, #e2e8f0);

        .title {
          font-size: 14px;
          font-weight: 600;
          color: var(--text-primary, #0f172a);
        }

        .link-btn {
          border: none;
          background: none;
          padding: 0;
          font-size: 12px;
          color: $primary-color;
          cursor: pointer;
        }
      }

      &__body {
        padding: 8px 0;
        overflow-y: auto;
      }
    }

    .notification-empty {
      padding: 24px 16px;
      text-align: center;
      font-size: 13px;
      color: var(--text-secondary, #64748b);
    }

    .notification-item {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      padding: 10px 14px;
      gap: 8px;
      cursor: default;

      &:hover {
        background: #f8fafc;
      }

      &__main {
        flex: 1;
      }

      &__title {
        font-size: 13px;
        font-weight: 600;
        color: var(--text-primary, #0f172a);
        margin-bottom: 2px;
      }

      &__message {
        font-size: 12px;
        color: var(--text-secondary, #64748b);
      }

      &__meta {
        font-size: 11px;
        color: #94a3b8;
        white-space: nowrap;
        margin-left: 4px;
      }

      &.failed &__title {
        color: $danger-color;
      }

      &.blocked &__title {
        color: #ea580c;
      }
    }
  }

  &__content {
    flex: 1;
    padding: 24px;
    overflow-y: auto;
  }

  // 侧边栏收起时的样式
  &.sidebar-collapsed {
    &__main {
      margin-left: 64px;
    }
  }
}

// 暗色主题 - 使用全局变量
:global(.dark) {
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --bg-tertiary: #334155;
  --bg-white: #1e293b;
  --text-primary: #f1f5f9;
  --text-secondary: #94a3b8;
  --border-color: #334155;

  .cyp-layout {
    &__header {
      border-color: #334155;
    }

    .header-title {
      color: #f1f5f9;
    }

    .header-btn {
      background: #334155;
      color: #94a3b8;

      &:hover {
        background: #475569;
        color: #f1f5f9;
      }
    }

    .user-name {
      color: #f1f5f9;
    }

    .logout-btn {
      border-color: #334155;
      color: #94a3b8;

      &:hover {
        border-color: #ef4444;
      }
    }
  }
}

@media (max-width: 768px) {
  .cyp-layout {
    &__sidebar {
      width: 64px;
    }

    &__main {
      margin-left: 64px;
    }

    .nav-label,
    .logo-text {
      display: none;
    }

    .nav-item {
      justify-content: center;
      padding: 12px;
    }
  }
}
</style>
