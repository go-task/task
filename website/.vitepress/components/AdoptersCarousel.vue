<script setup lang="ts">
import { adopters } from '../adopters';

const loop = [...adopters, ...adopters];
</script>

<template>
  <section class="adopters-carousel">
    <p class="label">
      <span class="slashes">//</span>
      Trusted by open source projects
    </p>

    <div class="viewport">
      <div class="track">
        <a
          v-for="(item, i) in loop"
          :key="`${item.name}-${i}`"
          :href="item.url"
          target="_blank"
          rel="noopener"
          class="chip"
        >
          <img :src="item.img" :alt="`${item.name} logo`" class="logo" />
          <span class="name">{{ item.name }}</span>
          <span class="chevron">&rarr;</span>
        </a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.adopters-carousel {
  max-width: 1248px;
  margin: 5rem auto 2rem;
  padding: 0 24px;
}

.label {
  font-family: var(--vp-font-family-mono);
  font-size: 0.8rem;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-2);
  text-transform: uppercase;
  text-align: center;
  margin: 0 0 2rem;
}

.slashes {
  color: var(--vp-c-brand-1);
  margin-right: 0.4em;
}

.viewport {
  overflow: hidden;
  -webkit-mask-image: linear-gradient(
    90deg,
    transparent 0,
    #000 6%,
    #000 94%,
    transparent 100%
  );
  mask-image: linear-gradient(
    90deg,
    transparent 0,
    #000 6%,
    #000 94%,
    transparent 100%
  );
}

.track {
  display: flex;
  gap: 0.875rem;
  width: max-content;
  animation: scroll 55s linear infinite;
  padding: 6px 0;
}

.track:hover {
  animation-play-state: paused;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 1.125rem 0.625rem 0.625rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 999px;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  text-decoration: none !important;
  white-space: nowrap;
  transition:
    border-color 0.25s ease,
    background 0.25s ease,
    transform 0.25s ease,
    box-shadow 0.25s ease;
}

.chip:hover {
  border-color: var(--vp-c-brand-1);
  background: var(--vp-c-bg);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px -10px
    color-mix(in srgb, var(--vp-c-brand-1) 60%, transparent);
}

.logo {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  object-fit: cover;
  flex-shrink: 0;
  background: #fff;
}

.name {
  font-size: 0.9rem;
  font-weight: 500;
  letter-spacing: -0.005em;
}

.chevron {
  font-family: var(--vp-font-family-mono);
  font-size: 0.85rem;
  color: var(--vp-c-text-3);
  opacity: 0;
  transform: translateX(-4px);
  transition:
    opacity 0.25s ease,
    transform 0.25s ease,
    color 0.25s ease;
  margin-left: -0.25rem;
}

.chip:hover .chevron {
  opacity: 1;
  transform: translateX(0);
  color: var(--vp-c-brand-1);
}

@keyframes scroll {
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(calc(-50% - 0.4375rem));
  }
}

@media (max-width: 640px) {
  .adopters-carousel {
    margin-top: 3.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .track {
    animation: none;
    flex-wrap: wrap;
    justify-content: center;
    width: 100%;
  }
  .chip:hover {
    transform: none;
  }
}
</style>
