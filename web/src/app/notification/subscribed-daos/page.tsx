'use client';

import { useState } from 'react';
import Image from 'next/image';
import { CustomTable } from '@/components/custom-table';
import { ColumnType } from '@/components/custom-table';
import Link from 'next/link';
import { useCallback } from 'react';
import { useConfirm } from '@/contexts/confirm-context';
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

const columns = ({ onRemove }: ColumnProps): ColumnType<any>[] => [
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
    render(value, index) {
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

export default function SubscribedDAOsPage() {
  const [subscriptions, setSubscriptions] = useState(daoSubscriptions);
  const { confirm } = useConfirm();

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
    <div className="bg-card h-[calc(100vh-300px)] rounded-[14px]">
      <CustomTable
        columns={columns({ onRemove: handleUnsubscribe })}
        dataSource={subscriptions}
        isLoading={false}
        rowKey="id"
      />
    </div>
  );
}
