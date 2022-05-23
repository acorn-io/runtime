// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Acorn Docs',
  tagline: 'Welcome to Acorn Docs',
  url: 'http://docs.acorn.io',
  baseUrl: '/',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'img/favicon.ico',
  organizationName: 'acorn-io',
  projectName: 'acorn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: '/', // Serve the docs at the site's root
          sidebarPath: require.resolve('./sidebars.js'),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/acorn-io/acorn/tree/main/docs/docs/',
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'Acorn Docs',
        logo: {
          alt: 'Acorn Logo',
          src: 'img/logo.png',
        },
        items: [
          {
            href: 'https://acorn.io',
            label: 'Acorn Project Home',
            position: 'right',
            target: '_self',
          },
          {
            href: 'https://github.com/acorn-io/acorn',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'GitHub',
            items: [
              {
                label: 'GitHub',
                to: 'https://github.com/acorn-io/acorn',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'Slack Channel',
                href: 'https://slack.com',
              },
              {
                label: 'Twitter',
                href: 'https://twitter.com',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Acorn Labs, Inc. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
      },
    }),
};

module.exports = config;
