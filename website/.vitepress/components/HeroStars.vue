<script setup lang="ts">
import { data as stars } from '../githubStars.data';
</script>

<template>
  <a
    class="hero-stars"
    href="https://github.com/go-task/task"
    target="_blank"
    rel="noreferrer"
    data-umami-event="home-hero-github"
  >
    <span class="vpi-social-github" aria-hidden="true"></span>
    {{ stars.label }} stars on GitHub
  </a>
</template>

<style scoped>
/* inline-flex so the hero container's own text-align keeps this in step with
   the buttons: centred below 960px, left-aligned above. */
.hero-stars {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  margin-top: 20px;
  color: var(--vp-c-text-2);
  font-size: 0.85rem;
  font-weight: 500;
  text-decoration: none;
  transition: color 0.25s;
}

.hero-stars:hover,
.hero-stars:focus-visible {
  color: var(--vp-c-brand-1);
}
</style>
