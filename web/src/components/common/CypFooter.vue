<script setup lang="ts">
import { ref, onMounted } from "vue";
import apiClient from "@/services/api";

const version = ref<string>("加载中...");

onMounted(async () => {
  try {
    const res: any = await apiClient.get("/health");
    if (res && typeof res.version === "string" && res.version) {
      version.value = res.version;
    } else {
      version.value = "未知版本";
    }
  } catch (error) {
    console.error("获取系统版本失败:", error);
    version.value = "未知版本";
  }
});
</script>

<template>
  <footer class="cyp-footer">
    <p class="cyp-footer__line">
      版本号：{{ version }} | 版权所有：© 2026 CYP 保留所有权利 |
      联系方式：nasDSSCYP@outlook.com
    </p>
    <p class="cyp-footer__line">
      告别公共仓库风险，私有镜像仓库：定制化存储，全链路安全可控！
    </p>
  </footer>
</template>

<style scoped lang="scss">
.cyp-footer {
  padding: 8px 24px 16px;
  text-align: center;
  font-size: 12px;
  line-height: 1.3;
  color: #9e9e9e;
  border-top: 1px solid var(--border-color, #e2e8f0);

  &__line + &__line {
    margin-top: 4px;
  }
}

// 深色模式适配
:global(.dark) .cyp-footer {
  border-color: #334155;
  color: #e0e0e0;
}
</style>
