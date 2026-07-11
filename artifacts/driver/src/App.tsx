import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/toaster';
import { Toaster as SonnerToaster } from '@/components/ui/sonner';
import { Route, Switch, Router as WouterRouter } from 'wouter';
import NotFound from '@/pages/not-found';

import LoginPage from '@/app/login/page';
import { DriverLayout } from '@/components/DriverLayout';
import HomePage from '@/app/page';
import EarningsPage from '@/app/earnings/page';
import HistoryPage from '@/app/history/page';
import ProfilePage from '@/app/profile/page';
import SupportPage from '@/app/support/page';
import ChangePasswordPage from '@/app/change-password/page';

const queryClient = new QueryClient();

function DriverRoutes() {
  return (
    <DriverLayout>
      <Switch>
        <Route path="/" component={HomePage} />
        <Route path="/earnings" component={EarningsPage} />
        <Route path="/history" component={HistoryPage} />
        <Route path="/profile" component={ProfilePage} />
        <Route path="/support" component={SupportPage} />
        <Route path="/change-password" component={ChangePasswordPage} />
        <Route component={NotFound} />
      </Switch>
    </DriverLayout>
  );
}

function Router() {
  return (
    <Switch>
      <Route path="/login" component={LoginPage} />
      <Route path="/" component={DriverRoutes} />
      <Route path="/:rest*" component={DriverRoutes} />
    </Switch>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <WouterRouter base={import.meta.env.BASE_URL.replace(/\/$/, '')}>
        <Router />
      </WouterRouter>
      <SonnerToaster />
      <Toaster />
    </QueryClientProvider>
  );
}
