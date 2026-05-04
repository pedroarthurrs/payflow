<script setup lang="ts">
import ToastContainer from '@/core/components/ToastContainer.vue'
import { Button } from '@/core/components/ui/button'
import { Separator } from '@/core/components/ui/separator'
import { useAuth } from '@/core/composables/useAuth'
import { cn } from '@/core/utils/formatting'
import { LayoutDashboard, LogOut, Menu, PanelLeftClose, PanelLeftOpen, UserCog, Users } from 'lucide-vue-next'
import { ref } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const { hasRole, logout } = useAuth()

const sidebarOpen = ref(false)
const sidebarCollapsed = ref(false)

function toggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

function closeSidebar() {
  sidebarOpen.value = false
}

function toggleSidebarCollapsed() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(`${path}/`)
}

function handleLogout() {
  logout()
}
</script>

<template>
  <div class="flex min-h-svh">
    <!-- Mobile hamburger -->
    <Button
      variant="ghost"
      size="icon"
      aria-label="Abrir menu"
      class="fixed top-3 left-3 z-[1100] md:hidden"
      @click="toggleSidebar"
    >
      <Menu class="size-5" />
    </Button>

    <!-- Overlay mobile -->
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 z-[999] bg-black/40 md:hidden"
      @click="closeSidebar"
    />

    <!-- Sidebar -->
    <aside
      :class="cn(
        'fixed inset-y-0 left-0 z-[1000] flex w-44 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground transition-all duration-200 md:static md:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full',
        sidebarCollapsed ? 'md:w-16' : 'md:w-44'
      )"
    >
      <div class="flex items-center justify-between border-b border-sidebar-border px-3 py-3">
        <span :class="cn('text-xl font-extrabold tracking-tight', sidebarCollapsed && 'md:hidden')">i9pay</span>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Expandir ou retrair sidebar"
          class="rounded-none text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
          @click="toggleSidebarCollapsed"
        >
          <PanelLeftOpen v-if="sidebarCollapsed" class="size-5" />
          <PanelLeftClose v-else class="size-5" />
        </Button>
      </div>

      <nav class="flex flex-1 flex-col gap-1 p-2">
        <router-link
          to="/dashboard"
          :class="cn(
            'group relative flex items-center gap-3 rounded-none px-3 py-2.5 text-base transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
            isActive('/dashboard') && 'bg-sidebar-accent text-sidebar-accent-foreground',
            sidebarCollapsed && 'md:justify-center md:px-2'
          )"
          @click="closeSidebar"
        >
          <LayoutDashboard class="size-5" />
          <span :class="cn(sidebarCollapsed && 'md:hidden')">Dashboard</span>
          <span
            v-if="sidebarCollapsed"
            class="pointer-events-none absolute left-full top-1/2 z-50 ml-4 hidden -translate-y-1/2 whitespace-nowrap rounded-none border border-border bg-popover px-2.5 py-1.5 text-sm text-popover-foreground opacity-0 shadow-md transition-opacity group-hover:opacity-100 md:block"
          >
            Dashboard
          </span>
        </router-link>

        <router-link
          to="/customers"
          :class="cn(
            'group relative flex items-center gap-3 rounded-none px-3 py-2.5 text-base transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
            isActive('/customers') && 'bg-sidebar-accent text-sidebar-accent-foreground',
            sidebarCollapsed && 'md:justify-center md:px-2'
          )"
          @click="closeSidebar"
        >
          <Users class="size-5" />
          <span :class="cn(sidebarCollapsed && 'md:hidden')">Clientes</span>
          <span
            v-if="sidebarCollapsed"
            class="pointer-events-none absolute left-full top-1/2 z-50 ml-4 hidden -translate-y-1/2 whitespace-nowrap rounded-none border border-border bg-popover px-2.5 py-1.5 text-sm text-popover-foreground opacity-0 shadow-md transition-opacity group-hover:opacity-100 md:block"
          >
            Clientes
          </span>
        </router-link>

        <router-link
          v-if="hasRole('admin')"
          to="/members"
          :class="cn(
            'group relative flex items-center gap-3 rounded-none px-3 py-2.5 text-base transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
            isActive('/members') && 'bg-sidebar-accent text-sidebar-accent-foreground',
            sidebarCollapsed && 'md:justify-center md:px-2'
          )"
          @click="closeSidebar"
        >
          <UserCog class="size-5" />
          <span :class="cn(sidebarCollapsed && 'md:hidden')">Members</span>
          <span
            v-if="sidebarCollapsed"
            class="pointer-events-none absolute left-full top-1/2 z-50 ml-4 hidden -translate-y-1/2 whitespace-nowrap rounded-none border border-border bg-popover px-2.5 py-1.5 text-sm text-popover-foreground opacity-0 shadow-md transition-opacity group-hover:opacity-100 md:block"
          >
            Members
          </span>
        </router-link>

        <Separator class="mt-auto bg-sidebar-border" />
        <button
          :class="cn(
            'group relative flex justify-center w-full items-center gap-3 rounded-none px-3 py-2.5 text-left text-base transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
            sidebarCollapsed && 'md:justify-center md:px-2'
          )"
          @click="handleLogout"
        >
          <LogOut class="size-5" />
          <span :class="cn(sidebarCollapsed && 'md:hidden')">Logout</span>
          <span
            v-if="sidebarCollapsed"
            class="pointer-events-none absolute left-full top-1/2 z-50 ml-4 hidden -translate-y-1/2 whitespace-nowrap rounded-none border border-border bg-popover px-2.5 py-1.5 text-sm text-popover-foreground opacity-0 shadow-md transition-opacity group-hover:opacity-100 md:block"
          >
            Logout
          </span>
        </button>
      </nav>
    </aside>

    <!-- Main content -->
    <div class="flex min-w-0 flex-1 flex-col">
      <main class="flex-1 bg-muted/20 p-6 pt-14 md:pt-6">
        <slot />
      </main>
    </div>

    <ToastContainer />
  </div>
</template>
