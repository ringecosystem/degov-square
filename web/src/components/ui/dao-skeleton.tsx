import { Skeleton } from './skeleton';

export function DaoItemSkeleton() {
  return (
    <div className="bg-card flex items-center justify-between rounded-[20px] border p-[20px]">
      <div className="flex items-center gap-[10px]">
        <Skeleton className="h-[34px] w-[34px] rounded-full" />
        <Skeleton className="h-[20px] w-[120px]" />
      </div>
      <div className="flex items-center gap-[10px]">
        <Skeleton className="h-[24px] w-[24px] rounded-full" />
        <Skeleton className="h-[16px] w-[80px]" />
      </div>
      <Skeleton className="h-[16px] w-[40px]" />
    </div>
  );
}

export function DaoListSkeleton() {
  return (
    <div className="flex flex-col gap-[10px] md:hidden">
      {Array.from({ length: 6 }).map((_, index) => (
        <DaoItemSkeleton key={index} />
      ))}
    </div>
  );
}

export function DaoTableSkeleton() {
  return (
    <div className="hidden rounded-lg border md:block">
      <div className="border-b p-4">
        <div className="grid grid-cols-[34%_28%_28%] gap-4">
          <Skeleton className="h-[20px] w-[60px]" />
          <Skeleton className="h-[20px] w-[80px] justify-self-center" />
          <Skeleton className="h-[20px] w-[70px] justify-self-center" />
        </div>
      </div>
      {Array.from({ length: 3 }).map((_, index) => (
        <div key={index} className="border-b p-4 last:border-b-0">
          <div className="grid grid-cols-[34%_28%_28%] gap-4">
            <div className="flex items-center gap-[10px]">
              <Skeleton className="h-[34px] w-[34px] rounded-full" />
              <Skeleton className="h-[16px] w-[100px]" />
            </div>
            <div className="flex items-center justify-center gap-[10px]">
              <Skeleton className="h-[24px] w-[24px] rounded-full" />
              <Skeleton className="h-[16px] w-[70px]" />
            </div>
            <Skeleton className="h-[16px] w-[30px] justify-self-center" />
          </div>
        </div>
      ))}
    </div>
  );
}
