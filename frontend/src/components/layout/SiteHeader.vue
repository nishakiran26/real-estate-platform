<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePropertyStore } from '@/stores/property'

const auth = useAuthStore()
const properties = usePropertyStore()
const router = useRouter()
const route = useRoute()

const menuOpen = ref(false)
const accountOpen = ref(false)
const query = ref('')

const savedCount = computed(() => properties.favoriteIds.size)

watch(
  () => route.fullPath,
  () => {
    menuOpen.value = false
    accountOpen.value = false
  },
)

function search() {
  const q = query.value.trim()
  router.push({ name: 'properties', query: q ? { q } : {} })
}

async function logout() {
  await auth.logout()
  router.push({ name: 'home' })
}
</script>

<template>
  <header class="site-header">
    <div class="container bar">
      <RouterLink :to="{ name: 'home' }" class="brand">
        <span class="brand-mark" aria-hidden="true">R</span>
        <span class="brand-text">
          <strong>RERA</strong>
          <em>Properties</em>
        </span>
      </RouterLink>

      <form class="quick-search hide-mobile" role="search" @submit.prevent="search">
        <input
          v-model="query"
          class="input"
          type="search"
          placeholder="Search by project, locality or builder"
          aria-label="Search properties"
        />
        <button class="btn btn-primary btn-sm" type="submit">Search</button>
      </form>

      <nav class="nav hide-mobile" aria-label="Main">
        <RouterLink :to="{ name: 'properties' }">Buy</RouterLink>
        <RouterLink :to="{ name: 'rent' }">Rent</RouterLink>
        <RouterLink :to="{ name: 'properties', query: { status: 'UPCOMING' } }">
          New projects
        </RouterLink>
        <RouterLink :to="{ name: 'properties', query: { rera_verified: 'true' } }">
          RERA
        </RouterLink>
      </nav>

      <div class="actions">
        <RouterLink
          v-if="auth.isAuthenticated && auth.isCustomer"
          :to="{ name: 'customer-saved' }"
          class="icon-link hide-mobile"
          title="Saved properties"
        >
          <span aria-hidden="true">♥</span>
          <span v-if="savedCount" class="count">{{ savedCount }}</span>
          <span class="visually-hidden">Saved properties</span>
        </RouterLink>

        <template v-if="auth.isAuthenticated">
          <div class="account">
            <button class="avatar" :aria-expanded="accountOpen" @click="accountOpen = !accountOpen">
              {{ auth.initials }}
            </button>
            <div v-if="accountOpen" class="dropdown" @click="accountOpen = false">
              <div class="dropdown-head">
                <strong class="truncate">{{ auth.displayName }}</strong>
                <span class="tiny muted truncate">{{ auth.user?.email }}</span>
              </div>
              <RouterLink v-if="auth.isAdmin" :to="{ name: 'admin-dashboard' }">
                Admin console
              </RouterLink>
              <RouterLink v-if="auth.isClient" :to="{ name: 'client-dashboard' }">
                Builder dashboard
              </RouterLink>
              <RouterLink v-if="auth.isCustomer" :to="{ name: 'customer-dashboard' }">
                My dashboard
              </RouterLink>
              <RouterLink v-if="auth.isCustomer" :to="{ name: 'customer-saved' }">
                Saved properties
              </RouterLink>
              <RouterLink v-if="auth.isCustomer" :to="{ name: 'customer-inquiries' }">
                My inquiries
              </RouterLink>
              <button class="dropdown-item" @click="logout">Log out</button>
            </div>
          </div>
        </template>

        <template v-else>
          <RouterLink :to="{ name: 'login' }" class="btn btn-ghost btn-sm hide-mobile">
            Log in
          </RouterLink>
          <RouterLink :to="{ name: 'register' }" class="btn btn-primary btn-sm">
            Post property
          </RouterLink>
        </template>

        <button
          class="burger only-mobile"
          :aria-expanded="menuOpen"
          aria-label="Menu"
          @click="menuOpen = !menuOpen"
        >
          <span></span><span></span><span></span>
        </button>
      </div>
    </div>

    <Transition name="slide">
      <div v-if="menuOpen" class="mobile-menu">
        <form class="quick-search" role="search" @submit.prevent="search">
          <input
            v-model="query"
            class="input"
            type="search"
            placeholder="Search projects or localities"
            aria-label="Search properties"
          />
          <button class="btn btn-primary btn-sm" type="submit">Go</button>
        </form>
        <RouterLink :to="{ name: 'properties' }">Buy a property</RouterLink>
        <RouterLink :to="{ name: 'rent' }">Rent a property</RouterLink>
        <RouterLink :to="{ name: 'properties', query: { status: 'READY_TO_MOVE' } }">
          Ready to move
        </RouterLink>
        <RouterLink :to="{ name: 'properties', query: { status: 'UPCOMING' } }">
          New launches
        </RouterLink>
        <RouterLink :to="{ name: 'compare' }">Compare</RouterLink>
        <template v-if="!auth.isAuthenticated">
          <RouterLink :to="{ name: 'login' }">Log in</RouterLink>
          <RouterLink :to="{ name: 'register' }">Create an account</RouterLink>
        </template>
        <button v-else class="dropdown-item" @click="logout">Log out</button>
      </div>
    </Transition>
  </header>
