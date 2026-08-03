'use client';

import { type ReactNode, useCallback, useEffect, useState } from 'react';
import { ChevronDown, ShieldAlert, X } from 'lucide-react';

const STORAGE_KEY_PREFIX = 'newapi-antifraud-acknowledged';

const B = ({ children }: { children: ReactNode }) => (
  <strong className="text-fd-foreground">{children}</strong>
);

const i18nContent: Record<
  string,
  {
    title: string;
    summary: string;
    sections: { heading: string; body: ReactNode }[];
    advice: { heading: string; body: ReactNode };
    showDetails: string;
    hideDetails: string;
    acknowledge: string;
    dismiss: string;
  }
> = {
  zh: {
    title: '谨防假冒 New API 官方服务',
    summary:
      'New API 从未公开销售 API 服务，也未授权任何代理或经销商。请仅信任官方开源仓库。',
    sections: [
      {
        heading: '第一：官方绝未公开发售接口，谨防受骗！',
        body: (
          <>
            New API 为纯粹的开源项目，我司
            <B>从未授权任何机构或个人开展代理或销售</B>。凡在外打着 &ldquo;New
            API 官方 / 合作方 / 自营&rdquo;旗号售卖 API 额度或中转服务的，
            <B>100% 均为假冒李鬼</B>
            ，请广大用户务必擦亮眼睛！
          </>
        ),
      },
      {
        heading: '第二：开源不等于弃权，法务已持续监控取证！',
        body: (
          <>
            针对违规闭源魔改代码、涉嫌窃取隐私与虚假宣传的黑产平台，
            <B>我司不承担其带来的任何技术与法律责任</B>
            。同时，我司已委派专业法务团队持续监控取证，针对冒用名义、诈骗等侵权行为，将视情况通过
            <B>行政举报及诉讼等法律手段</B>依法追责。
          </>
        ),
      },
    ],
    advice: {
      heading: '官方忠告',
      body: (
        <>
          强烈建议开发者前往官方 GitHub 仓库拉取源码进行
          <B>本地私有化部署</B>
          ，牢牢掌握自己的网关数据，切勿将核心业务接入不明黑产站点！
        </>
      ),
    },
    showDetails: '查看完整声明',
    hideDetails: '收起完整声明',
    acknowledge: '我知道了',
    dismiss: '关闭提示',
  },
  en: {
    title: 'Beware of counterfeit New API services',
    summary:
      'New API has never sold API access or authorized any agents or resellers. Trust only the official open-source repository.',
    sections: [
      {
        heading: 'We have NEVER publicly sold API access — beware of scams!',
        body: (
          <>
            New API is a purely open-source project. We have{' '}
            <B>
              never authorized any organization or individual to act as an agent
              or reseller
            </B>
            . Anyone selling API quotas or relay services under the name
            &ldquo;New API Official / Partner / Self-operated&rdquo; is{' '}
            <B>100% fraudulent</B>.
          </>
        ),
      },
      {
        heading:
          'Open source does not mean waiving rights — legal action is ongoing!',
        body: (
          <>
            We bear <B>no technical or legal responsibility</B> for unauthorized
            closed-source forks, privacy-stealing modifications, or fraudulent
            platforms. Our legal team is actively monitoring and collecting
            evidence, and will pursue{' '}
            <B>administrative complaints and litigation</B> against
            impersonation and fraud as appropriate.
          </>
        ),
      },
    ],
    advice: {
      heading: 'Official Recommendation',
      body: (
        <>
          We strongly recommend developers pull the source code from the
          official GitHub repository for <B>private local deployment</B>.
          Maintain full control of your gateway data and never connect critical
          services to unknown third-party platforms!
        </>
      ),
    },
    showDetails: 'Read the full statement',
    hideDetails: 'Hide the full statement',
    acknowledge: 'Got it',
    dismiss: 'Dismiss notice',
  },
  ja: {
    title: '偽の New API 公式サービスにご注意ください',
    summary:
      'New API は API を販売しておらず、代理店や再販業者も認可していません。公式のオープンソースリポジトリのみをご利用ください。',
    sections: [
      {
        heading: '公式は API を一切販売していません — 詐欺にご注意ください！',
        body: (
          <>
            New API は純粋なオープンソースプロジェクトです。
            <B>いかなる機関や個人にも代理販売を許可したことはありません</B>
            。「New API 公式 / パートナー / 自営」を名乗り API
            クォータや中継サービスを販売するものは、<B>100% 偽物です</B>。
          </>
        ),
      },
      {
        heading: 'オープンソースは権利放棄ではありません — 法的措置を継続中！',
        body: (
          <>
            無断でクローズドソース化した改変コード、プライバシー窃取、虚偽宣伝を行う不正プラットフォームについて、
            <B>当社は一切の技術的・法的責任を負いません</B>
            。法務チームが継続的に監視・証拠収集を行い、なりすましや詐欺行為に対して
            <B>行政通報および訴訟</B>を通じて法的責任を追及します。
          </>
        ),
      },
    ],
    advice: {
      heading: '公式からの推奨',
      body: (
        <>
          開発者の皆様には、公式 GitHub リポジトリからソースコードを取得し、
          <B>ローカルでプライベートデプロイ</B>
          を行うことを強くお勧めします。ゲートウェイデータを自身で管理し、不明な第三者プラットフォームには絶対に接続しないでください！
        </>
      ),
    },
    showDetails: '声明全文を表示',
    hideDetails: '声明全文を閉じる',
    acknowledge: '確認しました',
    dismiss: '通知を閉じる',
  },
};

