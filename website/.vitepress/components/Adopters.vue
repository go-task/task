<script setup lang="ts">
import { adopters } from '../adopters';

const featured = adopters.slice(0, 3);
const rest = adopters.slice(3);

const pad = (n: number) => String(n).padStart(2, '0');
const githubPath = (url: string) =>
  url.replace(/^https?:\/\/github\.com\//, '').replace(/\/$/, '');
</script>

<template>
  <article class="adopters-page">
    <header class="hero">
      <p class="kicker">
        <span class="slashes">//</span>
        Who builds with Task
      </p>
      <h1 class="title">
        Trusted by teams shipping<br />
        production software.
      </h1>
      <p class="lead">
        Thousands of open source projects use Task as their build and release
        orchestrator — from hyperscaler platforms and enterprise security tools
        to CLI utilities downloaded millions of times. Below are a few
        organizations whose public repositories ship a
        <code>Taskfile.yml</code>. Every entry links to real, production code
        you can inspect yourself.
      </p>
    </header>

    <section class="featured" aria-labelledby="featured-heading">
      <h2 id="featured-heading" class="section-title">
        <span class="slashes">//</span>
        Featured adopters
      </h2>
      <div class="featured-grid">
        <a
          v-for="item in featured"
          :key="item.name"
          :href="item.url"
          target="_blank"
          rel="noopener"
          class="featured-card"
          :aria-label="`${item.name} on GitHub`"
        >
          <span class="corner tl" aria-hidden="true"></span>
          <span class="corner tr" aria-hidden="true"></span>
          <span class="corner bl" aria-hidden="true"></span>
          <span class="corner br" aria-hidden="true"></span>

          <img
            :src="item.img"
            :alt="`${item.name} logo`"
            class="featured-logo"
            loading="lazy"
            decoding="async"
            width="64"
            height="64"
          />
          <h3 class="featured-name">{{ item.name }}</h3>
          <p class="featured-desc">{{ item.description }}</p>
          <span class="featured-cta">
            <span class="cta-label">View Taskfile on GitHub</span>
            <span class="cta-arrow" aria-hidden="true">&rarr;</span>
          </span>
        </a>
      </div>
    </section>

    <section class="grid-section" aria-labelledby="grid-heading">
      <h2 id="grid-heading" class="section-title">
        <span class="slashes">//</span>
        More projects using Task
      </h2>
      <div class="grid">
        <a
          v-for="(item, i) in rest"
          :key="item.name"
          :href="item.url"
          target="_blank"
          rel="noopener"
          class="card"
          :aria-label="`${item.name} on GitHub`"
        >
          <div class="card-head">
            <img
              :src="item.img"
              :alt="`${item.name} logo`"
              class="card-logo"
              loading="lazy"
              decoding="async"
              width="44"
              height="44"
            />
            <span class="card-index"
              >N&deg; {{ pad(i + featured.length + 1) }}</span
            >
          </div>
          <h3 class="card-name">{{ item.name }}</h3>
          <p class="card-desc">{{ item.description }}</p>
          <div class="card-foot">
            <span class="card-path">{{ githubPath(item.url) }}</span>
            <span class="card-arrow" aria-hidden="true">&rarr;</span>
          </div>
        </a>
      </div>
    </section>

    <section class="faq" aria-labelledby="why-heading">
      <h2 id="why-heading" class="section-title">
        <span class="slashes">//</span>
        Why Task?
      </h2>
      <dl class="faq-list">
        <div class="faq-item">
          <dt>Is Task production-ready?</dt>
          <dd>
            Yes. Task ships as a single static binary, has been in wide
            production use since 2018, and powers the release workflows of
            projects with millions of downloads including Arduino CLI,
            GoReleaser, FerretDB, and Gogs.
          </dd>
        </div>
        <div class="faq-item">
          <dt>Who uses Task in enterprise?</dt>
          <dd>
            Docker, Vercel, HashiCorp, Microsoft (Azure Sentinel), Google Cloud,
            AWS, and Anthropic are among the organizations that ship code with a
            <code>Taskfile.yml</code>. Task is also embedded end-to-end in
            Arduino’s developer tooling stack across more than 70 repositories.
          </dd>
        </div>
        <div class="faq-item">
          <dt>How is Task different from Make?</dt>
          <dd>
            Task uses plain YAML instead of Make’s tab-sensitive syntax, runs
            identically on Linux, macOS, and Windows, and provides built-in
            caching based on file fingerprints. It also comes with an
            <a href="/docs/integrations"
              >ecosystem of editor and CI integrations</a
            >
            that Make lacks by default.
          </dd>
        </div>
        <div class="faq-item">
          <dt>Where can I find real-world Taskfile examples?</dt>
          <dd>
            Every project above links directly to a public repository containing
            a production <code>Taskfile.yml</code>. Browsing those is the
            fastest way to see Task used in real codebases at different scales.
          </dd>
        </div>
      </dl>
    </section>

    <aside class="submit-cta">
      <div class="submit-body">
        <p class="submit-kicker">
          <span class="slashes">//</span>
          Using Task in your project?
        </p>
        <p class="submit-text">
          Open a pull request on
          <a
            href="https://github.com/go-task/task/blob/main/website/.vitepress/adopters.ts"
            target="_blank"
            rel="noopener"
            ><code>.vitepress/adopters.ts</code></a
          >
          to get featured here.
        </p>
      </div>
    </aside>
  </article>
</template>

<style scoped>
.adopters-page {
  max-width: 1152px;
  margin: 0 auto;
  padding: 0 24px 6rem;
}

.slashes {
  color: var(--vp-c-brand-1);
  margin-right: 0.4em;
}

.section-title {
  font-family: var(--vp-font-family-mono);
  font-size: 0.8rem;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-2);
  text-transform: uppercase;
  margin: 0 0 1.5rem;
}

