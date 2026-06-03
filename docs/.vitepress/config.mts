import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  lang: 'zh-CN',
  title: '瑞科 API',
  description: '一个密钥，接入多种 AI 服务',
  head: [['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }]],
  srcExclude: ['superpowers/**/*.md'],
  themeConfig: {
    logo: '/logo.svg',
    nav: [
      { text: '首页', link: '/' },
      { text: '中转站', link: 'https://api.ruikon.com' }
    ],

    sidebar: {
      '/guide/': [
        {
          text: '中转站使用',
          items: [
            { text: '使用流程总览', link: '/guide/relay-station#overview' },
            { text: '创建账号', link: '/guide/relay-station#account' },
            { text: '计费与订阅', link: '/guide/relay-station#billing-subscription' },
            { text: '创建 API Key', link: '/guide/relay-station#api-key' },
            { text: '选择分组', link: '/guide/relay-station#group' },
            { text: '查看用量', link: '/guide/relay-station#usage' }
          ]
        },
        {
          text: '接入使用',
          items: [
            { text: '接入前准备', link: '/guide/integration#prepare' },
            { text: '一键接入（推荐）', link: '/guide/integration#one-click' },
            { text: 'CC Switch（推荐）', link: '/guide/integration#cc-switch' },
            { text: '手动配置参考', link: '/guide/integration#manual-config' },
            { text: '接入后验证', link: '/guide/integration#verify' }
          ]
        },
        {
          text: '其他',
          items: [
            { text: '常见问题', link: '/guide/faq' }
          ]
        }
      ]
    },

    search: {
      provider: 'local'
    },

    footer: {
      message: '瑞科 API 文档',
      copyright: 'Copyright © 2026 瑞科 API'
    }
  }
})
