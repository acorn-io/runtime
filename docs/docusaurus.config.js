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
  trailingSlash: false,
  onBrokenMarkdownLinks: 'warn',
  onDuplicateRoutes: 'warn',
  favicon: 'img/favicon.png',
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
            'https://github.com/acorn-io/acorn/tree/main/docs/',
        },
        blog: false,
        gtag: {
          trackingID: 'G-B0PL797F38',
          anonymizeIP: true,
        },
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
        title: 'Docs',
        style: 'dark',
        logo: {
          alt: 'Acorn Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            to: 'https://acorn.io',
            label: 'Acorn Home',
            position: 'right',
            target: '_self',
          },
          {
            to: 'https://github.com/acorn-io/acorn',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            label: 'GitHub',
            to: 'https://github.com/acorn-io/acorn',
          },
          {
            label: 'Users Slack',
            to: 'https://slack.acorn.com',
          },
          {
            label: 'Twitter',
            to: 'https://twitter.com/acornlabs',
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Acorn Labs, Inc`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
        additionalLanguages: ['cue','docker'],
      },
    }),
};

module.exports = config;
