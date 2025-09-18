import Image from 'next/image';
import Link from 'next/link';

import { ProposalState } from '@/components/proposal-status';
import { ProposalStatus } from '@/components/proposal-status';
import { formatTimeAgo } from '@/utils/helper';

type ItemProps = {
  name: string;
  proposalLink: string;
  createdAt: string;
  status: string | ProposalState;
  onRemove: () => void;
};

export const Item = ({ name, proposalLink, createdAt, status, onRemove }: ItemProps) => {
  const normalizedStatus =
    typeof status === 'string'
      ? ProposalState[status as keyof typeof ProposalState] || ProposalState.Pending
      : status;

  return (
    <div className="flex items-center justify-between gap-[10px]">
      <div className="bg-card flex flex-1 flex-col gap-[10px] rounded-[14px] p-[10px]">
        <h3 className="text-[14px]">
          {proposalLink ? (
            <Link
              href={proposalLink}
              target="_blank"
              rel="noopener noreferrer"
              className="transition-opacity hover:underline hover:opacity-80"
              title={name}
            >
              {name}
            </Link>
          ) : (
            <span>{name}</span>
          )}
        </h3>
        <div className="flex items-center justify-between gap-[10px]">
          <span className="text-muted-foreground text-[12px]">{formatTimeAgo(createdAt)}</span>
          <ProposalStatus status={normalizedStatus} />
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
