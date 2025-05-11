'use client';
import { useMedia } from 'react-use';

export const useMobile = () => {
  const isMobile = useMedia('(max-width: 768px)', true);
  return isMobile;
};
