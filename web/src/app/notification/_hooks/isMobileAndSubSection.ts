'use client';
import { usePathname } from 'next/navigation';

import { useMobile } from '@/hooks/useMobile';

export type GetPathLevelType = {
  pathname: string;
  moduleName: string;
};

export type PathLevelType = {
  isFirstLevel: boolean;
  isSecondLevel: boolean;
  section?: string;
};

export function getPathLevel({ pathname, moduleName }: GetPathLevelType): PathLevelType {
  const parts = pathname.split('/').filter(Boolean);
  const isFirstLevel = parts.length === 1 && parts[0] === moduleName;
  const isSecondLevel = parts.length === 2 && parts[0] === moduleName;
  const section = isSecondLevel ? parts[1] : undefined;
  return { isFirstLevel, isSecondLevel, section };
}

export function useIsMobileAndSubSection() {
  const isMobile = useMobile();
  const pathname = usePathname();

  const { section } = getPathLevel({ pathname, moduleName: 'notification' });

  return isMobile && section;
}
