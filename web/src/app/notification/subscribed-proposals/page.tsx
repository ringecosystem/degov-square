'use client';

import { useState } from 'react';
import Image from 'next/image';
import { CustomTable } from '@/components/custom-table';
import { ColumnType } from '@/components/custom-table';
import { ProposalState, ProposalStatus } from '@/components/proposal-status';
import { useCallback } from 'react';
import { useConfirm } from '@/contexts/confirm-context';
import Link from 'next/link';
// Mock data for subscribed proposals
const proposalSubscriptions = [
  {
    id: 1,
    name: 'Proposal 1: Fund Allocation',
    daoName: 'DAO Name 1',
    daoLogo: '/example/dao1.svg',
    status: ProposalState.Pending,
    notifications: true
  },
  {
    id: 2,
    name: 'Enhancing Multichain Governance: Upgrading RARI Governance Token on ArbitrumEnhancing Multichain Governance: Upgrading RARI Governance Token on Arbitrum',
    daoName: 'DAO Name 2',
    daoLogo: '/example/dao2.svg',
    status: ProposalState.Active,
    notifications: false
  },
  {
    id: 3,
    name: 'Proposal 3: Community Initiative',
    daoName: 'DAO Name 1',
    daoLogo: '/example/dao1.svg',
    status: ProposalState.Succeeded,
    notifications: true
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
    render(value, index) {
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
  const [subscriptions, setSubscriptions] = useState(proposalSubscriptions);
  const { confirm } = useConfirm();
  const handleUnsubscribe = useCallback((id: number) => {
    confirm({
      title: 'Unsubscribe',
      description: 'Are you sure you want to unsubscribe notification?',
      cancelText: 'Cancel',
      confirmText: 'Confirm',
      onConfirm: () => setSubscriptions((prev) => prev.filter((sub) => sub.id !== id))
    });
  }, []);

  return (
    <div className="bg-card rounded-[14px]">
      <CustomTable
        columns={columns({ onRemove: handleUnsubscribe })}
        dataSource={subscriptions}
        isLoading={false}
        rowKey="id"
      />

      {subscriptions.length === 0 && (
        <div className="text-muted-foreground py-8 text-center">No proposals subscribed yet</div>
      )}
    </div>
  );
}
