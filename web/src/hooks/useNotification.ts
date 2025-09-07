// Re-export notification-related hooks from GraphQL hooks
export {
  useBindNotificationChannel,
  useResendOTP,
  useVerifyNotificationChannel,
  useListNotificationChannels,
  useSubscribedDaos
} from '@/lib/graphql/hooks';