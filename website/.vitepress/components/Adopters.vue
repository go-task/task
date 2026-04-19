<script setup lang="ts">
import { adopters } from '../adopters';

const pad = (n: number) => String(n).padStart(2, '0');

const githubPath = (url: string) =>
  url.replace(/^https?:\/\/github\.com\//, '').replace(/\/$/, '');
</script>

<template>
  <div class="adopters">
    <header class="intro">
      <p class="kicker">
        <span class="slashes">//</span>
        {{ pad(adopters.length) }} projects and counting
      </p>
      <h1 class="title">Built with Task.</h1>
      <p class="lede">
        A curated list of open source projects that rely on Task for their build
        and release workflows. From hardware toolchains to AI frameworks, Task
        powers the command line of teams worldwide.
      </p>
    </header>

    <section class="grid">
      <a
        v-for="(item, i) in adopters"
        :key="item.name"
        :href="item.url"
        target="_blank"
        rel="noopener"
        class="card"
      >
        <span class="corner tl"></span>
        <span class="corner tr"></span>
        <span class="corner bl"></span>
        <span class="corner br"></span>

        <div class="card-head">
          <img :src="item.img" :alt="`${item.name} logo`" class="card-logo" />
          <span class="card-index">N&deg; {{ pad(i + 1) }}</span>
        </div>

        <h2 class="card-name">{{ item.name }}</h2>

        <div class="card-foot">
          <span class="card-path">{{ githubPath(item.url) }}</span>
          <span class="card-cta">
            <span class="cta-label">View</span>
            <span class="cta-arrow">&rarr;</span>
          </span>
        </div>
      </a>
    </section>

    <aside class="cta">
      <div class="cta-body">
        <p class="cta-kicker">
          <span class="slashes">//</span>
          Using Task in your project?
        </p>
        <h3 class="cta-title">Add your project.</h3>
        <p class="cta-text">
          Open a pull request updating
          <code>.vitepress/adopters.ts</code> — the only requirement is that
          Task is the task runner for your open source project.
        </p>
      </div>
      <a
        class="cta-button"
        href="https://github.com/go-task/task/blob/main/website/.vitepress/adopters.ts"
        target="_blank"
        rel="noopener"
      >
        <span>Edit adopters.ts</span>
        <span class="cta-arrow">&rarr;</span>
      </a>
    </aside>
  </div>
</template>

<style scoped>
.adopters {
  max-width: 1152px;
  margin: 0 auto;
  padding: 0 24px 6rem;
}

/* ---------- Intro ---------- */
.intro {
  padding: 3rem 0 4rem;
  max-width: 42rem;
}

.kicker {
  font-family: var(--vp-font-family-mono);
  font-size: 0.8rem;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-2);
  text-transform: uppercase;
  margin: 0 0 1rem;
}

.slashes {
  color: var(--vp-c-brand-1);
  margin-right: 0.4em;
}

.title {
  font-size: clamp(2.25rem, 5vw, 3.5rem);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1;
  margin: 0 0 1.5rem;
  color: var(--vp-c-text-1);
}

.lede {
  font-size: 1.1rem;
  line-height: 1.6;
  color: var(--vp-c-text-2);
  margin: 0;
}

/* ---------- Grid ---------- */
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

/* ---------- Card ---------- */
.card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding: 1.5rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 14px;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  text-decoration: none !important;
  transition:
    border-color 0.3s ease,
    background 0.3s ease,
    transform 0.3s ease,
    box-shadow 0.3s ease;
  isolation: isolate;
  overflow: hidden;
}

.card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(
    600px circle at var(--x, 50%) var(--y, 50%),
    color-mix(in srgb, var(--vp-c-brand-1) 12%, transparent),
    transparent 40%
  );
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
  z-index: -1;
}

.card:hover {
  border-color: color-mix(
    in srgb,
    var(--vp-c-brand-1) 50%,
    var(--vp-c-divider)
  );
  transform: translateY(-2px);
  box-shadow: 0 14px 40px -24px
    color-mix(in srgb, var(--vp-c-brand-1) 40%, transparent);
}

.card:hover::before {
  opacity: 1;
}

/* Crosshair corner marks */
.corner {
  position: absolute;
  width: 10px;
  height: 10px;
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
}

