'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function NotificationPage() {
  const router = useRouter();

  useEffect(() => {
    router.push('/notification/subscription');
  }, [router]);

  return null;
}
