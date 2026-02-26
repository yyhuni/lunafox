import type { NextConfig } from "next";
import createNextIntlPlugin from 'next-intl/plugin';

const withNextIntl = createNextIntlPlugin('./i18n/request.ts');

// Check if running on Vercel
const isVercel = process.env.VERCEL === '1';

const nextConfig: NextConfig = {
  // Use standalone mode for Docker deployment (not needed on Vercel)
  ...(isVercel ? {} : { output: 'standalone' }),
  // Disable Next.js automatic add/remove trailing slash behavior
  // Let us manually control URL format
  skipTrailingSlashRedirect: true,
  // Don't interrupt production build due to ESLint errors (keep lint in dev environment)
  eslint: {
    ignoreDuringBuilds: true,
  },
  // Allow LAN IP access to dev server (eliminate CORS warnings)
  allowedDevOrigins: ['192.168.*.*', '10.*.*.*', '172.16.*.*'],

  // Optimize package imports to reduce bundle size and improve performance
  experimental: {
    optimizePackageImports: [
      '@carbon/icons-react',
      '@tabler/icons-react',
      'lucide-react',
      '@radix-ui/react-*',
      '@radix-ui/react-select',
      '@radix-ui/react-dialog',
      '@radix-ui/react-dropdown-menu',
      '@radix-ui/react-tooltip',
      '@radix-ui/react-hover-card',
      '@radix-ui/react-tabs',
      '@radix-ui/react-avatar',
      '@radix-ui/react-checkbox',
      '@radix-ui/react-switch',
      '@radix-ui/react-radio-group',
      '@radix-ui/react-toggle',
      '@radix-ui/react-toggle-group',
      '@radix-ui/react-separator',
      '@radix-ui/react-collapsible',
      'recharts',
    ],
  },


  async rewrites() {
    // Skip rewrites on Vercel when using mock data
    if (isVercel) {
      return [];
    }
    // Use server service name in Docker environment, localhost for local development
    const apiHost = process.env.API_HOST || 'localhost';
    return [
      // Only match API paths with trailing slash
      {
        source: '/api/:path*/',
        destination: `http://${apiHost}:8080/api/:path*/`,
      },
    ];
  },
};

// Force restart
export default withNextIntl(nextConfig);
