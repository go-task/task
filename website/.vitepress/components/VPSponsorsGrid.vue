<script setup lang="ts">
import { ref } from 'vue';
import { useSponsorsGrid } from 'vitepress/dist/client/theme-default/composables/sponsor-grid';
import type { Sponsor } from '../sponsors';

interface Props {
  size?: 'xmini' | 'mini' | 'small' | 'medium' | 'big';
  data: Sponsor[];
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium'
});

const el = ref(null);

useSponsorsGrid({ el, size: props.size });
</script>

<template>
  <div class="VPSponsorsGrid vp-sponsor-grid" :class="[size]" ref="el">
    <div
      v-for="sponsor in data"
      :key="sponsor.name"
      class="vp-sponsor-grid-item"
      :class="{ 'has-dark-logo': sponsor.imgDark }"
    >
      <a
        class="vp-sponsor-grid-link"
        :href="sponsor.url"
        target="_blank"
        rel="sponsored noopener"
      >
        <article class="vp-sponsor-grid-box">
          <h4 class="visually-hidden">{{ sponsor.name }}</h4>
          <!-- Both logos are rendered and one is hidden in CSS, so the right
               one is already in place before hydration. -->
          <img
            class="vp-sponsor-grid-image logo-light"
            :src="sponsor.img"
            :alt="sponsor.name"
          />
          <img
            v-if="sponsor.imgDark"
            class="vp-sponsor-grid-image logo-dark"
            :src="sponsor.imgDark"
            :alt="sponsor.name"
          />
        </article>
      </a>
      <!-- Sits inside the tile: the grid clips anything that overflows it. -->
      <span v-if="sponsor.description" class="tooltip">
        {{ sponsor.description }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.vp-sponsor-grid-item {
  position: relative;
}

.tooltip {
  position: absolute;
  bottom: 14px;
  left: 50%;
  transform: translate(-50%, 6px);
  max-width: calc(100% - 28px);
  padding: 7px 12px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 10px;
  background-color: var(--vp-c-bg-elv);
  box-shadow: var(--vp-shadow-2);
  color: var(--vp-c-text-1);
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
  text-align: center;
  text-wrap: balance;
  opacity: 0;
  transition:
    opacity 0.2s,
    transform 0.2s;
  pointer-events: none;
}

.vp-sponsor-grid-item:hover .tooltip,
.vp-sponsor-grid-item:focus-within .tooltip {
  opacity: 1;
  transform: translate(-50%, 0);
}

@media (prefers-reduced-motion: reduce) {
  .tooltip {
    transition: none;
  }
}
</style>
