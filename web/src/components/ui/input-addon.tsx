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
  inputContainerClassName?: string;
}

const InputAddon = forwardRef<HTMLInputElement, InputAddonProps>(
  (
    {
      className,
      suffix,
      prefix,
      suffixClassName,
      prefixClassName,
      containerClassName,
      inputContainerClassName,
      ...props
    },
    ref
  ) => {
    return (
      <div className={cn('flex w-full gap-2', containerClassName)}>
        {prefix && (
          <div
            className={cn(
              'border-input bg-muted flex h-10 items-center rounded-md px-3',
              prefixClassName
            )}
          >
            {prefix}
          </div>
        )}

        <div
          className={cn(
            'flex-1',
            prefix && !suffix && 'w-full',
            !prefix && suffix && 'w-full',
            inputContainerClassName
          )}
        >
          <Input ref={ref} className={cn(className)} {...props} />
        </div>

        {suffix && (
          <div
            className={cn(
              'border-input bg-background flex h-9 items-center rounded-md border px-4 text-sm',
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
