<script setup>
import { computed, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const sidebarOpen = ref(false)

/**
 * One shell serves all three consoles; the navigation is derived from the
 * caller's role, so a customer never sees a builder or admin link at all.
 */
const sections = computed(() => {
  if (route.path.startsWith('/admin')) {
    return [
      {
        label: 'Overview',
        items: [
          { name: 'admin-dashboard', label: 'Dashboard', icon: '▤' },
          { name: 'admin-analytics', label: 'Analytics', icon: '◷' },
        ],
      },
      {
        label: 'Listings',
        items: [
          { name: 'admin-approvals', label: 'Approval queue', icon: '✓' },
          { name: 'admin-properties', label: 'All properties', icon: '⌂' },
          { name: 'admin-reports', label: 'Reported listings', icon: '⚠' },
        ],
      },
      {
        label: 'People',
        items: [
          { name: 'admin-clients', label: 'Builders', icon: '⚒' },
          { name: 'admin-users', label: 'Users', icon: '☺' },
          { name: 'admin-inquiries', label: 'Leads', icon: '✉' },
        ],
      },
      {
        label: 'Platform',
        items: [
          { name: 'admin-locations', label: 'Cities & localities', icon: '⚑' },
          { name: 'admin-amenities', label: 'Amenities', icon: '❋' },
          { name: 'admin-content', label: 'Homepage content', icon: '❐' },
          { name: 'admin-audit', label: 'Audit log', icon: '⎗' },
        ],
      },
    ]
  }

  if (route.path.startsWith('/client')) {
    return [
      {
        label: 'Overview',
        items: [{ name: 'client-dashboard', label: 'Dashboard', icon: '▤' }],
      },
      {
        label: 'Projects',
        items: [
          { name: 'client-properties', label: 'My properties', icon: '⌂' },
          { name: 'client-property-new', label: 'Add a property', icon: '＋' },
        ],
      },
      {
        label: 'Business',
        items: [
          { name: 'client-leads', label: 'Leads', icon: '✉' },
          { name: 'client-visits', label: 'Visit requests', icon: '📅' },
          { name: 'client-profile', label: 'Company profile', icon: '⚒' },
          { name: 'client-settings', label: 'Settings', icon: '⚙' },
        ],
      },
    ]
  }

  return [
    {
      label: 'My account',
      items: [
        { name: 'customer-dashboard', label: 'Dashboard', icon: '▤' },
        { name: 'customer-saved', label: 'Saved properties', icon: '♥' },
        { name: 'customer-inquiries', label: 'My inquiries', icon: '✉' },
        { name: 'customer-visits', label: 'My visits', icon: '⌚' },
        { name: 'customer-saved-searches', label: 'Saved searches', icon: '🔔' },
        { name: 'customer-recent', label: 'Recently viewed', icon: '◷' },
      ],
    },
    // The dashboard follows what the customer said they are here for, so both
    // catalogues stay one click away. Someone who signed up to rent and later
    // decides to buy should not have to work out how to get there.
    {
      label: 'Browse',
      items: [
        { name: 'rent', label: 'Homes to rent', icon: '🔑' },
        { name: 'properties', label: 'Property to buy', icon: '⌂' },
      ],
    },
    {
      label: 'Settings',
      items: [
        { name: 'customer-profile', label: 'Profile', icon: '☺' },
        { name: 'customer-preferences', label: 'Preferences', icon: '⚙' },
      ],
    },
  ]
})

const consoleName = computed(() => {
  if (route.path.startsWith('/admin')) return 'Admin console'
  if (route.path.startsWith('/client')) return 'Builder portal'
  return 'My account'
})

watch(() => route.fullPath, () => (sidebarOpen.value = false))

async function logout() {
  await auth.logout()
  router.push({ name: 'home' })
}
</script>

<template>
  <div class="dash">
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <RouterLink :to="{ name: 'home' }" class="brand">
        <span class="brand-mark" aria-hidden="true">R</span>
        <span class="brand-text">
          <strong>RERA</strong>
          <em>{{ consoleName }}</em>
        </span>
      </RouterLink>

      <nav class="side-nav">
        <div v-for="section in sections" :key="section.label" class="nav-section">
          <span class="nav-label">{{ section.label }}</span>
          <RouterLink
            v-for="item in section.items"
            :key="item.name"
            :to="{ name: item.name }"
            class="nav-item"
          >
            <span class="nav-icon" aria-hidden="true">{{ item.icon }}</span>
            {{ item.label }}
          </RouterLink>
        </div>
      </nav>

      <div class="side-foot">
        <RouterLink :to="{ name: 'home' }" class="nav-item">
          <span class="nav-icon" aria-hidden="true">←</span> Back to site
        </RouterLink>
        <button class="nav-item" @click="logout">
          <span class="nav-icon" aria-hidden="true">⏻</span> Log out
        </button>
      </div>
    </aside>

    <div class="content">
      <header class="topbar">
        <button class="burger only-mobile" aria-label="Menu" @click="sidebarOpen = !sidebarOpen">
          ☰
        </button>
        <h2 class="page-title">{{ route.meta.title || consoleName }}</h2>
        <div class="user">
          <div class="user-meta hide-mobile">
            <strong class="small">{{ auth.displayName }}</strong>
            <span class="tiny muted">{{ auth.roles.join(', ') }}</span>
          </div>
          <span class="avatar">{{ auth.initials }}</span>
        </div>
      </header>

      <div
        v-if="auth.needsVerification"
        class="verify-banner"
        role="status"
      >
        Your contact details are not verified yet.
        <RouterLink :to="{ name: 'verify-otp' }">Verify now</RouterLink>
      </div>

      <main class="page">
        <RouterView />
      </main>
    </div>

    <div v-if="sidebarOpen" class="scrim only-mobile" @click="sidebarOpen = false"></div>
  </div>
</template>

<style scoped>
.dash { display: grid; grid-template-columns: 250px 1fr; min-height: 100vh; }

.sidebar {
  background: var(--brand-900);
  color: var(--ink-300);
  padding: 18px 14px;
  display: flex;
  flex-direction: column;
  gap: 22px;
  position: sticky;
  top: 0;
  height: 100vh;
  overflow-y: auto;
}

.brand { display: flex; align-items: center; gap: 10px; padding: 0 6px; }
.brand-mark {
  width: 32px; height: 32px;
  display: grid; place-items: center;
  border-radius: 9px;
  background: var(--brand-500);
  color: var(--white);
  font-weight: 800;
  font-family: var(--font-display);
}
.brand-text { display: flex; flex-direction: column; line-height: 1.1; }
.brand-text strong { color: var(--white); font-size: 0.95rem; }
.brand-text em { font-style: normal; font-size: 0.68rem; color: var(--brand-100); letter-spacing: 0.06em; text-transform: uppercase; }

.side-nav { display: flex; flex-direction: column; gap: 20px; flex: 1; }
.nav-section { display: flex; flex-direction: column; gap: 2px; }
.nav-label {
  padding: 0 10px 6px;
  font-size: 0.66rem;
  font-weight: 800;
  letter-spacing: 0.09em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.38);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border-radius: var(--radius);
  color: rgba(255, 255, 255, 0.76);
  font-size: 0.87rem;
  font-weight: 500;
  background: none;
  border: 0;
  cursor: pointer;
  text-align: left;
  width: 100%;
  transition: all var(--transition);
}
.nav-item:hover { background: rgba(255, 255, 255, 0.08); color: var(--white); }
.nav-item.router-link-exact-active {
  background: var(--brand-600);
  color: var(--white);
  font-weight: 600;
}
.nav-icon { width: 18px; text-align: center; opacity: 0.85; }

