import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { Project } from "@/types";
import { projectApi } from "@/services/project";

export type NotificationSource =
  | "scan"
  | "project"
  | "registry"
  | "webhook"
  | "system";

export type NotificationStatus = "success" | "failed" | "blocked" | "info";

export interface NotificationItem {
  id: string;
  source: NotificationSource;
  title: string;
  message: string;
  status: NotificationStatus;
  createdAt: string;
  read: boolean;
}

const MAX_NOTIFICATION_COUNT = 50;

export const useNotificationStore = defineStore("notification", () => {
  const items = ref<NotificationItem[]>([]);

  const unreadCount = computed(() => items.value.filter((n) => !n.read).length);

  function addNotification(input: {
    source: NotificationSource;
    title: string;
    message: string;
    status?: NotificationStatus;
    createdAt?: string;
  }) {
    const now = new Date();
    const item: NotificationItem = {
      id: `${now.getTime()}-${Math.random().toString(36).slice(2, 8)}`,
      source: input.source,
      title: input.title,
      message: input.message,
      status: input.status ?? "info",
      createdAt: input.createdAt ?? now.toISOString(),
      read: false,
    };
    items.value.unshift(item);
    if (items.value.length > MAX_NOTIFICATION_COUNT) {
      items.value.splice(MAX_NOTIFICATION_COUNT);
    }
  }

  function setNotifications(list: NotificationItem[]) {
    items.value = list.slice(0, MAX_NOTIFICATION_COUNT);
  }

  function markAllRead() {
    items.value = items.value.map((n) => ({ ...n, read: true }));
  }

  // 从后端加载一次性“系统快照”类通知：最近创建的项目
  async function loadFromServer() {
    try {
      const projectPage = await projectApi.getProjects({
        page: 1,
        pageSize: 5,
      });
      const projectList = ((projectPage as any)?.list ??
        (projectPage as any)?.projects ??
        []) as Project[];

      const projectNotifications: NotificationItem[] = projectList.map(
        (project) => ({
          id: `project-${project.id}`,
          source: "project",
          title: "新项目创建",
          message: `项目「${project.name}」已创建${project.isPublic ? "（公开项目）" : ""}`,
          status: "success",
          createdAt:
            (project as any).createdAt ?? (project as any).created_at ?? "",
          read: false,
        }),
      );

      const combined = [...projectNotifications];

      combined.sort((a, b) => {
        const t1 = Date.parse(a.createdAt || "");
        const t2 = Date.parse(b.createdAt || "");
        if (isNaN(t1) || isNaN(t2)) return 0;
        return t2 - t1;
      });

      setNotifications(combined);
    } catch (e) {
      // 静默失败，避免影响主流程
      // eslint-disable-next-line no-console
      console.error("加载系统通知失败", e);
    }
  }

  return {
    items,
    unreadCount,
    addNotification,
    setNotifications,
    markAllRead,
    loadFromServer,
  };
});
