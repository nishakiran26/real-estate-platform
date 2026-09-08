<script setup>
import { RouterView } from 'vue-router'
</script>

<template>
  <div class="auth-shell">
    <aside class="promo">
      <RouterLink :to="{ name: 'home' }" class="brand">
        <span class="brand-mark" aria-hidden="true">R</span>
        <span>RERA Properties</span>
      </RouterLink>

      <div class="promo-body">
        <h2 class="display">Every project, checked before it is published.</h2>
        <ul class="points">
          <li><strong>RERA first.</strong> Registration details are shown on every listing.</li>
          <li><strong>Reviewed listings.</strong> A builder cannot publish without approval.</li>
          <li><strong>Real configurations.</strong> Carpet area and price, not just a brochure.</li>
        </ul>
      </div>

      <p class="tiny footnote">
        Buyers should independently verify project details with the relevant RERA authority.
      </p>
    </aside>

    <main class="form-side">
      <div class="form-wrap">
        <RouterView />
      </div>
    </main>
  </div>
</template>

<style scoped>
.auth-shell {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 460px) 1fr;
}

.promo {
  background: linear-gradient(160deg, var(--brand-900) 0%, var(--brand-700) 100%);
  color: var(--white);
  padding: 34px 38px;
  display: flex;
  flex-direction: column;
  gap: 30px;
}

.brand { display: flex; align-items: center; gap: 10px; color: var(--white); font-weight: 700; }
.brand-mark {
  width: 34px; height: 34px;
  display: grid; place-items: center;
  border-radius: 9px;
  background: rgba(255, 255, 255, 0.16);
  font-family: var(--font-display);
  font-weight: 800;
}

.promo-body { margin-top: auto; }
.promo h2 { color: var(--white); font-size: 1.9rem; line-height: 1.22; margin-bottom: 24px; }

.points { list-style: none; padding: 0; margin: 0; display: grid; gap: 14px; }
.points li {
  padding-left: 26px;
  position: relative;
  font-size: 0.92rem;
  color: rgba(255, 255, 255, 0.85);
}
.points li::before {
  content: '✓';
  position: absolute;
  left: 0;
  top: 0;
  width: 18px; height: 18px;
  display: grid; place-items: center;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.18);
  font-size: 0.66rem;
}
.points strong { color: var(--white); }

.footnote { margin: auto 0 0; color: rgba(255, 255, 255, 0.55); }

.form-side {
  display: grid;
  place-items: center;
  padding: 40px 20px;
  background: var(--ink-50);
}
.form-wrap { width: 100%; max-width: 440px; }

@media (max-width: 940px) {
  .auth-shell { grid-template-columns: 1fr; }
  .promo { padding: 20px; gap: 18px; }
  .promo-body h2 { font-size: 1.4rem; margin-bottom: 14px; }
  .points, .footnote { display: none; }
}
</style>
