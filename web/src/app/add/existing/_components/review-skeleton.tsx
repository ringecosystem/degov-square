'use client';

import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';

/**
 * 信息块骨架屏组件
 */
function InfoBlockSkeleton({ title, rows = 3 }: { title: string; rows?: number }) {
  return (
    <div className="bg-background rounded-lg p-4">
      <h4 className="mb-4 text-lg font-bold">{title}</h4>
      <div className="space-y-2">
        {Array(rows)
          .fill(0)
          .map((_, index) => (
            <div key={index} className="flex">
              <Skeleton className="h-5 w-1/3" />
              <Skeleton className="ml-2 h-5 w-2/3" />
            </div>
          ))}
      </div>
    </div>
  );
}

/**
 * Review页面的骨架屏组件
 */
export function ReviewSkeleton() {
  return (
    <>
      <h3 className="text-[18px] font-semibold">
        Review all the information of the DAO before proceeding to build the DAO.
      </h3>

      <div className="mt-4 flex flex-col gap-[15px] md:gap-[20px]">
        <InfoBlockSkeleton title="Basic" rows={3} />
        <InfoBlockSkeleton title="Governor" rows={5} />
        <InfoBlockSkeleton title="Token" rows={5} />
        <InfoBlockSkeleton title="TimeLock" rows={2} />

        <Separator className="my-0" />

        <div className="grid grid-cols-[1fr_1fr] gap-[20px] md:flex md:justify-between">
          <Skeleton className="h-10 w-auto rounded-full p-[10px] md:w-[140px]" />
          <Skeleton className="h-10 w-auto rounded-full p-[10px] md:w-[140px]" />
        </div>
      </div>
    </>
  );
}
