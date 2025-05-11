'use client';
import { useTheme } from 'next-themes';
import { Button } from './ui/button';
import { Moon, Sun } from 'lucide-react';
import { useMounted } from '@/hooks/useMounted';
export function ThemeButton() {
  const { setTheme, resolvedTheme } = useTheme();

  const mounted = useMounted();
  if (!mounted) return null;
  return (
    <Button onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')} variant="outline">
      {resolvedTheme === 'dark' ? <Moon /> : <Sun />}
    </Button>
  );
}