/* ---------- Hero ---------- */
.hero {
  padding: 3.5rem 0 4rem;
  max-width: 48rem;
}

.kicker {
  font-family: var(--vp-font-family-mono);
  font-size: 0.8rem;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-2);
  text-transform: uppercase;
  margin: 0 0 1.25rem;
}

.title {
  font-size: clamp(2.25rem, 5vw, 3.5rem);
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1.05;
  margin: 0 0 1.75rem;
  color: var(--vp-c-text-1);
}

.lead {
  font-size: 1.1rem;
  line-height: 1.65;
  color: var(--vp-c-text-2);
  margin: 0;
}

.lead code {
  font-family: var(--vp-font-family-mono);
  font-size: 0.9em;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-brand-1);
}

/* ---------- Featured ---------- */
.featured {
  margin-bottom: 4rem;
}

.featured-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.25rem;
}

.featured-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 2rem 1.75rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 16px;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  text-decoration: none !important;
  transition:
    border-color 0.3s ease,
    transform 0.3s ease,
    box-shadow 0.3s ease;
  isolation: isolate;
  overflow: hidden;
}

.featured-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(
    600px circle at 50% 0%,
    color-mix(in srgb, var(--vp-c-brand-1) 14%, transparent),
    transparent 50%
  );
  opacity: 0;
  transition: opacity 0.3s ease;
  pointer-events: none;
  z-index: -1;
}

.featured-card:hover {
  border-color: color-mix(
    in srgb,
    var(--vp-c-brand-1) 50%,
    var(--vp-c-divider)
  );
  transform: translateY(-3px);
  box-shadow: 0 18px 48px -28px
    color-mix(in srgb, var(--vp-c-brand-1) 50%, transparent);
}

.featured-card:hover::before {
  opacity: 1;
}

.featured-logo {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  object-fit: cover;
  background: #fff;
}

.featured-name {
  font-size: 1.4rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0;
  line-height: 1.2;
}

.featured-desc {
  font-size: 0.95rem;
  line-height: 1.55;
  color: var(--vp-c-text-2);
  margin: 0;
  flex: 1;
}

.featured-cta {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-family: var(--vp-font-family-mono);
  font-size: 0.78rem;
  color: var(--vp-c-text-2);
  transition: color 0.3s ease;
  margin-top: 0.5rem;
}

.featured-card:hover .featured-cta {
  color: var(--vp-c-brand-1);
}

.cta-arrow {
  display: inline-block;
  transition: transform 0.3s ease;
}

