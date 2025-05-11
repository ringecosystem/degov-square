'use client';
import { redirect } from 'next/navigation';
import { useMobile } from '@/hooks/useMobile';

export default function NotificationPage() {
  const isMobile = useMobile();

  if (isMobile) {
    return null;
  }

  redirect('/notification/subscription');
}
