'use client';

import Image from 'next/image';
import { useCallback } from 'react';

import { ColumnType } from '@/components/custom-table';
import { CustomTable } from '@/components/custom-table';
import { Button } from '@/components/ui/button';
import Link from 'next/link';
import { useConfirm } from '@/contexts/confirm-context';

const erc20Data = [
  {
    id: 1,
    name: 'USDC',
    tokenLogo: '/example/token1.svg',
    balance: '1000',
    value: '1000'
  },
  {
    id: 2,
    name: 'USDC',
    tokenLogo: '/example/token2.svg',
    balance: '1000',
    value: '1000'
  }
];

type ColumnProps = {
  assetTitle: string;
  onDelete: (id: number) => void;
};

const columns = ({ assetTitle, onDelete }: ColumnProps): ColumnType<any[number]>[] => [
  {
    title: assetTitle,
    key: 'name',
    className: 'text-left',
    width: 250,
    render(value, index) {
      return (
        <Link
          href={`/setting/treasury/${value?.id}`}
          className="flex items-center gap-[10px] hover:underline"
        >
          <Image src={value?.tokenLogo} alt="token" width={34} height={34} />
          <span className="text-[16px]">{value?.name}</span>
          <Image src="/external-link.svg" alt="arrow" width={16} height={16} />
        </Link>
      );
    }
  },
  {
    title: 'Balance',
    key: 'balance',
    className: 'text-center',
    width: 250,
    render(value, index) {
      return <span className="text-[16px]">{value?.balance}</span>;
    }
  },
  {
    title: 'Value',
    key: 'value',
    className: 'text-center',
    width: 250,
    render(value, index) {
      return <span className="text-[16px]">{value?.value}</span>;
    }
  },
  {
    title: 'Action',
    key: 'action',
    className: 'text-right',
    width: 90,
    render(value, index) {
      return (
        <button
          className="cursor-pointer rounded-[100px]"
          onClick={() => {
            onDelete(value?.id);
          }}
        >
          <Image src="/delete.svg" alt="delete" width={20} height={20} />
        </button>
      );
    }
  }
];

export default function Treasury() {
  const { confirm } = useConfirm();
  const handleDelete = useCallback(
    (type: string, id: number) => {
      console.log(type, id);
      confirm({
        title: 'Delete Confirmation',
        description: 'Are you sure you want to delete this asset from treasury?',
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        onConfirm: () => {
          console.log('confirmed');
        }
      });
    },
    [confirm]
  );
  return (
    <div className="flex flex-col gap-[20px]">
      <header className="flex items-center justify-between">
        <h3 className="text-[18px] font-extrabold">Treasury Assets</h3>
        <Button className="rounded-[100px]">
          <Image src="/plus.svg" alt="add" width={20} height={20} />
          Add Tokens
        </Button>
      </header>

      <CustomTable
        columns={columns({
          assetTitle: 'ERC-20 Assets',
          onDelete: (id) => handleDelete('ERC20', id)
        })}
        dataSource={erc20Data}
        isLoading={false}
        rowKey="id"
      />

      <CustomTable
        columns={columns({
          assetTitle: 'ERC-721 Assets',
          onDelete: (id) => handleDelete('ERC721', id)
        })}
        dataSource={erc20Data}
        isLoading={false}
        rowKey="id"
      />
    </div>
  );
}
