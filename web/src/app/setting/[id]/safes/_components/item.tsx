// {
//     id: 1,
//     name: 'Test',
//     network: 'Ethereum',
//     networkLogo: '/example/network1.svg',
//     safeLink: 'https://safe.gnosis.io/app/eth:0x0000000000000000000000000000000000000000/balances'
//   },

import Image from 'next/image';
import Link from 'next/link';
interface ItemProps {
  id: number;
  name: string;
  network: string;
  networkLogo: string;
  safeLink: string;
  onDelete: () => void;
}
export function Item({ id, name, network, networkLogo, safeLink, onDelete }: ItemProps) {
  return (
    <div className="bg-card flex items-center justify-between gap-[10px] rounded-[14px] p-[10px]">
      <div className="flex flex-col gap-[10px]">
        <p className="text-[18px] leading-[100%]">{name}</p>
        <div className="flex items-center gap-[5px]">
          <Image src={networkLogo} alt={network} width={16} height={16} />
          <p>{network}</p>
        </div>
      </div>
      <div className="flex items-center justify-end gap-[10px]">
        <Link href={safeLink} target="_blank" rel="noopener noreferrer">
          <Image src="/safe.svg" alt="safe" width={16} height={17} />
        </Link>
        <button className="cursor-pointer rounded-[100px]" onClick={onDelete}>
          <Image src="/delete.svg" alt="delete" width={20} height={20} />
        </button>
      </div>
    </div>
  );
}
