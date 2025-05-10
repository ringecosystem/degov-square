import { Button } from '@/components/ui/button';
import { LoadedButton } from '@/components/ui/loaded-button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (value: boolean) => void;
  title: string;
  description: string;
  cancelText?: string;
  confirmText?: string;
  variant?: 'default' | 'destructive';
  isLoading?: boolean;
  onConfirm?: () => void;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  cancelText,
  confirmText,
  variant = 'default',
  isLoading = false,
  onConfirm
}: ConfirmDialogProps) {
  if (!cancelText && !confirmText) {
    throw new Error('cancelText and confirmText are required');
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="border-border/20 bg-card w-[400px] rounded-[26px] p-[20px] sm:rounded-[26px]">
        <DialogHeader className="flex w-full flex-row items-center justify-between">
          <DialogTitle className="text-[18px] font-normal">{title}</DialogTitle>
        </DialogHeader>
        <Separator className="bg-muted-foreground/40 my-0" />
        <div className="w-[360px] text-[14px] leading-normal font-normal">{description}</div>
        <Separator className="bg-muted-foreground/40 my-0" />
        <div
          className={cn(
            'grid grid-cols-2 gap-[20px]',
            cancelText && confirmText ? 'grid-cols-2' : 'grid-cols-1'
          )}
        >
          {cancelText && (
            <Button
              className="border-border/20 bg-card rounded-[100px] border"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              {cancelText}
            </Button>
          )}
          {confirmText && (
            <LoadedButton
              className="rounded-[100px]"
              variant={variant}
              isLoading={isLoading}
              onClick={onConfirm}
            >
              {confirmText}
            </LoadedButton>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
