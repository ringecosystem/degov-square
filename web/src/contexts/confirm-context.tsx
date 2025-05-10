import { createContext, useContext } from 'react';

import type { ConfirmDialogProps } from '@/components/ui/confirm-dialog';

export type ConfirmOptions = Omit<ConfirmDialogProps, 'open' | 'onOpenChange'>;

export interface ConfirmContextValue {
  confirm: (options: ConfirmOptions) => Promise<boolean>;
}

export const ConfirmContext = createContext<ConfirmContextValue | undefined>(undefined);

export function useConfirm() {
  const context = useContext(ConfirmContext);
  if (!context) {
    throw new Error('useConfirm must be used within a ConfirmProvider');
  }
  return context;
}
