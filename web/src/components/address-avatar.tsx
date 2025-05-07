import { blo } from 'blo';
import Image from 'next/image';

import { cn } from '@/lib/utils';

import type { Address } from 'viem';

interface AddressAvatarProps {
  address: Address;
  size?: number;
  className?: string;
}

export const AddressAvatar = ({ address, size = 40, className }: AddressAvatarProps) => {
  const avatarUrl = blo(address as `0x${string}`);

  return (
    <Image
      src={avatarUrl}
      alt={`Avatar for ${address}`}
      width={size}
      height={size}
      className={cn('flex-shrink-0 rounded-full', className)}
      style={{
        width: size,
        height: size
      }}
    />
  );
};
