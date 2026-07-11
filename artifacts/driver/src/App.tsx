import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/toaster';
import { Toaster as SonnerToaster } from '@/components/ui/sonner';
import DriverPage from '@/app/page';

const queryClient = new QueryClient();

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <DriverPage />
      <SonnerToaster position="top-center" />
      <Toaster />
    </QueryClientProvider>
  );
}
