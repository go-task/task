<template>
  <div class="author-compact" v-if="author">
    <img :src="author.avatar" :alt="author.name" class="author-avatar" />
    <div class="author-info">
      <div class="author-name-line">
        <span class="author-name">{{ author.name }}</span>

        <div class="author-socials">
          <a
            v-for="{ link, icon } in author.links"
            :key="link"
            :href="link"
            target="_blank"
            class="social-link"
          >
            <span :class="`vpi-social-${icon}`"></span>
          </a>
        </div>
      </div>
      <span class="author-bio">{{ author.title }}</span>
    </div>
  </div>
</template>

<script setup>
import { team } from '../team.ts';
import { computed } from 'vue';
const props = defineProps({
  author: String
});

const author = computed(() => {
  return team.find((m) => m.slug === props.author) || null;
});
</script>

<style scoped>
.author-compact {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin: 1.5rem 0;
}

.author-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
}

.author-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  flex: 1;
}

.author-name-line {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.author-name {
  font-weight: 600;
  color: var(--vp-c-text-1);
  font-size: 0.95rem;
}

.author-bio {
  color: var(--vp-c-text-2);
  font-size: 0.85rem;
}

.author-socials {
  display: flex;
  gap: 0.5rem;
}

.social-link {
  color: var(--vp-c-text-2);
  transition: color 0.2s;
  display: flex;
  align-items: center;
}

.social-link:hover {
  color: var(--vp-c-brand-1);
}

@media (max-width: 768px) {
  .author-compact {
    margin-bottom: 1rem;
  }
}
</style>
