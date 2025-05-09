'use client';

import { useCallback, useState } from 'react';
import Image from 'next/image';
import { cn } from '@/lib/utils';

export const SortCell = ({
  className,
  textClassName,
  onClick
}: {
  className?: string;
  textClassName?: string;
  onClick?: (sortState: 'asc' | 'desc' | undefined) => void;
}) => {
  const [sortState, setSortState] = useState<'asc' | 'desc' | undefined>(undefined);

  const handleClick = useCallback(() => {
    let newSortState = sortState;
    if (!sortState) {
      newSortState = 'asc';
    } else {
      newSortState = sortState === 'asc' ? 'desc' : undefined;
    }
    setSortState(newSortState);
    onClick?.(newSortState);
  }, [sortState, onClick]);

  return (
    <div
      className={cn('flex w-full cursor-pointer items-center justify-center gap-[4px]', className)}
      onClick={handleClick}
    >
      <span className={cn('text-[12px]', textClassName)}>Proposals</span>
      {!sortState && (
        <Image src="/arrow-full.svg" alt="sort" width={6} height={9} className="flex-shrink-0" />
      )}
      {sortState === 'asc' && (
        <Image src="/arrow-up.svg" alt="sort" width={6} height={9} className="flex-shrink-0" />
      )}
      {sortState === 'desc' && (
        <Image src="/arrow-down.svg" alt="sort" width={6} height={9} className="flex-shrink-0" />
      )}
    </div>
  );
};

export default SortCell;
