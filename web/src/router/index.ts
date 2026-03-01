import { createRouter, createWebHistory } from "vue-router";
import type { RouteRecordRaw } from "vue-router";
import { LEGAL_STATEMENT_STORAGE_KEY } from "@/constants/legal";

// 布局组件
import CypLayout from "@/components/layout/CypLayout.vue";

// 视图组件
import LoginView from "@/views/auth/LoginView.vue";
import DashboardView from "@/views/DashboardView.vue";
import ProjectListView from "@/views/project/ProjectListView.vue";
import ProjectDetailView from "@/views/project/ProjectDetailView.vue";
import WebhookListView from "@/views/webhook/WebhookListView.vue";
import SettingsView from "@/views/settings/SettingsView.vue";
import ApiDocsView from "@/views/ApiDocsView.vue";
import LogsView from "@/views/admin/LogsView.vue";
import LegalStatementView from "@/views/legal/LegalStatementView.vue";

// 路由配置
const routes: RouteRecordRaw[] = [
  // 公开路由
  {
    path: "/login",
    name: "Login",
    component: LoginView,
    meta: { title: "登录", public: true },
  },
  // 公开注册功能已关闭：保留历史路由占位，直接重定向到登录页，避免旧链接报错
  {
    path: "/register",
    name: "Register",
    redirect: { name: "Login" },
    meta: { public: true },
  },

  // 需要认证的主布局路由
  {
    path: "/",
    component: CypLayout,
    meta: { requiresAuth: true },
    children: [
      {
        path: "",
        name: "Dashboard",
        component: DashboardView,
        meta: { title: "仪表盘" },
      },
      {
        path: "dashboard",
        name: "DashboardAlt",
        component: DashboardView,
        meta: { title: "仪表盘" },
      },
      {
        path: "projects",
        name: "Projects",
        component: ProjectListView,
        meta: { title: "项目管理" },
      },
      {
        path: "projects/:id",
        name: "ProjectDetail",
        component: ProjectDetailView,
        meta: { title: "项目详情" },
      },
      {
        path: "webhooks",
        name: "Webhooks",
        component: WebhookListView,
        meta: { title: "Webhook管理" },
      },
      {
        path: "settings",
        name: "Settings",
        component: SettingsView,
        meta: { title: "系统设置" },
      },
      {
        path: "logs",
        name: "Logs",
        component: LogsView,
        meta: { title: "系统日志" },
      },
      {
        path: "docs",
        name: "ApiDocs",
        component: ApiDocsView,
        meta: { title: "API文档" },
      },
    ],
  },

  // 声明与数据处理页面：登录后单独展示，不嵌入主控制台布局
  {
    path: "/legal/statement",
    name: "LegalStatement",
    component: LegalStatementView,
    meta: { title: "声明与数据处理", requiresAuth: true },
  },

  // 404页面
  {
    path: "/:pathMatch(.*)*",
    name: "NotFound",
    component: () => import("@/views/NotFoundView.vue"),
    meta: { title: "页面未找到" },
  },
];

// 创建路由
const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    if (savedPosition) {
      return savedPosition;
    } else {
      return { top: 0 };
    }
  },
});

// 路由守卫 - 认证检查 & 声明与数据处理确认检查
router.beforeEach((to, _from, next) => {
  try {
    // 设置页面标题
    document.title = to.meta.title
      ? `${to.meta.title} - CYP-Registry`
      : "CYP-Registry";

    // 检查是否需要认证
    if (to.meta.requiresAuth) {
      const token = localStorage.getItem("token");
      if (!token) {
        next({ name: "Login", query: { redirect: to.fullPath } });
        return;
      }

      // 登录后全局检查：未确认当前版本声明与数据处理，且目标路由不是声明页时，强制跳转到声明页
      const hasAcknowledgedLegalStatement =
        localStorage.getItem(LEGAL_STATEMENT_STORAGE_KEY) === "1";
      if (!hasAcknowledgedLegalStatement && to.name !== "LegalStatement") {
        next({ name: "LegalStatement" });
        return;
      }
    }

    // 检查是否是公开页面但已登录
    if (to.meta.public && localStorage.getItem("token")) {
      // 如果已登录但未确认声明，先跳转到声明页
      const hasAcknowledgedLegalStatement =
        localStorage.getItem(LEGAL_STATEMENT_STORAGE_KEY) === "1";
      if (!hasAcknowledgedLegalStatement) {
        next({ name: "LegalStatement" });
        return;
      }
      // 已确认声明，跳转到首页
      next({ name: "Dashboard" });
      return;
    }

    next();
  } catch (error) {
    // 捕获路由守卫中的错误
    console.error("[Router Error] beforeEach guard error:", {
      error,
      to: to.path,
      from: _from.path,
      timestamp: new Date().toISOString(),
    });
    // 继续导航，避免阻塞
    next();
  }
});

// 路由错误处理
router.onError((error) => {
  console.error("[Router Error] Navigation error:", {
    error: error.message,
    stack: error.stack,
    timestamp: new Date().toISOString(),
    url: window.location.href,
  });
});

export default router;
