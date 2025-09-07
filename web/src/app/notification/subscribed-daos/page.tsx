'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useCallback } from 'react';

import { useIsMobileAndSubSection } from '@/app/notification/_hooks/isMobileAndSubSection';
import { CustomTable } from '@/components/custom-table';
import type { ColumnType } from '@/components/custom-table';
import { useConfirm } from '@/contexts/confirm-context';
import { useSubscribedDaos, useUnsubscribeDao } from '@/lib/graphql/hooks';
import type { SubscribedDaoItem } from '@/lib/graphql/types';

import { Item } from './_components/item';

type ColumnProps = {
  onRemove: (daoCode: string) => void;
};

const columns = ({
  onRemove
}: ColumnProps): ColumnType<SubscribedDaoItem>[] => [
  {
    title: 'Name',
    key: 'dao',
    width: 375,
    render: (value) => (
      <div className="flex items-center gap-2">
        <div className="bg-background flex h-8 w-8 items-center justify-center rounded-full">
          <Image
            src={value.dao?.logo || '/example/dao-placeholder.svg'}
            alt={value.dao?.name || 'DAO'}
            width={24}
            height={24}
          />
        </div>
        <span>{value.dao?.name || 'Unknown DAO'}</span>
      </div>
    )
  },
  {
    title: 'Network',
    key: 'dao',
    width: 375,
    className: 'text-center',
    render(value) {
      return (
        <div className="flex items-center justify-center gap-[10px]">
          <Image 
            src={value.dao?.chainLogo || '/example/network-placeholder.svg'} 
            alt={value.dao?.chainName || 'Network'} 
            width={16} 
            height={17} 
          />
          <span className="text-[16px]">{value.dao?.chainName || 'Unknown Network'}</span>
        </div>
      );
    }
  },
  {
    title: 'Action',
    key: 'action',
    width: 60,
    className: 'text-right',
    render(value) {
      return (
        <button
          className="cursor-pointer transition-opacity hover:opacity-80"
          onClick={() => onRemove(value.dao?.code || '')}
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

export default function SubscribedDAOsPage() {
  const { data: subscriptions, isLoading, refetch } = useSubscribedDaos();
  const unsubscribeMutation = useUnsubscribeDao();
  const { confirm } = useConfirm();
  const isMobileAndSubSection = useIsMobileAndSubSection();
  
  const handleUnsubscribe = useCallback(
    (daoCode: string) => {
      if (!daoCode) return;
      
      confirm({
        title: 'Unsubscribe',
        description: 'Are you sure you want to unsubscribe from this DAO?',
        cancelText: 'Cancel',
        confirmText: 'Confirm',
        onConfirm: () => {
          unsubscribeMutation.mutate(
            { daoCode },
            {
              onSuccess: () => {
                refetch();
              }
            }
          );
        }
      });
    },
    [confirm, unsubscribeMutation, refetch]
  );

  return (
    <div className="md:bg-card md:h-[calc(100vh-300px)] md:rounded-[14px]">
      {isMobileAndSubSection && (
        <Link href={`/notification`} className="flex items-center gap-[5px] md:gap-[10px]">
          <Image
            src="/back.svg"
            alt="back"
            width={32}
            height={32}
            className="size-[32px] flex-shrink-0"
          />
          <h1 className="text-[18px] font-semibold">Subscribed DAOs</h1>
        </Link>
      )}
      <CustomTable
        columns={columns({ onRemove: handleUnsubscribe })}
        dataSource={subscriptions || []}
        isLoading={isLoading}
        rowKey={(record) => record.dao?.code || ''}
        className="hidden md:block"
      />

      <div className="mt-[15px] flex flex-col gap-[15px] md:hidden">
        {(subscriptions || []).map((subscription) => (
          <Item
            key={subscription.dao?.code}
            id={subscription.dao?.code || ''}
            name={subscription.dao?.name || 'Unknown DAO'}
            logo={subscription.dao?.logo || '/example/dao-placeholder.svg'}
            network={subscription.dao?.chainName || 'Unknown Network'}
            networkLogo={subscription.dao?.chainLogo || '/example/network-placeholder.svg'}
            onRemove={() => handleUnsubscribe(subscription.dao?.code || '')}
          />
        ))}
      </div>
    </div>
  );
}
