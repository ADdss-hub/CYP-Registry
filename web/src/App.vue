<script setup lang="ts">
import { onMounted } from "vue";
import { useUserStore } from "@/stores/user";

const userStore = useUserStore();

// 初始化时检查登录状态
onMounted(() => {
  if (userStore.token && !userStore.user) {
    userStore.fetchCurrentUser();
  }
});
</script>

<template>
  <router-view />
</template>

<style lang="scss">
#app {
  min-height: 100vh;
}
</style>
