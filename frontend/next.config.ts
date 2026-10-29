import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";
import {
  assertProductionRuntimeFlags,
  assertVercelDemoHasNoBackend,
  getRequiredVercelBackendOrigin,
} from "./lib/production-runtime-config";

const withNextIntl = createNextIntlPlugin('./i18n/request.ts');

// Check if running on Vercel
const isVercel = process.env.VERCEL === '1';
const demoSite = process.env.DEMO_SITE === 'true'
const useMock = process.env.NEXT_PUBLIC_USE_MOCK === 'true'
const skipAuth = process.env.NEXT_PUBLIC_SKIP_AUTH === 'true'

assertProductionRuntimeFlags({ useMock, skipAuth, demoSite, isVercel })
if (demoSite && process.env.NODE_ENV === "production") {
  assertVercelDemoHasNoBackend()
}
const vercelBackendOrigin = isVercel && process.env.NODE_ENV === "production" && !demoSite
  ? getRequiredVercelBackendOrigin()
  : null

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
  // Allow local and LAN browser smoke origins to request dev assets.
  allowedDevOrigins: ['localhost', '127.0.0.1', '::1', '192.168.*.*', '10.*.*.*', '172.16.*.*'],

  // Optimize package imports to reduce bundle size and improve performance
  experimental: {
    optimizePackageImports: [
      // Icons
      '@tabler/icons-react',
      // Charts
      'recharts',
      // 3D / WebGL
      'three',
      'postprocessing',
      // Diagram rendering
      'mermaid',
      // Animation
      'framer-motion',
      'gsap',
      'motion',
      // Flow editor
      '@xyflow/react',
      // Code editor
      '@codemirror/commands',
      '@codemirror/lang-yaml',
      '@codemirror/language',
      '@codemirror/state',
      '@codemirror/theme-one-dark',
      '@codemirror/view',
      'codemirror',
      // Terminal emulator
      '@xterm/xterm',
      '@xterm/addon-fit',
      '@xterm/addon-web-links',
    ],
  },


  async rewrites() {
    // Mock mode owns /v1 in the browser, so a request escaping MSW must never reach a real backend.
    if (useMock) {
      return [];
    }
    if (isVercel) {
      if (!vercelBackendOrigin) return [];
      return [
        {
          source: '/v1/:path*',
          destination: `${vercelBackendOrigin}/v1/:path*`,
        },
        {
          source: '/mcp',
          destination: `${vercelBackendOrigin}/mcp`,
        },
      ];
    }
    // Use server service name in Docker environment, localhost for local development
    const apiHost = process.env.API_HOST || 'localhost';
    return [
      // Proxy the versioned backend API during local Next.js development.
      {
        source: '/v1/:path*',
        destination: `http://${apiHost}:8080/v1/:path*`,
      },
      // Client configuration uses the frontend origin; direct dev access also
      // needs this protocol route when it does not pass through Nginx.
      {
        source: '/mcp',
        destination: `http://${apiHost}:8080/mcp`,
      },
    ];
  },
};

// Force restart
export default withNextIntl(nextConfig);
