'use client';
import Image from 'next/image';
import Link from 'next/link';
import { useState, useEffect, useCallback } from 'react';

import type { ColumnType } from '@/components/custom-table';
import { CustomTable } from '@/components/custom-table';
import { SortableCell } from '@/components/sortable-cell';
import { Button } from '@/components/ui/button';
import { daoInfo } from '@/data/daoInfo';

import { DaoList } from './_components/daoList';
import { MobileSearchDialog } from './_components/MobileSearchDialog';
type SortState = 'asc' | 'desc';

export default function Home() {
  const [isFetchingNextPage, setIsFetchingNextPage] = useState(false);
  const [sortState, setSortState] = useState<SortState | undefined>(undefined);
  const [searchQuery, setSearchQuery] = useState('');
  const [openSearchDialog, setOpenSearchDialog] = useState(false);

  const columns: ColumnType<(typeof daoInfo)[number]>[] = [
    {
      title: 'Name',
      key: 'name',
      className: 'w-[34%] text-left',
      render(value) {
        return (
          <div className="flex items-center gap-[10px]">
            <Image src={value?.daoIcon} alt="dao" width={34} height={34} />
            <span className="text-[16px]">{value?.name}</span>
          </div>
        );
      }
    },
    {
      title: 'Network',
      key: 'network',
      className: 'w-[28%] text-center',
      render(value) {
        return (
          <div className="flex items-center justify-center gap-[10px]">
            <Image src={value?.networkIcon} alt="network" width={24} height={24} />
            <span className="text-[16px]">{value?.network}</span>
          </div>
        );
      }
    },
    {
      title: <SortableCell onClick={setSortState} sortState={sortState} />,
      key: 'proposals',
      className: 'w-[28%] text-center',
      render(value) {
        return <span className="text-[16px]">{value?.proposals}</span>;
      }
    },
    {
      title: 'Action',
      key: 'action',
      className: 'w-[20%] text-right',
      render(value) {
        return (
          <div className="flex items-center justify-end gap-[10px]">
            <Link
              href={`/setting/${value?.id}`}
              className="cursor-pointer transition-opacity hover:opacity-80"
            >
              <Image src="/setting.svg" alt="setting" width={20} height={20} />
            </Link>
            <button className="cursor-pointer transition-opacity hover:opacity-80">
              <Image src="/favorite.svg" alt="favorite" width={20} height={20} />
            </button>
          </div>
        );
      }
    }
  ];

  const handleSearch = useCallback((query: string) => {
    setSearchQuery(query);
    setOpenSearchDialog(false);
  }, []);

  useEffect(() => {
    return () => {
      setSortState(undefined);
    };
  }, []);

  return (
    <div className="container flex flex-col gap-[20px]">
      <div className="flex items-center justify-between">
        <span className="text-[18px] font-semibold">All DAOs(5)</span>
        <div className="flex items-center gap-[20px]">
          <div className="bg-card flex h-[36px] w-[109px] items-center gap-[13px] rounded-[19px] border px-[17px] py-[9px] md:h-auto md:w-[388px] md:gap-[10px]">
            <Image src="/search.svg" alt="search" width={16} height={16} />
            <input
              className="placeholder:text-muted-foreground hidden h-[17px] outline-none placeholder:text-[14px] md:block"
              placeholder="Search by Name, Chain"
            />
            <span className="text-muted-foreground block text-[14px] md:hidden">Search</span>
          </div>
          <div className="fixed right-0 bottom-[20px] left-[30px] grid grid-cols-[calc(50%-20px)_calc(50%-20px)] gap-[20px] md:static md:grid-cols-2 md:justify-end">
            <Button variant="outline" className="rounded-[100px]" asChild>
              <Link href="/add/existing">Add Existing DAO</Link>
            </Button>
            <Button variant="outline" className="rounded-[100px]">
              With Assistance
            </Button>
          </div>
        </div>
      </div>
      <CustomTable
        columns={columns}
        dataSource={daoInfo}
        className="hidden md:block"
        rowKey="name"
        caption={
          <div className="text-foreground hover:text-foreground/80 cursor-pointer transition-colors">
            {isFetchingNextPage ? 'Loading more...' : 'View more'}
          </div>
        }
      />
      <DaoList daoInfo={daoInfo} isLoading={false} />
      <MobileSearchDialog
        open={openSearchDialog}
        onOpenChange={setOpenSearchDialog}
        onConfirm={handleSearch}
      />
    </div>
  );
}
