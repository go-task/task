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
        :title="sponsor.description"
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
    </div>
  </div>
</template>