.side-foot { display: flex; flex-direction: column; gap: 2px; border-top: 1px solid rgba(255, 255, 255, 0.1); padding-top: 12px; }

.content { display: flex; flex-direction: column; min-width: 0; }

.topbar {
  position: sticky;
  top: 0;
  z-index: 20;
  height: 62px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 0 22px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--ink-200);
}
.page-title { margin: 0; font-size: 1.05rem; flex: 1; }

.user { display: flex; align-items: center; gap: 10px; }
.user-meta { display: flex; flex-direction: column; align-items: flex-end; line-height: 1.2; }
.avatar {
  width: 34px; height: 34px;
  display: grid; place-items: center;
  border-radius: 50%;
  background: var(--brand-50);
  color: var(--brand-800);
  font-size: 0.78rem;
  font-weight: 700;
}

.verify-banner {
  padding: 10px 22px;
  background: var(--warning-100);
  color: var(--warning-600);
  font-size: 0.85rem;
  border-bottom: 1px solid #f0dcae;
}
.verify-banner a { color: var(--warning-600); font-weight: 700; text-decoration: underline; }

.page { padding: 22px; flex: 1; }

.burger {
  width: 36px; height: 36px;
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  background: var(--white);
  cursor: pointer;
  font-size: 1rem;
}

.scrim { position: fixed; inset: 0; background: rgba(15, 23, 32, 0.45); z-index: 40; }

@media (max-width: 900px) {
  .dash { grid-template-columns: 1fr; }
  .sidebar {
    position: fixed;
    left: 0; top: 0; bottom: 0;
    width: 250px;
    z-index: 50;
    transform: translateX(-100%);
    transition: transform 200ms ease;
  }
  .sidebar.open { transform: translateX(0); }
  .page { padding: 16px; }
}
</style>
