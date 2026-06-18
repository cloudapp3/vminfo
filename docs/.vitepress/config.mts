import { defineConfig } from 'vitepress';
import { defineTeekConfig } from 'vitepress-theme-teek/config';

const siteTitle = 'vminfo';
const siteDescription =
  'Cross-platform host runtime information toolkit with terminal UI, JSON output, web dashboard, and embeddable Go APIs.';

const teekConfig = defineTeekConfig({
  teekTheme: true,
  teekHome: false,
  vpHome: true,
  vitePlugins: {
    mdH1: true,
    autoFrontmatter: false,
    sidebarOption: {
      ignoreList: ['assets'],
      scannerRootMd: false,
    },
  },
});

export default defineConfig({
  extends: teekConfig,

  lang: 'en-US',
  title: siteTitle,
  titleTemplate: ':title | vminfo',
  description: siteDescription,

  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: false,

  sitemap: {
    hostname: 'https://docs.example.com',
  },

  head: [
    ['meta', { name: 'theme-color', content: '#22c55e' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'vminfo Documentation' }],
    ['meta', { property: 'og:description', content: siteDescription }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['link', { rel: 'icon', href: '/favicon.svg' }],
  ],

  themeConfig: {
    logo: '/logo.svg',
    search: {
      provider: 'local',
    },
    nav: [
      { text: 'Guide', link: '/guide/quick-start' },
      { text: 'Commands', link: '/commands/' },
      { text: 'HTTP API', link: '/api' },
      { text: 'Go Library', link: '/library/' },
      { text: '中文', link: '/zh/' },
      {
        text: 'Community',
        items: [
          { text: 'GitHub', link: 'https://github.com/cloudapp3/vminfo' },
          { text: 'Contributing', link: 'https://github.com/cloudapp3/vminfo/blob/main/CONTRIBUTING.md' },
          { text: 'Changelog', link: '/changelog' },
          { text: 'Roadmap', link: '/roadmap/feature-benchmark' },
        ],
      },
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Getting Started',
          items: [
            { text: 'Quick Start', link: '/guide/quick-start' },
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Deployment', link: '/guide/deployment' },
            { text: 'Platform Support', link: '/guide/platform-support' },
          ],
        },
        {
          text: 'Using vminfo',
          items: [
            { text: 'Overview', link: '/guide/overview' },
            { text: 'Web Dashboard', link: '/guide/web-dashboard' },
            { text: 'TUI Controls', link: '/guide/tui-controls' },
          ],
        },
      ],
      '/commands/': [
        {
          text: 'Commands',
          items: [
            { text: 'Command Reference', link: '/commands/' },
            { text: 'summary', link: '/commands/summary' },
            { text: 'watch', link: '/commands/watch' },
            { text: 'ps', link: '/commands/ps' },
            { text: 'kill', link: '/commands/kill' },
            { text: 'update', link: '/commands/update' },
          ],
        },
      ],
      '/library/': [
        {
          text: 'Go Library',
          items: [
            { text: 'Library Usage', link: '/library/' },
            { text: 'Collect Metrics', link: '/library/collect' },
            { text: 'Embed TUI', link: '/library/embed-tui' },
          ],
        },
      ],
      '/roadmap/': [
        {
          text: 'Roadmap',
          items: [{ text: 'Feature Benchmark', link: '/roadmap/feature-benchmark' }],
        },
      ],
      '/zh/': [
        {
          text: '中文文档',
          items: [
            { text: '中文首页', link: '/zh/' },
            { text: '快速开始', link: '/zh/quick-start' },
            { text: '命令参考', link: '/zh/commands' },
          ],
        },
      ],
    },
    socialLinks: [{ icon: 'github', link: 'https://github.com/cloudapp3/vminfo' }],
    editLink: {
      pattern: 'https://github.com/cloudapp3/vminfo/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2026 cloudapp3',
    },
  },
});
