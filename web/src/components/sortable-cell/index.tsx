'use client';

import { useCallback } from 'react';
import { cn } from '@/lib/utils';
import { ArrowUp } from './arrow-up';
import { ArrowDown } from './arrow-down';

export const SortableCell = ({
  className,
  textClassName,
  onClick,
  sortState
}: {
  className?: string;
  textClassName?: string;
  onClick?: (sortState: 'asc' | 'desc' | undefined) => void;
  sortState?: 'asc' | 'desc' | undefined;
}) => {
  const handleClick = useCallback(() => {
    let newSortState = sortState;
    if (!sortState) {
      newSortState = 'asc';
    } else {
      newSortState = sortState === 'asc' ? 'desc' : undefined;
    }
    onClick?.(newSortState);
  }, [sortState, onClick]);

  return (
    <div
      className={cn('flex w-full cursor-pointer items-center justify-center gap-[4px]', className)}
      onClick={handleClick}
    >
      <span className={cn('text-[12px]', textClassName)}>Proposals</span>
      <span className="flex flex-col">
        <span
          style={{
            verticalAlign: '-0.125em'
          }}
        >
          <ArrowUp
            className={cn(
              sortState === 'asc' && 'opacity-100',
              sortState === 'desc' && 'opacity-50'
            )}
          />
        </span>

        <span
          className="-mt-[0.3em]"
          style={{
            verticalAlign: '-0.125em'
          }}
        >
          <ArrowDown
            className={cn(
              sortState === 'asc' && 'opacity-50',
              sortState === 'desc' && 'opacity-100'
            )}
          />
        </span>
      </span>
    </div>
  );
};
