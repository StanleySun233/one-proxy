import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./i18n/request.ts');

export default withNextIntl({
  output: 'standalone',
  async headers() {
    if (process.env.NODE_ENV !== 'production') {
      return [];
    }

    return [
      {
        source: '/:path*',
        headers: [
          {key: 'Strict-Transport-Security', value: 'max-age=31536000; includeSubDomains'},
          {key: 'X-Content-Type-Options', value: 'nosniff'},
          {key: 'X-Frame-Options', value: 'DENY'},
          {key: 'Referrer-Policy', value: 'no-referrer'},
          {key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=(), interest-cohort=()'},
          {key: 'Cross-Origin-Opener-Policy', value: 'same-origin'},
          {key: 'Cross-Origin-Resource-Policy', value: 'same-origin'},
          {key: 'X-DNS-Prefetch-Control', value: 'off'}
        ]
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
