'use client';

import Link from 'next/link';

export function ComplianceNotice({ lang }: { lang: string }) {
  if (lang !== 'zh') return null;

  return (
    <div className="bg-amber-50 dark:bg-amber-950/30 border-b border-amber-200 dark:border-amber-800 px-4 py-2 text-sm text-amber-800 dark:text-amber-200">
      <div className="mx-auto max-w-[var(--fd-layout-width)] flex items-center gap-2">
        <span className="shrink-0">⚠️</span>
        <span>
          合规提示：本项目仅用于合法授权的 API
          网关、内部管理和私有化部署场景。请遵守上游服务条款、平台规则、监管要求和内容安全要求。
        </span>
      </div>
    </div>
  );
}