.featured-card:hover .cta-arrow {
  transform: translateX(4px);
}

/* Crosshair corner marks (shared with grid cards) */
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
  top: 8px;
  left: 8px;
}
.corner.tr {
  top: 8px;
  right: 8px;
}
.corner.bl {
  bottom: 8px;
  left: 8px;
}
.corner.br {
  bottom: 8px;
  right: 8px;
}

.featured-card:hover .corner {
  opacity: 0.8;
}

/* ---------- Grid ---------- */
.grid-section {
  margin-bottom: 4rem;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 1rem;
}

.card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.25rem 1.25rem 1rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 12px;
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
  text-decoration: none !important;
  transition:
    border-color 0.25s ease,
    transform 0.25s ease,
    box-shadow 0.25s ease;
}

.card:hover {
  border-color: color-mix(
    in srgb,
    var(--vp-c-brand-1) 45%,
    var(--vp-c-divider)
  );
  transform: translateY(-2px);
  box-shadow: 0 10px 28px -22px
    color-mix(in srgb, var(--vp-c-brand-1) 40%, transparent);
}

.card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.card-logo {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  object-fit: cover;
  background: #fff;
  flex-shrink: 0;
}

.card-index {
  font-family: var(--vp-font-family-mono);
  font-size: 0.7rem;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-3);
  font-variant-numeric: tabular-nums;
  transition: color 0.25s ease;
}

.card:hover .card-index {
  color: var(--vp-c-brand-1);
}

.card-name {
  font-size: 1.05rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  line-height: 1.25;
  margin: 0;
}

.card-desc {
  font-size: 0.85rem;
  line-height: 1.5;
  color: var(--vp-c-text-2);
  margin: 0;
  flex: 1;
}

.card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding-top: 0.75rem;
  border-top: 1px dashed var(--vp-c-divider);
}

.card-path {
  font-family: var(--vp-font-family-mono);
  font-size: 0.72rem;
  color: var(--vp-c-text-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  flex: 1;
}

.card-arrow {
  font-family: var(--vp-font-family-mono);
  font-size: 0.8rem;
  color: var(--vp-c-text-2);
  transition:
    transform 0.25s ease,
    color 0.25s ease;
}

.card:hover .card-arrow {
  transform: translateX(3px);
  color: var(--vp-c-brand-1);
}

/* ---------- FAQ ---------- */
.faq {
  margin-bottom: 3.5rem;
  max-width: 48rem;
}

.faq-list {
  margin: 0;
}

.faq-item {
  padding: 1.25rem 0;
  border-bottom: 1px solid var(--vp-c-divider);
}

.faq-item:first-of-type {
  border-top: 1px solid var(--vp-c-divider);
}

.faq-item dt {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--vp-c-text-1);
  margin: 0 0 0.5rem;
  letter-spacing: -0.01em;
}

.faq-item dd {
  font-size: 0.95rem;
  line-height: 1.65;
  color: var(--vp-c-text-2);
  margin: 0;
}

.faq-item dd a {
  color: var(--vp-c-brand-1);
  font-weight: 500;
}

.faq-item dd code {
  font-family: var(--vp-font-family-mono);
  font-size: 0.88em;
  padding: 0.08rem 0.35rem;
  border-radius: 4px;
  background: var(--vp-c-bg-alt);
  color: var(--vp-c-brand-1);
}

/* ---------- Submit CTA ---------- */
.submit-cta {
  padding: 1.75rem 2rem;
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

.submit-cta::before {
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

.submit-kicker {
  font-family: var(--vp-font-family-mono);
  font-size: 0.75rem;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-2);
  text-transform: uppercase;
  margin: 0 0 0.5rem;
}

.submit-text {
  font-size: 0.95rem;
  line-height: 1.55;
  color: var(--vp-c-text-2);
  margin: 0;
}

.submit-text a {
  color: var(--vp-c-brand-1);
  font-weight: 500;
}

.submit-text code {
  font-family: var(--vp-font-family-mono);
  font-size: 0.88em;
  padding: 0.08rem 0.4rem;
  border-radius: 4px;
  background: var(--vp-c-bg-alt);
}
</style>
