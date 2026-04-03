<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import ToastContainer from '@/components/ToastContainer.vue'

const route = useRoute()
const { hasRole, logout } = useAuth()

const sidebarOpen = ref(false)

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

function closeSidebar() {
  sidebarOpen.value = false
}

function isActive(path: string): boolean {
  return route.path === path
}

function handleLogout() {
  logout()
}
</script>

<template>
  <div class="layout">
    <button
      class="hamburger"
      aria-label="Abrir menu"
      @click="toggleSidebar"
    >
      ☰
    </button>

    <div
      v-if="sidebarOpen"
      class="sidebar-overlay"
      @click="closeSidebar"
    />

    <aside class="sidebar" :class="{ 'sidebar--open': sidebarOpen }">
      <div class="sidebar__brand">i9pay</div>
      <nav class="sidebar__nav">
        <router-link
          to="/dashboard"
          class="sidebar__link"
          :class="{ 'sidebar__link--active': isActive('/dashboard') }"
          @click="closeSidebar"
        >
          Dashboard
        </router-link>
        <router-link
          to="/customers"
          class="sidebar__link"
          :class="{ 'sidebar__link--active': isActive('/customers') }"
          @click="closeSidebar"
        >
          Clientes
        </router-link>
        <router-link
          v-if="hasRole('admin')"
          to="/users"
          class="sidebar__link"
          :class="{ 'sidebar__link--active': isActive('/users') }"
          @click="closeSidebar"
        >
          Usuários
        </router-link>
        <button class="sidebar__link sidebar__logout" @click="handleLogout">
          Logout
        </button>
      </nav>
    </aside>

    <div class="main">
      <header class="header">
        <span class="header__brand">i9pay</span>
      </header>
      <main class="content">
        <slot />
      </main>
    </div>

    <ToastContainer />
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  min-height: 100vh;
}

.hamburger {
  display: none;
  position: fixed;
  top: 0.75rem;
  left: 0.75rem;
  z-index: 1100;
  background: #1e293b;
  color: #fff;
  border: none;
  font-size: 1.5rem;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  cursor: pointer;
}

.sidebar-overlay {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 999;
}

.sidebar {
  width: 220px;
  background: #1e293b;
  color: #e2e8f0;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.sidebar__brand {
  padding: 1.25rem 1rem;
  font-size: 1.25rem;
  font-weight: 700;
  border-bottom: 1px solid #334155;
}

.sidebar__nav {
  display: flex;
  flex-direction: column;
  padding: 0.5rem 0;
  flex: 1;
}

.sidebar__link {
  display: block;
  padding: 0.625rem 1rem;
  color: #cbd5e1;
  text-decoration: none;
  font-size: 0.875rem;
  transition: background 0.15s, color 0.15s;
}

.sidebar__link:hover {
  background: #334155;
  color: #fff;
}

.sidebar__link--active {
  background: #3b82f6;
  color: #fff;
}

.sidebar__logout {
  margin-top: auto;
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  font-family: inherit;
  border-top: 1px solid #334155;
}

.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.header {
  height: 56px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  padding: 0 1.5rem;
}

.header__brand {
  font-weight: 600;
  font-size: 1rem;
  color: #1e293b;
}

.content {
  flex: 1;
  padding: 1.5rem;
  background: #f8fafc;
}

@media (max-width: 767px) {
  .hamburger {
    display: block;
  }

  .sidebar-overlay {
    display: block;
  }

  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    z-index: 1000;
    transform: translateX(-100%);
    transition: transform 0.25s ease;
  }

  .sidebar--open {
    transform: translateX(0);
  }

  .header {
    padding-left: 3.5rem;
  }
}
</style>
