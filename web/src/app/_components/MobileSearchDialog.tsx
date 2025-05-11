'use client';

import { useState, useRef, useEffect, useCallback } from 'react';
import Image from 'next/image';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { LoadedButton } from '@/components/ui/loaded-button';
interface SearchDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isLoading?: boolean;
  placeholder?: string;
  onConfirm?: (query: string) => void;
}

export function MobileSearchDialog({
  open,
  onOpenChange,
  isLoading = false,
  onConfirm
}: SearchDialogProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open && inputRef.current) {
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
    }
  }, [open]);

  // Clear search when dialog closes
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setSearchQuery('');
    }
    onOpenChange(newOpen);
  };

  const handleConfirm = useCallback(() => {
    onConfirm?.(searchQuery);
  }, [onConfirm, searchQuery]);

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px] md:hidden">
        <DialogHeader className="flex w-full flex-row items-center justify-between">
          <DialogTitle>Search</DialogTitle>
        </DialogHeader>
        <div className="bg-card flex w-full items-center gap-[10px] rounded-[19px] border px-[17px] py-[9px]">
          <Image src="/search.svg" alt="search" width={16} height={16} />
          <input
            className="placeholder:text-muted-foreground h-[17px] outline-none placeholder:text-[14px]"
            placeholder="Search by Name, Chain"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <LoadedButton
          className="rounded-[100px]"
          variant="default"
          isLoading={isLoading}
          onClick={handleConfirm}
          disabled={!searchQuery}
        >
          Search
        </LoadedButton>
      </DialogContent>
    </Dialog>
  );
}
