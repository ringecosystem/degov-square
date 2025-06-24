'use client';

import Image from 'next/image';
import { useState, useRef, useEffect, useCallback } from 'react';

import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { LoadedButton } from '@/components/ui/loaded-button';
interface SearchDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isLoading?: boolean;
  placeholder?: string;
  onConfirm?: (query: string) => void;
  initialQuery?: string;
}

export function MobileSearchDialog({
  open,
  onOpenChange,
  isLoading = false,
  onConfirm,
  initialQuery = ''
}: SearchDialogProps) {
  const [searchQuery, setSearchQuery] = useState(initialQuery);
  const inputRef = useRef<HTMLInputElement>(null);

  // Update local state when initialQuery changes
  useEffect(() => {
    setSearchQuery(initialQuery);
  }, [initialQuery]);

  useEffect(() => {
    if (open && inputRef.current) {
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
    }
  }, [open]);

  // Don't clear search when dialog closes - keep the initial query
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setSearchQuery(initialQuery);
    }
    onOpenChange(newOpen);
  };

  const handleConfirm = useCallback(() => {
    onConfirm?.(searchQuery);
  }, [onConfirm, searchQuery]);

  const handleClear = useCallback(() => {
    setSearchQuery('');
    onConfirm?.('');
  }, [onConfirm]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px] md:hidden">
        <DialogHeader className="flex w-full flex-row items-center justify-between">
          <DialogTitle>Search</DialogTitle>
        </DialogHeader>
        <div className="bg-card flex w-full items-center gap-[10px] rounded-[19px] border px-[17px] py-[9px]">
          <Image src="/search.svg" alt="search" width={16} height={16} />
          <input
            ref={inputRef}
            className="placeholder:text-muted-foreground h-[17px] w-full outline-none placeholder:text-[14px]"
            placeholder="Search by Name, Chain"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                handleConfirm();
              }
            }}
          />
          {searchQuery && (
            <button
              onClick={handleClear}
              className="text-muted-foreground hover:text-foreground flex items-center justify-center"
              title="Clear search"
            >
              <Image src="/close.svg" alt="clear" width={12} height={12} />
            </button>
          )}
        </div>
        <div className="flex gap-2">
          <LoadedButton
            className="flex-1 rounded-[100px]"
            variant="default"
            isLoading={isLoading}
            onClick={handleConfirm}
          >
            Search
          </LoadedButton>
          {searchQuery && (
            <LoadedButton className="rounded-[100px]" variant="outline" onClick={handleClear}>
              Clear
            </LoadedButton>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
