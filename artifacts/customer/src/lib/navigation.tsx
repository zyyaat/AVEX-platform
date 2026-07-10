import React, { useReducer, useEffect, useMemo } from 'react';
import { useLocation, useParams as useWouterParams } from 'wouter';

const NAV_EVENT = 'avex-navigate';

// Strip trailing slash from BASE_URL so we can prepend it to paths like "/login"
const BASE = (import.meta.env.BASE_URL || '/').replace(/\/$/, '');

function toFullPath(href: string): string {
  // If it looks like an absolute path (starts with /), prepend the app base
  if (href.startsWith('/')) return `${BASE}${href}`;
  return href;
}

function dispatchNavigate() {
  window.dispatchEvent(new PopStateEvent('popstate'));
  window.dispatchEvent(new Event(NAV_EVENT));
}

/** Drop-in replacement for next/navigation useRouter */
export function useRouter() {
  return useMemo(() => ({
    push(href: string) {
      const url = new URL(toFullPath(href), window.location.origin);
      window.history.pushState({}, '', url.pathname + url.search + url.hash);
      dispatchNavigate();
    },
    replace(href: string) {
      const url = new URL(toFullPath(href), window.location.origin);
      window.history.replaceState({}, '', url.pathname + url.search + url.hash);
      dispatchNavigate();
    },
    back() { window.history.back(); },
    forward() { window.history.forward(); },
    refresh() { window.location.reload(); },
    prefetch(_href: string) { /* no-op */ },
  }), []);
}

/** Drop-in replacement for next/navigation usePathname.
 *  Returns the path RELATIVE to the app base (e.g. "/orders" not "/admin/orders"),
 *  matching Next.js App Router behaviour inside a sub-path deploy.
 */
export function usePathname() {
  const [location] = useLocation();
  return location;
}

/** Drop-in replacement for next/navigation useSearchParams.
 *  Returns the ReadonlyURLSearchParams object directly (NOT a tuple),
 *  matching Next.js 13+ App Router behaviour. Reactive on URL changes.
 */
export function useSearchParams(): URLSearchParams {
  const [, tick] = useReducer((x: number) => x + 1, 0);

  useEffect(() => {
    window.addEventListener('popstate', tick);
    window.addEventListener(NAV_EVENT, tick);
    return () => {
      window.removeEventListener('popstate', tick);
      window.removeEventListener(NAV_EVENT, tick);
    };
  }, [tick]);

  return new URLSearchParams(window.location.search);
}

/** Drop-in replacement for next/navigation useParams */
export function useParams<
  T extends Record<string, string | undefined> = Record<string, string | undefined>,
>(): T {
  return useWouterParams() as T;
}

/** Drop-in replacement for next/link Link */
export function Link({
  href,
  children,
  className,
  onClick,
  ...props
}: {
  href: string;
  children: React.ReactNode;
  className?: string;
  onClick?: React.MouseEventHandler<HTMLAnchorElement>;
  [key: string]: unknown;
}) {
  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (onClick) onClick(e);
    if (!e.defaultPrevented && !e.metaKey && !e.ctrlKey && !e.shiftKey) {
      e.preventDefault();
      const url = new URL(toFullPath(href), window.location.origin);
      window.history.pushState({}, '', url.pathname + url.search + url.hash);
      dispatchNavigate();
    }
  };
  return (
    <a
      href={toFullPath(href)}
      onClick={handleClick}
      className={className}
      {...(props as React.AnchorHTMLAttributes<HTMLAnchorElement>)}
    >
      {children}
    </a>
  );
}

/** Fallback for server-side redirect() — use useRouter().push() in client code */
export function redirect(href: string): never {
  window.location.assign(toFullPath(href));
  throw new Error('redirect');
}

/** Fallback for notFound() */
export function notFound(): never {
  throw new Error('NEXT_NOT_FOUND');
}
