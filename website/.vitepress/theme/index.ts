import DefaultTheme from 'vitepress/theme';
import type { Theme } from 'vitepress';
import './custom.css';
import HomePage from '../components/HomePage.vue';
import AuthorCard from '../components/AuthorCard.vue';
import BlogPost from '../components/BlogPost.vue';
import Version from '../components/Version.vue';
import { enhanceAppWithTabs } from 'vitepress-plugin-tabs/client';
import { h } from 'vue';
import 'virtual:group-icons.css';
import CopyOrDownloadAsMarkdownButtons from 'vitepress-plugin-llms/vitepress-components/CopyOrDownloadAsMarkdownButtons.vue';

export default {
  extends: DefaultTheme,
  Layout() {
    return h(DefaultTheme.Layout, null, {
      'home-features-after': () => h(HomePage)
    });
  },
  enhanceApp({ app }) {
    app.component('AuthorCard', AuthorCard);
    app.component('BlogPost', BlogPost);
    app.component('Version', Version);
    app.component('CopyOrDownloadAsMarkdownButtons', CopyOrDownloadAsMarkdownButtons);
    enhanceAppWithTabs(app);
  }
} satisfies Theme;
