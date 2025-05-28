'use client';
import { redirect } from 'next/navigation';
import { useParams } from 'next/navigation';

import { useMobile } from '@/hooks/useMobile';

export default function SettingPage() {
  const isMobile = useMobile();
  const params = useParams();
  const { id } = params || {};

  if (isMobile) {
    return null;
  }

  redirect(`/setting/${id}/basic`);
}
