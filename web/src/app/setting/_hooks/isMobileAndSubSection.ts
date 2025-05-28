'use client';
import { usePathname } from 'next/navigation';

import { useMobile } from '@/hooks/useMobile';
import { getPathLevel } from '@/utils/helper';

export function useIsMobileAndSubSection() {
  const isMobile = useMobile();
  const pathname = usePathname();
  const { section } = getPathLevel({ pathname, moduleName: 'setting' });

  return isMobile && section;
}
