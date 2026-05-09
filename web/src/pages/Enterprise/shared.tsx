import { cn } from '@/lib/utils';
import type { ReactNode, SelectHTMLAttributes } from 'react';

export function EnterpriseHero({
  eyebrow,
  title,
  description,
  actions,
}: {
  eyebrow: string;
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <section className="relative overflow-hidden rounded-[28px] border border-emerald-500/15 bg-[radial-gradient(circle_at_top_left,rgba(16,185,129,0.16),transparent_34%),linear-gradient(145deg,rgba(255,255,255,0.95),rgba(241,245,249,0.86))] p-7 shadow-[0_24px_80px_-36px_rgba(16,185,129,0.45)] dark:border-emerald-400/15 dark:bg-[radial-gradient(circle_at_top_left,rgba(16,185,129,0.18),transparent_34%),linear-gradient(145deg,rgba(7,12,16,0.96),rgba(8,15,18,0.84))]">
      <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <div className="max-w-3xl space-y-4">
          <span className="inline-flex items-center rounded-full border border-emerald-500/20 bg-emerald-500/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.26em] text-emerald-700 dark:text-emerald-300">
            {eyebrow}
          </span>
          <div className="space-y-3">
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 dark:text-white md:text-4xl">
              {title}
            </h1>
            <p className="max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300 md:text-[15px]">
              {description}
            </p>
          </div>
        </div>
        {actions && <div className="flex flex-wrap items-center gap-3">{actions}</div>}
      </div>
    </section>
  );
}

export function EnterprisePanel({
  title,
  description,
  action,
  children,
  className,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={cn('rounded-[24px] border border-slate-200/80 bg-white/88 p-5 shadow-[0_20px_60px_-38px_rgba(15,23,42,0.22)] backdrop-blur-xl dark:border-white/[0.08] dark:bg-[rgba(8,13,18,0.82)]', className)}>
      <div className="mb-5 flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div className="space-y-1.5">
          <h2 className="text-lg font-semibold tracking-tight text-slate-900 dark:text-white">{title}</h2>
          {description && <p className="text-sm text-slate-500 dark:text-slate-400">{description}</p>}
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

export function DataPill({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="rounded-2xl border border-slate-200/80 bg-slate-50/90 px-4 py-3 dark:border-white/[0.07] dark:bg-white/[0.03]">
      <p className="text-[11px] uppercase tracking-[0.22em] text-slate-400">{label}</p>
      <p className="mt-2 text-xl font-semibold text-slate-950 dark:text-white">{value}</p>
    </div>
  );
}

export function Select({ className, children, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      className={cn(
        'w-full rounded-lg border border-gray-300/90 bg-white/90 px-3 py-2 text-sm text-slate-900 transition-all focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/25 dark:border-white/[0.1] dark:bg-[rgba(0,0,0,0.45)] dark:text-white',
        className
      )}
      {...props}
    >
      {children}
    </select>
  );
}

export function TinyTable({ children }: { children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-2xl border border-slate-200/80 dark:border-white/[0.06]">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200/80 text-sm dark:divide-white/[0.06]">
          {children}
        </table>
      </div>
    </div>
  );
}
