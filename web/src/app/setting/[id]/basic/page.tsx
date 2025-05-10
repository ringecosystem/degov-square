'use client';

import { useState } from 'react';
import NextImage from 'next/image';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { toast } from 'react-toastify';

import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Separator } from '@/components/ui/separator';
import { AddressAvatar } from '@/components/address-avatar';
import { useAccount } from 'wagmi';
import { InputAddon } from '@/components/ui/input-addon';

// 表单验证
const formSchema = z.object({
  name: z.string().min(1, 'DAO name is required'),
  description: z.string().max(500, 'Description cannot exceed 500 characters'),
  daoUrl: z.string().min(1, 'DAO URL is required'),
  website: z.string().url('Please enter a valid URL').or(z.literal('')),
  twitter: z.string().optional(),
  discord: z.string().optional(),
  telegram: z.string().optional(),
  github: z.string().optional(),
  email: z.string().email('Invalid email').or(z.literal(''))
});

type FormValues = z.infer<typeof formSchema>;

// 图片配置
const IMAGE_CONFIG = {
  maxWidth: 800,
  maxHeight: 800,
  quality: 0.7,
  acceptedFormats: ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'image/svg+xml'],
  maxSizeMB: 10
};

export default function BasicSettingPage() {
  const { address } = useAccount();
  const [isLoading, setIsLoading] = useState(false);
  const [avatar, setAvatar] = useState<string | null>(null);
  const [isAvatarUploading, setIsAvatarUploading] = useState(false);

  // 表单初始化
  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      description: '',
      daoUrl: '',
      website: '',
      twitter: '',
      discord: '',
      telegram: '',
      github: '',
      email: ''
    }
  });

  // 表单提交
  async function onSubmit(values: FormValues) {
    setIsLoading(true);
    try {
      console.log('Form Values:', values);
      console.log('Avatar:', avatar);

      // 这里添加API调用保存数据
      await new Promise((resolve) => setTimeout(resolve, 1000)); // 模拟API调用

      toast.success('Settings saved successfully');
    } catch (error) {
      console.error('Error saving settings:', error);
      toast.error('Failed to save settings');
    } finally {
      setIsLoading(false);
    }
  }

  // 图片压缩处理
  const compressImage = (file: File): Promise<string> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();

      reader.onload = (event) => {
        const img = new Image();
        img.onload = () => {
          let width = img.width;
          let height = img.height;

          if (width > IMAGE_CONFIG.maxWidth) {
            height = (height * IMAGE_CONFIG.maxWidth) / width;
            width = IMAGE_CONFIG.maxWidth;
          }

          if (height > IMAGE_CONFIG.maxHeight) {
            width = (width * IMAGE_CONFIG.maxHeight) / height;
            height = IMAGE_CONFIG.maxHeight;
          }

          const canvas = document.createElement('canvas');
          canvas.width = width;
          canvas.height = height;

          const ctx = canvas.getContext('2d');
          if (!ctx) {
            reject(new Error('Failed to get canvas context'));
            return;
          }

          ctx.drawImage(img, 0, 0, width, height);

          const outputFormat = file.type === 'image/png' ? 'image/png' : 'image/jpeg';
          const base64 = canvas.toDataURL(outputFormat, IMAGE_CONFIG.quality);
          resolve(base64);
        };

        img.onerror = () => {
          reject(new Error('Failed to load image'));
        };

        img.src = event.target?.result as string;
      };

      reader.onerror = () => {
        reject(new Error('Failed to read file'));
      };

      reader.readAsDataURL(file);
    });
  };

  // 头像上传处理
  const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!IMAGE_CONFIG.acceptedFormats.includes(file.type)) {
      toast.error('Please upload a JPG, PNG, GIF or WebP image.');
      return;
    }

    if (file.size > IMAGE_CONFIG.maxSizeMB * 1024 * 1024) {
      toast.error(`Maximum file size is ${IMAGE_CONFIG.maxSizeMB}MB.`);
      return;
    }

    try {
      setIsAvatarUploading(true);
      const base64 = await compressImage(file);
      setAvatar(base64);

      // 这里可以添加API调用，上传头像
      console.log('Avatar updated:', base64);
    } catch (error) {
      console.error('Error processing image:', error);
      toast.error('Image processing failed');
    } finally {
      setIsAvatarUploading(false);
    }
  };

  return (
    <div className="bg-card rounded-[14px] p-[20px]">
      <div className="flex items-start justify-center gap-[20px]">
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8 p-[20px]">
            <div className="space-y-[20px]">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">
                        DAO name
                      </FormLabel>
                      <FormControl>
                        <Input placeholder="Enter your DAO name" {...field} />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">
                        Description
                      </FormLabel>
                      <FormControl>
                        <Textarea
                          placeholder="Write a description for your DAO. This will be displayed on the DAO dashboard"
                          className="min-h-[120px] resize-none"
                          {...field}
                        />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="daoUrl"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">DAO URL</FormLabel>
                      <FormControl className="w-full">
                        <InputAddon
                          suffix=".degov.ai"
                          placeholder="DAO-name"
                          suffixClassName="bg-muted text-muted-foreground"
                          {...field}
                        />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="website"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">Website</FormLabel>
                      <FormControl>
                        <Input placeholder="The DAO site" {...field} />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="twitter"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">Twitter</FormLabel>
                      <FormControl>
                        <Input placeholder="@Username" {...field} />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="discord"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">Discord</FormLabel>
                      <FormControl>
                        <Input placeholder="Username" {...field} />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="telegram"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">
                        Telegram
                      </FormLabel>
                      <FormControl>
                        <Input placeholder="@Username" {...field} />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="github"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">Github</FormLabel>
                      <FormControl>
                        <Input placeholder="Username" {...field} />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <div className="flex items-center gap-[10px]">
                      <FormLabel className="w-[140px] flex-shrink-0 text-[14px]">Email</FormLabel>
                      <FormControl>
                        <Input type="email" placeholder="Email@example.com" {...field} />
                      </FormControl>
                    </div>
                    <div className="pl-[160px]">
                      <FormMessage />
                    </div>
                  </FormItem>
                )}
              />
            </div>
            <Separator className="my-6" />
            <div className="flex justify-center gap-[20px]">
              <Button
                type="button"
                variant="outline"
                className="w-[155px] rounded-full"
                onClick={() => form.reset()}
              >
                Cancel
              </Button>
              <Button type="submit" className="w-[155px] rounded-full" disabled={isLoading}>
                {isLoading ? 'Saving...' : 'Save'}
              </Button>
            </div>
          </form>
        </Form>

        <div className="flex flex-col items-center gap-[20px] p-[20px]">
          <div className="relative h-[110px] w-[110px] overflow-hidden rounded-full">
            {avatar ? (
              <NextImage
                src={avatar}
                alt="DAO avatar"
                width={110}
                height={110}
                className="h-full w-full object-cover"
              />
            ) : (
              address && (
                <AddressAvatar
                  address={address}
                  className="h-[110px] w-[110px] rounded-full"
                  size={110}
                />
              )
            )}

            {isAvatarUploading && (
              <div className="absolute inset-0 flex items-center justify-center bg-black/50">
                <div className="h-8 w-8 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
              </div>
            )}
          </div>

          <input
            type="file"
            id="avatar-upload"
            onChange={handleAvatarChange}
            accept={IMAGE_CONFIG.acceptedFormats.join(',')}
            className="hidden"
          />

          <Button
            type="button"
            variant="outline"
            className="w-full rounded-[14px] border"
            onClick={() => document.getElementById('avatar-upload')?.click()}
            disabled={isAvatarUploading}
          >
            {isAvatarUploading ? 'Uploading...' : 'Edit'}
          </Button>
        </div>
      </div>
    </div>
  );
}
