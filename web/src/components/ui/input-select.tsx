'use client';

import React, { forwardRef } from 'react';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select';

export type SelectOption = {
  value: string;
  label: string;
};

export interface InputSelectProps extends React.InputHTMLAttributes<HTMLInputElement> {
  options: SelectOption[];
  selectValue?: string;
  onSelectChange?: (value: string) => void;
  position?: 'prefix' | 'suffix';
  selectWidth?: string;
  selectClassName?: string;
  containerClassName?: string;
  selectPlaceholder?: string;
}

const InputSelect = forwardRef<HTMLInputElement, InputSelectProps>(
  (
    {
      className,
      options,
      selectValue,
      onSelectChange,
      position = 'suffix',
      selectWidth = 'w-[140px]',
      selectClassName,
      containerClassName,
      selectPlaceholder = 'Select option',
      ...props
    },
    ref
  ) => {
    const selectElement = (
      <Select value={selectValue} onValueChange={onSelectChange}>
        <SelectTrigger className={cn('h-9', selectWidth, selectClassName)}>
          <SelectValue placeholder={selectPlaceholder} />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );

    return (
      <div className={cn('flex items-end gap-2', containerClassName)}>
        {position === 'prefix' && selectElement}

        <div className="flex-1">
          <Input ref={ref} className={className} {...props} />
        </div>

        {position === 'suffix' && selectElement}
      </div>
    );
  }
);

InputSelect.displayName = 'InputSelect';

export { InputSelect };
