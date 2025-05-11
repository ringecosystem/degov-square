'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useState } from 'react';
import { useCallback } from 'react';

import { CustomTable } from '@/components/custom-table';
import type { ColumnType } from '@/components/custom-table';
import { useConfirm } from '@/contexts/confirm-context';
import { useIsMobileAndSubSection } from '@/app/notification/_hooks/isMobileAndSubSection';
import { Item } from './_components/item';
// Mock data for subscribed DAOs
const daoSubscriptions = [
  {
    id: 1,
    name: 'DAO Name 1',
    logo: '/example/dao1.svg',
    network: 'Ethereum',
    networkLogo: '/example/network1.svg'
  },
  {
    id: 2,
    name: 'DAO Name 2',
    logo: '/example/dao2.svg',
    network: 'Ethereum',
    networkLogo: '/example/network1.svg'
  },
  {
    id: 3,
    name: 'DAO Name 3',
    logo: '/example/dao3.svg',
    network: 'Ethereum',
    networkLogo: '/example/network1.svg'
  }
];

type ColumnProps = {
  onRemove: (id: number) => void;
};

const columns = ({
  onRemove
}: ColumnProps): ColumnType<{
  id: number;
  name: string;
  logo: string;
  network: string;
  networkLogo: string;
}>[] => [
  {
    title: 'Name',
    key: 'name',
    width: 375,
    render: (value) => (
      <div className="flex items-center gap-2">
        <div className="bg-background flex h-8 w-8 items-center justify-center rounded-full">
          <Image
            src={value.logo || '/example/dao-placeholder.svg'}
            alt={value.name}
            width={24}
            height={24}
          />
        </div>
        <span>{value.name}</span>
      </div>
    )
  },
  {
    title: 'Network',
    key: 'network',
    width: 375,
    className: 'text-center',
    render(value) {
      return (
        <Link
          href={`/setting/safes/${value?.id}`}
          className="flex items-center justify-center gap-[10px]"
        >
          <Image src={value?.networkLogo} alt="safe" width={16} height={17} />
          <span className="text-[16px]">{value?.network}</span>
        </Link>
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

export default function SubscribedDAOsPage() {
  const [subscriptions, setSubscriptions] = useState(daoSubscriptions);
  const { confirm } = useConfirm();
  const isMobileAndSubSection = useIsMobileAndSubSection();
  const handleUnsubscribe = useCallback(
    (id: number) => {
      confirm({
        title: 'Unsubscribe',
        description: 'Are you sure you want to unsubscribe notification?',
        cancelText: 'Cancel',
        confirmText: 'Confirm',
        onConfirm: () => {
          setSubscriptions((prev) => prev.filter((sub) => sub.id !== id));
        }
      });
    },
    [confirm]
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
