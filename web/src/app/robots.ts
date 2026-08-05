import type { MetadataRoute } from 'next';

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: '*',
      allow: '/',
      disallow: ['/add', '/oauth', '/notification', '/setting']
    },
    sitemap: ['https://square.degov.ai/sitemap.xml']
  };
}
