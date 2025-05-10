'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useCallback, useState } from 'react';

import type { ColumnType } from '@/components/custom-table';
import { CustomTable } from '@/components/custom-table';
import { Button } from '@/components/ui/button';
import { useConfirm } from '@/contexts/confirm-context';

import { LinkSafeDialog } from './_components/link-safe-dialog';

import type { Safe } from './_components/link-safe-dialog';

const safeData = [
  {
    id: 1,
    name: 'Test',
    network: 'Ethereum',
    networkLogo: '/example/network1.svg',
    safeLink: 'https://safe.gnosis.io/app/eth:0x0000000000000000000000000000000000000000/balances'
  },
  {
    id: 2,
    name: 'Test',
    network: 'Ethereum',
    networkLogo: '/example/network1.svg',
    safeLink: 'https://safe.gnosis.io/app/eth:0x0000000000000000000000000000000000000000/balances'
  }
];

type ColumnProps = {
  onDelete: (id: number) => void;
};

const columns = ({ onDelete }: ColumnProps): ColumnType<any[number]>[] => [
  {
    title: 'Name',
    key: 'name',
    className: 'text-left',
    width: '33%'
  },
  {
    title: 'Network',
    key: 'network',
    className: 'text-center',
    width: '33%',
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
    className: 'text-right',
    width: '33%',
    render(value, index) {
      return (
        <div className="flex items-center justify-end gap-[10px]">
          <Link href={value?.safeLink} target="_blank" rel="noopener noreferrer">
            <Image src="/safe.svg" alt="safe" width={16} height={17} />
          </Link>
          <button
            className="cursor-pointer rounded-[100px]"
            onClick={() => {
              onDelete(value?.id);
            }}
          >
            <Image src="/delete.svg" alt="delete" width={20} height={20} />
          </button>
        </div>
      );
    }
  }
];

export default function SafesPage() {
  const { confirm } = useConfirm();
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [safes, setSafes] = useState<Safe[]>(safeData as unknown as Safe[]);
  const [isLoading, setIsLoading] = useState(false);

  const handleDelete = useCallback(
    (id: number) => {
      confirm({
        title: 'Delete Confirmation',
        description: 'Are you sure you want to delete this safe?',
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        onConfirm: () => {
          // Filter out the deleted safe
          setSafes((prev) => prev.filter((safe) => safe.id !== id));
        }
      });
    },
    [confirm]
  );

  const handleAddSafe = useCallback((safe: Safe) => {
    setIsLoading(true);
    // Simulate API call delay
    setTimeout(() => {
      setSafes((prev) => [...prev, safe]);
      setIsLoading(false);
    }, 1000);
  }, []);

  return (
    <div className="flex flex-col gap-[20px]">
      <header className="flex items-center justify-between">
        <h3 className="text-[18px] font-extrabold">Safes</h3>
        <Button
          className="gap-[5px] rounded-full px-[10px] py-[5px]"
          onClick={() => setIsDialogOpen(true)}
        >
          <Image src="/plus.svg" alt="add" width={20} height={20} />
          Link Safe
        </Button>
      </header>

      <CustomTable
        columns={columns({
          onDelete: (id) => handleDelete(id)
        })}
        dataSource={safes}
        isLoading={false}
        rowKey="id"
      />

      <LinkSafeDialog
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        onAddSafe={handleAddSafe}
        isLoading={isLoading}
      />
    </div>
  );
}
