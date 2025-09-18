'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useCallback } from 'react';
import { toast } from 'react-toastify';

import { useIsMobileAndSubSection } from '@/app/notification/_hooks/isMobileAndSubSection';
import { CustomTable } from '@/components/custom-table';
import type { ColumnType } from '@/components/custom-table';
import { ProposalStatus } from '@/components/proposal-status';
import { Empty } from '@/components/ui/empty';
import { useConfirm } from '@/contexts/confirm-context';
import { useSubscribedProposals, useUnsubscribeProposal } from '@/hooks/useNotification';
import type { SubscribedProposalItem } from '@/lib/graphql/types';
import { extractErrorMessage } from '@/utils/graphql-error-handler';

import { Item } from './_components/item';

type ColumnProps = {
  onRemove: (daoCode: string, proposalId: string) => void;
};

const columns = ({ onRemove }: ColumnProps): ColumnType<SubscribedProposalItem>[] => [
  {
    title: 'DAO',
    key: 'daoName',
    width: '20.9%',
    className: 'text-left',
    render: (value: SubscribedProposalItem) => {
      const daoName = value.dao?.name || 'Unknown DAO';
      const daoLogo = value.dao?.logo || '/example/dao-placeholder.svg';
      const daoEndpoint = value.dao?.endpoint;

      return (
        <div className="flex items-center gap-[10px]">
          <Image
            src={daoLogo}
            alt={daoName}
            width={24}
            height={24}
            className="flex-shrink-0 rounded-full"
          />
          {daoEndpoint ? (
            <Link
              href={daoEndpoint}
              target="_blank"
              rel="noopener noreferrer"
              className="truncate transition-opacity hover:underline hover:opacity-80"
              title={daoName}
            >
              {daoName}
            </Link>
          ) : (
            <div className="truncate" title={daoName}>
              {daoName}
            </div>
          )}
        </div>
      );
    }
  },
  {
    title: 'Proposal',
    key: 'proposalTitle',
    width: '40.7%',
    className: 'text-left',
    style: { maxWidth: '0' },
    render: (value: SubscribedProposalItem) => {
      const proposalTitle = value.proposal?.title || 'Untitled Proposal';
      const proposalLink = value.proposal?.proposalLink;

      return (
        <div title={proposalTitle} className="truncate">
          {proposalLink ? (
            <Link
              href={proposalLink}
              target="_blank"
              rel="noopener noreferrer"
              className="transition-opacity hover:underline hover:opacity-80"
              title={proposalTitle}
            >
              {proposalTitle}
            </Link>
          ) : (
            <span>{proposalTitle}</span>
          )}
        </div>
      );
    }
  },

  {
    title: 'Status',
    key: 'proposalStatus',
    width: '28.1%',
    className: 'text-center',
    render: (value: SubscribedProposalItem) => {
      return (
        <div className="flex justify-center">
          <ProposalStatus status={value.proposal?.state as any} />
        </div>
      );
    }
  },
  {
    title: 'Action',
    key: 'action',
    width: '9.6%',
    className: 'text-right',
    render(value: SubscribedProposalItem) {
      return (
        <button
          className="cursor-pointer transition-opacity hover:opacity-80"
          onClick={() => onRemove(value.proposal?.daoCode || '', value.proposal?.proposalId || '')}
        >
          <Image
            src="/unsubscribed.svg"
            alt="unsubscribed"
            width={20}
            height={20}
            className="h-[20px] w-[20px] flex-shrink-0"
          />
        </button>
      );
    }
  }
];

export default function SubscribedProposalsPage() {
  const isMobileAndSubSection = useIsMobileAndSubSection();
  const { data: subscriptions, isLoading, refetch } = useSubscribedProposals();

  const unsubscribeProposalMutation = useUnsubscribeProposal();
  const { confirm } = useConfirm();

  const handleUnsubscribe = useCallback(
    (daoCode: string, proposalId: string) => {
      if (!daoCode || !proposalId) return;

      confirm({
        title: 'Unsubscribe',
        description: 'Are you sure you want to unsubscribe notifications for this proposal??',
        cancelText: 'Cancel',
        confirmText: 'Confirm',
        onConfirm: () => {
          return unsubscribeProposalMutation.mutateAsync(
            { daoCode, proposalId },
            {
              onSuccess: () => {
                refetch();
              },
              onError: (error: any) => {
                const errorMessage = extractErrorMessage(error) || 'Failed to unsubscribe proposal';
                toast.error(errorMessage);
              }
            }
          );
        }
      });
    },
    [confirm, unsubscribeProposalMutation, refetch]
  );

  return (
    <>
      {isMobileAndSubSection && (
        <Link href={`/notification`} className="flex items-center gap-[5px] lg:gap-[10px]">
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
      <CustomTable<SubscribedProposalItem>
        columns={columns({ onRemove: handleUnsubscribe })}
        dataSource={(subscriptions as SubscribedProposalItem[]) ?? []}
        isLoading={isLoading}
        rowKey={(record) => `${record.proposal?.daoCode}-${record.proposal?.proposalId}` || ''}
        emptyText="Haven't subscribed any proposals yet"
        className="hidden lg:block"
      />

      <div className="mt-[15px] flex flex-col gap-[15px] lg:hidden">
        {!isLoading && (!subscriptions || subscriptions.length === 0) ? (
          <Empty label="Haven't subscribed any proposals yet" className="mt-[100px]" />
        ) : (
          (subscriptions || []).map((subscription) => (
            <Item
              key={`${subscription.proposal?.daoCode}-${subscription.proposal?.proposalId}`}
              name={subscription.proposal?.title || 'Untitled Proposal'}
              proposalLink={subscription.proposal?.proposalLink || ''}
              createdAt={subscription.proposal?.proposalCreatedAt || ''}
              status={subscription.proposal?.state}
              onRemove={() =>
                handleUnsubscribe(
                  subscription.proposal?.daoCode || '',
                  subscription.proposal?.proposalId || ''
                )
              }
            />
          ))
        )}
      </div>
    </>
  );
}
