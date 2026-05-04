<script setup lang="ts">
import AppLayout from '@/core/layouts/AppLayout.vue';
import { useRoute } from 'vue-router';

const route = useRoute()
</script>

<template>
  <AppLayout v-if="route.meta.requiresAuth">
    <router-view v-slot="{ Component, route: currentRoute }">
      <KeepAlive>
        <component
          :is="Component"
          v-if="currentRoute.meta.keepAlive"
          :key="currentRoute.path"
        />
      </KeepAlive>
      <component
        :is="Component"
        v-if="!currentRoute.meta.keepAlive"
        :key="currentRoute.fullPath"
      />
    </router-view>
  </AppLayout>
  <router-view v-else />
</template>
