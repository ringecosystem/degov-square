'use client';

import Image from 'next/image';
import { useState } from 'react';
import { useCallback } from 'react';

import { CustomTable } from '@/components/custom-table';
import type { ColumnType } from '@/components/custom-table';
import { ProposalState, ProposalStatus } from '@/components/proposal-status';
import { useConfirm } from '@/contexts/confirm-context';
import Link from 'next/link';
import { useIsMobileAndSubSection } from '@/app/notification/_hooks/isMobileAndSubSection';
import { Item } from './_components/item';
// Mock data for subscribed proposals
const proposalSubscriptions = [
  {
    id: 1,
    name: 'Proposal 1: Fund Allocation',
    daoName: 'DAO Name 1',
    daoLogo: '/example/dao1.svg',
    status: ProposalState.Pending
  },
  {
    id: 2,
    name: 'Enhancing Multichain Governance: Upgrading RARI Governance Token on ArbitrumEnhancing Multichain Governance: Upgrading RARI Governance Token on Arbitrum',
    daoName: 'DAO Name 2',
    daoLogo: '/example/dao2.svg',
    status: ProposalState.Active
  },
  {
    id: 3,
    name: 'Proposal 3: Community Initiative',
    daoName: 'DAO Name 1',
    daoLogo: '/example/dao1.svg',
    status: ProposalState.Succeeded
  }
];

const columns = ({ onRemove }: ColumnProps): ColumnType<any>[] => [
  {
    title: 'Proposal',
    key: 'name',
    width: '60%',
    className: 'text-left',
    style: { maxWidth: '0' },
    render: (value) => {
      return (
        <div title={value.name} className="truncate">
          {value.name}
        </div>
      );
    }
  },
  {
    title: 'Status',
    key: 'status',
    width: 250,
    className: 'text-center',
    render: (value) => {
      return (
        <div className="flex justify-center">
          <ProposalStatus status={value.status} />
        </div>
      );
    }
  },
  {
    title: 'Action',
    key: 'action',
    width: 80,
    className: 'text-right',
    render(value) {
      return (
        <button
          className="cursor-pointer transition-opacity hover:opacity-80"
          onClick={() => onRemove(value.id)}
        >
          <Image
            src="/unsubscribed.svg"
            alt="unsubscribed"
            width={20}
            height={20}
            className="h-[20px] w-[20px]"
          />
        </button>
      );
    }
  }
];

type ColumnProps = {
  onRemove: (id: number) => void;
};

export default function SubscribedProposalsPage() {
  const isMobileAndSubSection = useIsMobileAndSubSection();
  const [subscriptions, setSubscriptions] = useState(proposalSubscriptions);
  const { confirm } = useConfirm();
  const handleUnsubscribe = useCallback(
    (id: number) => {
      confirm({
        title: 'Unsubscribe',
        description: 'Are you sure you want to unsubscribe notification?',
        cancelText: 'Cancel',
        confirmText: 'Confirm',
        onConfirm: () => setSubscriptions((prev) => prev.filter((sub) => sub.id !== id))
      });
    },
    [confirm]
  );

  return (
    <div className="md:bg-card space-y-[15px] md:min-h-[calc(100vh-300px)] md:space-y-0 md:rounded-[14px]">
      {isMobileAndSubSection && (
        <Link href={`/notification`} className="flex items-center gap-[5px] md:gap-[10px]">
          <Image
            src="/back.svg"
            alt="back"
            width={32}
            height={32}
            className="size-[32px] flex-shrink-0"
          />
          <h1 className="text-[18px] font-semibold">Subscribed Proposals</h1>
        </Link>
      )}
      <CustomTable
        columns={columns({ onRemove: handleUnsubscribe })}
        dataSource={subscriptions}
        isLoading={false}
        rowKey="id"
        className="hidden md:block"
      />

      <div className="mt-[15px] flex flex-col gap-[15px] md:hidden">
        {subscriptions.map((subscription) => (
          <Item
            key={subscription.id}
            {...subscription}
            onRemove={() => handleUnsubscribe(subscription.id)}
          />
        ))}
      </div>
    </div>
  );
}
