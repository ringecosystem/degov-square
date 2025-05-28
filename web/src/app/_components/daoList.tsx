import { Empty } from '@/components/ui/empty';
import { Skeleton } from '@/components/ui/skeleton';
import type { daoInfo } from '@/data/daoInfo';

import { DaoItem } from './daoItem';


interface DaoListProps {
  daoInfo: typeof daoInfo;
  isLoading?: boolean;
}

export const DaoListSkeleton = () => {
  return (
    <div className="flex flex-col gap-[10px] md:hidden">
      {Array(3)
        .fill(0)
        .map((_, index) => (
          <div
            key={index}
            className="bg-card flex items-center gap-[10px] rounded-[14px] border p-[15px]"
          >
            <Skeleton className="size-[60px] flex-shrink-0 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-5 w-[120px]" />
              <Skeleton className="h-4 w-[180px]" />
            </div>
          </div>
        ))}
    </div>
  );
};

export const DaoList = ({ daoInfo, isLoading }: DaoListProps) => {
  if (isLoading) {
    return <DaoListSkeleton />;
  }

  return daoInfo.length === 0 ? (
    <div className="text-muted-foreground py-8 text-center md:hidden">
      <Empty label="No DAOs found" />
    </div>
  ) : (
    <div className="flex flex-col gap-[10px] md:hidden">
      {daoInfo?.map((v) => {
        return <DaoItem {...v} key={v.id} />;
      })}
    </div>
  );
};
