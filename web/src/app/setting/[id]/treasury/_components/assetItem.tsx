import Image from 'next/image';

interface AssetItemProps {
  id: number;
  name: string;
  tokenLogo: string;
  balance: string;
  value: string;
  onDelete: () => void;
}

export function AssetItem({ name, tokenLogo, balance, value, onDelete }: AssetItemProps) {
  return (
    <div className="flex w-full items-center gap-[10px]">
      <div className="bg-card flex flex-1 items-center justify-between gap-[10px] rounded-[14px] p-[10px]">
        <div className="flex items-center gap-[5px]">
          <Image src={tokenLogo} alt={name} width={30} height={30} />
          <span>{name}</span>
          <Image src="/external-link.svg" alt="arrow" width={16} height={16} />
        </div>
        <div className="flex flex-col gap-[10px]">
          <span className="flex items-center justify-end gap-[5px]">
            <span className="text-muted-foreground text-[12px]">Value</span>
            <span className="text-[12px]">{value}</span>
          </span>
          <span className="flex items-center justify-end gap-[5px]">
            <span className="text-muted-foreground text-[12px]">Balance</span>
            <span className="text-[12px]">{balance}</span>
          </span>
        </div>
      </div>
      <button onClick={onDelete}>
        <Image src="/delete.svg" alt="delete" width={16} height={16} />
      </button>
    </div>
  );
}
