---
layout: page
---
<script setup>
import {
  VPTeamPage,
  VPTeamPageTitle,
  VPTeamMembers
} from 'vitepress/theme'

const members = [
  {
    avatar: 'https://www.github.com/andreynering.png',
    name: 'Andrey Nering',
    title: 'Creator & Maintainer',
    sponsor: 'https://github.com/sponsors/andreynering',
    links: [
      { icon: 'github', link: 'https://github.com/andreynering' },
      { icon: 'x', link: 'https://x.com/andreynering' }
    ]
  },
  {
    avatar: 'https://www.github.com/pd93.png',
    name: 'Pete Davison',
    title: 'Maintainer',
    sponsor: 'https://github.com/sponsors/pd93',
    links: [
      { icon: 'github', link: 'https://github.com/pd93' },
      { icon: 'bluesky', link: 'https://bsky.app/profile/pd93.uk' }
    ]
  },
  {
    avatar: 'https://www.github.com/vmaerten.png',
    name: 'Valentin Maerten',
    title: 'Maintainer',
    sponsor: 'https://github.com/sponsors/vmaerten',
    links: [
      { icon: 'github', link: 'https://github.com/vmaerten' },
      { icon: 'x', link: 'https://x.com/vmaerten' },
      { icon: 'bluesky', link: 'https://bsky.app/profile/vmaerten.bsky.social' }
    ]
  }

]
</script>

<VPTeamPage>
  <VPTeamPageTitle>
    <template #title>
      Our Team
    </template>
    <template #lead>
      The development of VitePress is guided by an international
      team, some of whom have chosen to be featured below.
    </template>
  </VPTeamPageTitle>
  <VPTeamMembers :members />
</VPTeamPage>
