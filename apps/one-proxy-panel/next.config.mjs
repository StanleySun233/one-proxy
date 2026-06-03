import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./i18n/request.ts');

export default withNextIntl({
  output: 'standalone',
  async redirects() {
    return [
      {
        source: '/:locale(zh|en)/chains/access-paths',
        destination: '/:locale/chains/network',
        permanent: false
      },
      {
        source: '/chains/access-paths',
        destination: '/chains/network',
        permanent: false
      }
    ];
  },
  webpack(config, {dev}) {
    if (dev) {
      config.watchOptions = {
        poll: 1000,
        aggregateTimeout: 300
      };
    }

    return config;
  }
});
