import { useMobile } from '@/hooks/useMobile';
import { getPathLevel } from '@/utils/helper';
import { usePathname } from 'next/navigation';

export function useIsMobileAndSubSection() {
  const isMobile = useMobile();
  const pathname = usePathname();
  const { section } = getPathLevel({ pathname, moduleName: 'setting' });

  return isMobile && section;
}
