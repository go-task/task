export interface Sponsor {
  name: string;
  url: string;
  img: string;
  // Logo variant for dark backgrounds. Sponsors that set one keep their own
  // colours instead of being inverted by the sponsor grid.
  imgDark?: string;
  // Tooltip explaining what this sponsor gives the project.
  description?: string;
}

export const sponsors: { tier: string; size: string; items: Sponsor[] }[] = [
  {
    tier: 'Gold Sponsors',
    size: 'big',
    items: [
      {
        name: 'devowl',
        url: 'https://devowl.io/',
        img: '/img/devowl.io.svg'
      },
      {
        name: 'GoodX',
        url: 'https://goodx.international/',
        img: '/img/goodx.svg'
      },
      {
        name: 'Magic',
        url: 'https://magic.dev/',
        img: '/img/magic.png'
      }
    ]
  },
  {
    tier: 'Community Sponsors',
    size: 'big',
    items: [
      {
        name: 'Cloudsmith',
        url: 'https://cloudsmith.com/',
        img: '/img/cloudsmith.svg',
        description: 'Hosts the deb and rpm package registries for free'
      },
      {
        name: 'JetBrains',
        url: 'https://jb.gg/OpenSource',
        img: '/img/jetbrains.svg',
        imgDark: '/img/jetbrains-mono.svg',
        description:
          'Provides free All Products Pack licenses to the maintainers'
      }
    ]
  }
];
