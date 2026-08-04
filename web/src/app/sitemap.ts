import type { MetadataRoute } from 'next';

export default function sitemap(): MetadataRoute.Sitemap {
  return [
    {
      url: 'https://square.degov.ai/',
      changeFrequency: 'daily',
      priority: 1
    }
  ];
}
