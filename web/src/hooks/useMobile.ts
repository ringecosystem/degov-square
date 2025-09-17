'use client';
import { useMedia } from 'react-use';

export const useMobile = () => {
  const isMobile = useMedia('(max-width: 1024px)', true);
  return isMobile;
};
