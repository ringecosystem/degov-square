import { Empty } from '@/components/ui/empty';
import { Skeleton } from '@/components/ui/skeleton';

import { Item } from './item';

interface SafeData {
  id: number;
  name: string;
  network: string;
  networkLogo: string;
  safeLink: string;
}

interface SafeListProps {
  safes: SafeData[];
  isLoading?: boolean;
  onDelete: (id: number) => void;
}

export const SafeItemSkeleton = () => {
  return (
    <div className="bg-card flex items-center justify-between gap-[10px] rounded-[14px] p-[10px]">
      <div className="flex flex-col gap-[10px]">
        <Skeleton className="h-6 w-[120px]" />
        <div className="flex items-center gap-[5px]">
          <Skeleton className="size-[16px] rounded-full" />
          <Skeleton className="h-4 w-[80px]" />
        </div>
      </div>
      <div className="flex items-center justify-end gap-[10px]">
        <Skeleton className="size-[16px]" />
        <Skeleton className="size-[20px]" />
      </div>
    </div>
  );
};

export const SafeList = ({ safes, isLoading, onDelete }: SafeListProps) => {
  return (
    <div className="flex flex-col gap-[10px] md:hidden">
      {isLoading ? (
        <>
          <SafeItemSkeleton />
          <SafeItemSkeleton />
          <SafeItemSkeleton />
        </>
      ) : safes.length === 0 ? (
        <div className="py-8 text-center">
          <Empty label="No safes found" />
        </div>
      ) : (
        safes.map((safe) => <Item key={safe.id} {...safe} onDelete={() => onDelete(safe.id)} />)
      )}
    </div>
  );
};
