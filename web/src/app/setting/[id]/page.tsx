'use client';
import { redirect } from 'next/navigation';
import { useMobile } from '@/hooks/useMobile';
import { useParams } from 'next/navigation';

export default function SettingPage() {
  const isMobile = useMobile();
  const params = useParams();
  const { id } = params || {};

  if (isMobile) {
    return null;
  }

  redirect(`/setting/${id}/basic`);
}
