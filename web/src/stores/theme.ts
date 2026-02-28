import { defineStore } from "pinia";
import { ref } from "vue";

export type Theme = "light" | "dark" | "auto";

export const useThemeStore = defineStore("theme", () => {
  const theme = ref<Theme>("light");
  const isDark = ref(false);

  // 从本地存储恢复主题
  function initTheme() {
    const savedTheme = localStorage.getItem("theme") as Theme | null;
    if (savedTheme) {
      theme.value = savedTheme;
    } else {
      // 检查系统偏好
      const prefersDark = window.matchMedia(
        "(prefers-color-scheme: dark)",
      ).matches;
      theme.value = prefersDark ? "dark" : "light";
    }
    applyTheme();
  }

  // 应用主题
  function applyTheme() {
    const root = document.documentElement;
    const systemDark = window.matchMedia(
      "(prefers-color-scheme: dark)",
    ).matches;

    // 计算实际主题
    let actualTheme: "light" | "dark";
    if (theme.value === "auto") {
      actualTheme = systemDark ? "dark" : "light";
    } else {
      actualTheme = theme.value;
    }

    isDark.value = actualTheme === "dark";

    // 应用到DOM
    if (actualTheme === "dark") {
      root.classList.add("dark");
    } else {
      root.classList.remove("dark");
    }

    // 保存到本地存储
    localStorage.setItem("theme", theme.value);
  }

  // 设置主题
  function setTheme(newTheme: Theme) {
    theme.value = newTheme;
    applyTheme();
  }

  // 切换主题
  function toggleTheme() {
    if (theme.value === "light") {
      setTheme("dark");
    } else if (theme.value === "dark") {
      setTheme("auto");
    } else {
      setTheme("light");
    }
  }

  // 监听系统主题变化
  function setupSystemThemeListener() {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    mediaQuery.addEventListener("change", () => {
      if (theme.value === "auto") {
        applyTheme();
      }
    });
  }

  return {
    theme,
    isDark,
    initTheme,
    setTheme,
    toggleTheme,
    setupSystemThemeListener,
  };
});
