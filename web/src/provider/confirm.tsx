'use client';

import { ReactNode, useState } from 'react';
import { ConfirmContext, type ConfirmOptions } from '@/contexts/confirm-context';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
interface ConfirmProviderProps {
  children: ReactNode;
}

export function ConfirmProvider({ children }: ConfirmProviderProps) {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<ConfirmOptions | null>(null);
  const [resolveRef, setResolveRef] = useState<((value: boolean) => void) | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const confirm = (options: ConfirmOptions) => {
    setOptions(options);
    setOpen(true);

    return new Promise<boolean>((resolve) => {
      setResolveRef(() => resolve);
    });
  };

  const handleConfirm = async () => {
    if (options?.onConfirm) {
      setIsLoading(true);
      try {
        await options.onConfirm();
        if (resolveRef) resolveRef(true);
      } catch (error) {
        console.error('Error during confirmation:', error);
      } finally {
        setIsLoading(false);
        setOpen(false);
      }
    } else {
      if (resolveRef) resolveRef(true);
      setOpen(false);
    }
  };

  const handleCancel = () => {
    if (resolveRef) resolveRef(false);
    setOpen(false);
  };

  return (
    <ConfirmContext.Provider value={{ confirm }}>
      {children}
      {options && (
        <ConfirmDialog
          open={open}
          onOpenChange={handleCancel}
          title={options.title}
          description={options.description}
          cancelText={options.cancelText}
          confirmText={options.confirmText}
          variant={options.variant}
          isLoading={isLoading || options.isLoading}
          onConfirm={handleConfirm}
        />
      )}
    </ConfirmContext.Provider>
  );
}