.corner::before,
.corner::after {
  content: '';
  position: absolute;
  background: var(--vp-c-brand-1);
}

.corner::before {
  width: 10px;
  height: 1px;
  top: 50%;
  left: 0;
}

.corner::after {
  width: 1px;
  height: 10px;
  top: 0;
  left: 50%;
}

.corner.tl {
  top: 6px;
  left: 6px;
}
.corner.tr {
  top: 6px;
  right: 6px;
}
.corner.bl {
  bottom: 6px;
  left: 6px;
}
.corner.br {
  bottom: 6px;
  right: 6px;
}

.card:hover .corner {
  opacity: 0.8;
}

/* Card head */
.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.card-logo {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  object-fit: cover;
  background: #fff;
  flex-shrink: 0;
}

.card-index {
  font-family: var(--vp-font-family-mono);
  font-size: 0.75rem;
  letter-spacing: 0.05em;
  color: var(--vp-c-text-3);
  font-variant-numeric: tabular-nums;
  transition: color 0.3s ease;
}

.card:hover .card-index {
  color: var(--vp-c-brand-1);
}

/* Card name */
.card-name {
  font-size: 1.15rem;
  font-weight: 600;
  letter-spacing: -0.015em;
  line-height: 1.2;
  margin: 0;
}

/* Card foot */
.card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: auto;
  padding-top: 0.25rem;
  border-top: 1px dashed var(--vp-c-divider);
  padding-top: 1rem;
}

.card-path {
  font-family: var(--vp-font-family-mono);
  font-size: 0.75rem;
  color: var(--vp-c-text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}

.card-cta {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-family: var(--vp-font-family-mono);
  font-size: 0.75rem;
  color: var(--vp-c-text-2);
  transition: color 0.3s ease;
  flex-shrink: 0;
}

.card:hover .card-cta {
  color: var(--vp-c-brand-1);
}

.cta-arrow {
  display: inline-block;
  transition: transform 0.3s ease;
}

.card:hover .cta-arrow {
  transform: translateX(3px);
}

/* ---------- Add your project ---------- */
.cta {
  margin-top: 3.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 2rem;
  padding: 2rem 2rem 2rem 2.25rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 14px;
  background:
    linear-gradient(
      135deg,
      color-mix(in srgb, var(--vp-c-brand-1) 6%, transparent) 0%,
      transparent 60%
    ),
    var(--vp-c-bg-soft);
  position: relative;
  overflow: hidden;
}

.cta::before {
  content: '';
  position: absolute;
  top: -1px;
  left: 2rem;
  right: 2rem;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent,
    var(--vp-c-brand-1),
    transparent
  );
  opacity: 0.5;
}

.cta-body {
  flex: 1;
  min-width: 0;
}

.cta-kicker {
  font-family: var(--vp-font-family-mono);
  font-size: 0.75rem;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-2);
  text-transform: uppercase;
  margin: 0 0 0.5rem;
}

.cta-title {
  font-size: 1.5rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0 0 0.5rem;
  color: var(--vp-c-text-1);
}

.cta-text {
  font-size: 0.95rem;
  line-height: 1.55;
  color: var(--vp-c-text-2);
  margin: 0;
}

.cta-text code {
  font-family: var(--vp-font-family-mono);
  font-size: 0.85rem;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-brand-1);
}

.cta-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border: 1px solid var(--vp-c-brand-1);
  border-radius: 999px;
  font-family: var(--vp-font-family-mono);
  font-size: 0.85rem;
  color: var(--vp-c-brand-1);
  background: transparent;
  text-decoration: none !important;
  transition:
    background 0.25s ease,
    color 0.25s ease,
    transform 0.25s ease;
  flex-shrink: 0;
}

.cta-button:hover {
  background: var(--vp-c-brand-1);
  color: var(--vp-c-bg);
  transform: translateY(-2px);
}

.cta-button .cta-arrow {
  transition: transform 0.25s ease;
}

.cta-button:hover .cta-arrow {
  transform: translateX(4px);
}

/* ---------- Responsive ---------- */
@media (max-width: 720px) {
  .cta {
    flex-direction: column;
    align-items: flex-start;
    padding: 1.75rem;
  }
  .cta-button {
    align-self: stretch;
    justify-content: center;
  }
}
</style>
