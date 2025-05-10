'use client';

import React, { forwardRef } from 'react';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

export interface InputAddonProps extends React.InputHTMLAttributes<HTMLInputElement> {
  suffix?: string;
  prefix?: string;
  suffixClassName?: string;
  prefixClassName?: string;
  containerClassName?: string;
}

const InputAddon = forwardRef<HTMLInputElement, InputAddonProps>(
  (
    { className, suffix, prefix, suffixClassName, prefixClassName, containerClassName, ...props },
    ref
  ) => {
    return (
      <div className={cn('flex items-end gap-2', containerClassName)}>
        {prefix && (
          <div
            className={cn(
              'border-input bg-muted flex h-9 items-center rounded-md border px-3',
              prefixClassName
            )}
          >
            {prefix}
          </div>
        )}

        <div className={cn('flex-1', !prefix && !suffix && 'w-full')}>
          <Input ref={ref} className={className} {...props} />
        </div>

        {suffix && (
          <div
            className={cn(
              'border-input bg-muted flex h-9 items-center rounded-md border px-3',
              suffixClassName
            )}
          >
            {suffix}
          </div>
        )}
      </div>
    );
  }
);

InputAddon.displayName = 'InputAddon';

export { InputAddon };
