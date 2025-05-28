'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useCallback, useState } from 'react';

import { useIsMobileAndSubSection } from '@/app/setting/_hooks/isMobileAndSubSection';
import type { ColumnType } from '@/components/custom-table';
import { CustomTable } from '@/components/custom-table';
import { Button } from '@/components/ui/button';
import { useConfirm } from '@/contexts/confirm-context';

import { AddTokensDialog } from './_components/add-tokens-dialog';
import { AssetList } from './_components/assetList';

import type { Token } from './_components/add-tokens-dialog';
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
  const { id } = useParams();
  const [isAddTokensOpen, setIsAddTokensOpen] = useState(false);
  const [erc20Tokens, setErc20Tokens] = useState(erc20Data);
  const [erc721Tokens, setErc721Tokens] = useState<typeof erc20Data>([]);
  const [isLoading, setIsLoading] = useState(false);
  const isMobileAndSubSection = useIsMobileAndSubSection();

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
          // Handle token deletion here
          if (type === 'ERC20') {
            setErc20Tokens((prev) => prev.filter((token) => token.id !== id));
          } else if (type === 'ERC721') {
            setErc721Tokens((prev) => prev.filter((token) => token.id !== id));
          }
        }
      });
    },
    [confirm]
  );

  const handleAddToken = useCallback((token: Token) => {
    setIsLoading(true);

    // Simulate API call
    setTimeout(() => {
      const newToken = {
        id: Date.now(), // Use timestamp as temp ID
        name: token.name,
        tokenLogo: token.tokenLogo,
        balance: '0',
        value: '0'
      };

      if (token.type === 'ERC20') {
        setErc20Tokens((prev) => [...prev, newToken]);
      } else {
        setErc721Tokens((prev) => [...prev, newToken]);
      }

      setIsLoading(false);
    }, 1000);
  }, []);

  return (
    <div className="flex flex-col gap-[20px]">
      {!isMobileAndSubSection ? (
        <header className="flex items-center justify-between">
          <h3 className="text-[18px] font-extrabold">Treasury Assets</h3>
          <Button
            className="gap-[5px] rounded-full px-[10px] py-[5px]"
            onClick={() => setIsAddTokensOpen(true)}
          >
            <Image src="/plus.svg" alt="add" width={20} height={20} />
            Add Tokens
          </Button>
        </header>
      ) : (
        <Link href={`/setting/${id}`} className="flex items-center gap-[5px] md:gap-[10px]">
          <Image
            src="/back.svg"
            alt="back"
            width={32}
            height={32}
            className="size-[32px] flex-shrink-0"
          />
          <h1 className="text-[18px] font-semibold">Treasury Assets</h1>
        </Link>
      )}

      <CustomTable
        columns={columns({
          assetTitle: 'ERC-20 Assets',
          onDelete: (id) => handleDelete('ERC20', id)
        })}
        dataSource={erc20Tokens}
        isLoading={false}
        rowKey="id"
        className="hidden md:block"
      />

      <CustomTable
        columns={columns({
          assetTitle: 'ERC-721 Assets',
          onDelete: (id) => handleDelete('ERC721', id)
        })}
        dataSource={erc721Tokens}
        isLoading={false}
        rowKey="id"
        className="hidden md:block"
      />

      <div className="flex flex-col gap-[10px] md:hidden">
        <h2 className="text-[12px] font-semibold">ERC-20 Assets</h2>
        <AssetList
          title="ERC-20 Assets"
          assets={erc20Tokens}
          isLoading={isLoading}
          onDelete={(id) => handleDelete('ERC20', id)}
        />
      </div>
      <div className="flex flex-col gap-[10px] md:hidden">
        <h2 className="text-[12px] font-semibold">ERC-721 Assets</h2>
        <AssetList
          title="ERC-721 Assets"
          assets={erc721Tokens}
          isLoading={isLoading}
          onDelete={(id) => handleDelete('ERC721', id)}
        />
      </div>

      <Button
        className="fixed bottom-[20px] w-[calc(100%-40px)] gap-[5px] rounded-full px-[10px] py-[5px] md:hidden"
        onClick={() => setIsAddTokensOpen(true)}
      >
        <Image src="/plus.svg" alt="add" width={20} height={20} />
        Add Tokens
      </Button>

      <AddTokensDialog
        open={isAddTokensOpen}
        onOpenChange={setIsAddTokensOpen}
        onAddToken={handleAddToken}
        isLoading={isLoading}
      />
    </div>
  );
}
