import { Empty } from '@/components/ui/empty';
import { Skeleton } from '@/components/ui/skeleton';
import { AssetItem } from './assetItem';

interface AssetData {
  id: number;
  name: string;
  tokenLogo: string;
  balance: string;
  value: string;
}

interface AssetListProps {
  title: string;
  assets: AssetData[];
  isLoading?: boolean;
  onDelete: (id: number) => void;
}

export const AssetItemSkeleton = () => {
  return (
    <div className="flex w-full items-center gap-[10px]">
      <div className="bg-card flex flex-1 items-center justify-between gap-[10px] rounded-[14px] p-[10px]">
        <div className="flex items-center gap-[5px]">
          <Skeleton className="size-[30px] rounded-full" />
          <Skeleton className="h-5 w-[80px]" />
          <Skeleton className="size-[16px]" />
        </div>
        <div className="flex flex-col gap-[10px]">
          <span className="flex items-center justify-end gap-[5px]">
            <span className="text-muted-foreground text-[12px]">Value</span>
            <Skeleton className="h-4 w-[60px]" />
          </span>
          <span className="flex items-center justify-end gap-[5px]">
            <span className="text-muted-foreground text-[12px]">Balance</span>
            <Skeleton className="h-4 w-[60px]" />
          </span>
        </div>
      </div>
      <Skeleton className="size-[16px]" />
    </div>
  );
};

export const AssetList = ({ title, assets, isLoading, onDelete }: AssetListProps) => {
  return (
    <div className="flex flex-col gap-[10px] md:hidden">
      {isLoading ? (
        <>
          <AssetItemSkeleton />
          <AssetItemSkeleton />
          <AssetItemSkeleton />
        </>
      ) : assets.length === 0 ? (
        <div className="py-8 text-center">
          <Empty label={`No ${title} found`} />
        </div>
      ) : (
        assets.map((asset) => (
          <AssetItem key={asset.id} {...asset} onDelete={() => onDelete(asset.id)} />
        ))
      )}
    </div>
  );
};
