// id: 3,
// name: 'Proposal 3: Community Initiative',
// daoName: 'DAO Name 1',
// daoLogo: '/example/dao1.svg',
// status: ProposalState.Succeeded

import Image from 'next/image';

import type { ProposalState} from '@/components/proposal-status';
import { ProposalStatus } from '@/components/proposal-status';

type ItemProps = {
  id: number;
  name: string;
  daoName: string;
  daoLogo: string;
  status: ProposalState;
  onRemove: () => void;
};

export const Item = ({ id, name, daoName, daoLogo, status, onRemove }: ItemProps) => {
  return (
    <div className="flex items-center justify-between gap-[10px]">
      <div className="bg-card flex flex-1 flex-col gap-[10px] rounded-[14px] p-[10px]">
        <h3 className="text-[14px]">{name}</h3>
        <div className="flex items-center justify-between gap-[10px]">
          <span className="text-muted-foreground text-[12px]">Jan 7th, 2025</span>
          <ProposalStatus status={status} />
        </div>
      </div>
      <button className="cursor-pointer transition-opacity hover:opacity-80" onClick={onRemove}>
        <Image
          src="/unsubscribed.svg"
          alt="unsubscribed"
          width={20}
          height={20}
          className="h-[20px] w-[20px]"
        />
      </button>
    </div>
  );
};