</template>

<style scoped>
.site-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--ink-200);
}

.bar {
  height: var(--header-height);
  display: flex;
  align-items: center;
  gap: 18px;
}

.brand { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.brand-mark {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 9px;
  background: var(--brand-700);
  color: var(--white);
  font-weight: 800;
  font-family: var(--font-display);
}
.brand-text { display: flex; flex-direction: column; line-height: 1.05; }
.brand-text strong { color: var(--ink-900); font-size: 1rem; letter-spacing: 0.02em; }
.brand-text em { font-style: normal; font-size: 0.72rem; color: var(--ink-500); letter-spacing: 0.1em; text-transform: uppercase; }

.quick-search { display: flex; gap: 8px; flex: 1; max-width: 420px; }
.quick-search .input { height: 38px; }

.nav { display: flex; gap: 18px; }
.nav a { color: var(--ink-700); font-weight: 600; font-size: 0.9rem; }
.nav a:hover, .nav a.router-link-active { color: var(--brand-700); }

.actions { display: flex; align-items: center; gap: 10px; margin-left: auto; }

.icon-link { position: relative; padding: 6px; color: var(--ink-600); font-size: 1.1rem; }
.icon-link:hover { color: var(--danger-600); }
.icon-link .count {
  position: absolute;
  top: -2px;
  right: -4px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: var(--radius-full);
  background: var(--danger-600);
  color: var(--white);
  font-size: 0.64rem;
  font-weight: 700;
  display: grid;
  place-items: center;
}

.account { position: relative; }
.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 1px solid var(--ink-200);
  background: var(--brand-50);
  color: var(--brand-800);
  font-weight: 700;
  font-size: 0.82rem;
  cursor: pointer;
}
.avatar:hover { background: var(--brand-100); }

.dropdown {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  min-width: 220px;
  background: var(--white);
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  padding: 6px;
  display: flex;
  flex-direction: column;
}
.dropdown-head {
  display: flex;
  flex-direction: column;
  padding: 10px 12px;
  border-bottom: 1px solid var(--ink-100);
  margin-bottom: 4px;
}
.dropdown a,
.dropdown-item {
  padding: 9px 12px;
  border-radius: var(--radius-sm);
  color: var(--ink-700);
  font-size: 0.88rem;
  text-align: left;
  background: none;
  border: 0;
  cursor: pointer;
  font-weight: 500;
}
.dropdown a:hover,
.dropdown-item:hover { background: var(--ink-100); color: var(--brand-800); }

.burger {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  gap: 4px;
  border: 1px solid var(--ink-200);
  border-radius: var(--radius);
  background: var(--white);
  cursor: pointer;
}
.burger span { width: 16px; height: 2px; background: var(--ink-700); border-radius: 2px; }

.mobile-menu {
  border-top: 1px solid var(--ink-200);
  background: var(--white);
  padding: 14px 20px 18px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.mobile-menu a,
.mobile-menu .dropdown-item {
  padding: 11px 4px;
  border-bottom: 1px solid var(--ink-100);
  color: var(--ink-700);
  font-weight: 600;
}
.mobile-menu .quick-search { max-width: none; margin-bottom: 8px; }

.slide-enter-active, .slide-leave-active { transition: opacity 160ms ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; }

@media (max-width: 900px) {
  .nav { display: none; }
}
</style>