export function AntifraudDialog({ lang }: { lang: string }) {
  const storageKey = `${STORAGE_KEY_PREFIX}-${lang}`;
  const [open, setOpen] = useState(false);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    setOpen(localStorage.getItem(storageKey) !== 'true');
    setExpanded(false);
  }, [storageKey]);

  const handleAcknowledge = useCallback(() => {
    localStorage.setItem(storageKey, 'true');
    setOpen(false);
  }, [storageKey]);

  if (!open) return null;

  const content = i18nContent[lang] || i18nContent.en;
  const detailsId = `antifraud-details-${lang}`;

  return (
    <aside
      role="status"
      aria-labelledby="antifraud-title"
      className="animate-in border-fd-border bg-fd-background/95 fade-in slide-in-from-bottom-3 fixed right-4 bottom-4 z-40 w-[calc(100vw-2rem)] max-w-sm overflow-hidden rounded-2xl border shadow-xl backdrop-blur-md duration-300"
    >
      <div className="h-1 w-full bg-gradient-to-r from-red-500 via-red-400 to-amber-400" />

      <div className="max-h-[calc(100vh-3rem)] overflow-y-auto p-4 sm:p-5">
        <div className="flex items-start gap-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-red-500/10">
            <ShieldAlert className="size-[18px] text-red-500" />
          </div>

          <div className="min-w-0 flex-1">
            <h2
              id="antifraud-title"
              className="text-fd-foreground pr-5 text-sm leading-snug font-semibold"
            >
              {content.title}
            </h2>
            <p className="text-fd-muted-foreground mt-1.5 text-[13px] leading-relaxed">
              {content.summary}
            </p>
          </div>

          <button
            type="button"
            onClick={handleAcknowledge}
            aria-label={content.dismiss}
            className="text-fd-muted-foreground hover:bg-fd-muted hover:text-fd-foreground focus-visible:ring-brand -mt-1 -mr-1 flex size-8 shrink-0 items-center justify-center rounded-lg transition-colors focus-visible:ring-2 focus-visible:outline-none"
          >
            <X className="size-4" />
          </button>
        </div>

        {expanded && (
          <div id={detailsId} className="mt-4 space-y-3">
            {content.sections.map((section) => (
              <section
                key={section.heading}
                className="border-fd-border/60 bg-fd-muted/30 rounded-lg border p-3.5"
              >
                <div className="mb-1.5 flex items-start gap-2">
                  <span className="mt-1.5 inline-block size-2 shrink-0 rounded-full bg-red-500" />
                  <h3 className="text-fd-foreground text-[13px] leading-snug font-semibold">
                    {section.heading}
                  </h3>
                </div>
                <p className="text-fd-muted-foreground text-[13px] leading-relaxed">
                  {section.body}
                </p>
              </section>
            ))}

            <section className="rounded-lg border border-amber-400/40 bg-amber-50 p-3.5 dark:bg-amber-500/5">
              <div className="mb-1.5 flex items-center gap-2">
                <span className="inline-block size-2 shrink-0 rounded-full bg-amber-500" />
                <h3 className="text-[13px] font-semibold text-amber-700 dark:text-amber-400">
                  {content.advice.heading}
                </h3>
              </div>
              <p className="text-[13px] leading-relaxed text-amber-900/70 dark:text-amber-200/70">
                {content.advice.body}
              </p>
            </section>
          </div>
        )}

        <div className="border-fd-border/70 mt-4 flex items-center justify-between gap-3 border-t pt-3">
          <button
            type="button"
            aria-expanded={expanded}
            aria-controls={detailsId}
            onClick={() => setExpanded((value) => !value)}
            className="text-fd-muted-foreground hover:text-fd-foreground focus-visible:ring-brand flex items-center gap-1.5 rounded-md px-1 py-1 text-xs font-medium transition-colors focus-visible:ring-2 focus-visible:outline-none"
          >
            {expanded ? content.hideDetails : content.showDetails}
            <ChevronDown
              className={`size-3.5 transition-transform ${expanded ? 'rotate-180' : ''}`}
            />
          </button>

          <button
            type="button"
            onClick={handleAcknowledge}
            className="bg-brand text-brand-foreground hover:bg-brand-200 focus-visible:ring-brand rounded-lg px-3.5 py-2 text-xs font-semibold transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
          >
            {content.acknowledge}
          </button>
        </div>
      </div>
    </aside>
  );
}
