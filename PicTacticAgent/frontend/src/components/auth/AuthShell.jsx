import { Link } from "react-router-dom";
import { ShieldCheck, Sparkles, Wand2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";

export function AuthShell({
  title,
  subtitle,
  badge,
  children,
  footer,
  className,
}) {
  const { t } = useLocale();

  const highlights = [
    {
      icon: Wand2,
      title: t("landing.feature1Title"),
      desc: t("landing.feature1Desc"),
    },
    {
      icon: ShieldCheck,
      title: t("landing.feature2Title"),
      desc: t("landing.feature2Desc"),
    },
  ];

  return (
    <div className="brand-shell relative min-h-screen overflow-hidden px-4 py-8 sm:px-6 lg:px-10">
      <div className="relative mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-7xl items-center">
        <div className="grid w-full gap-8 lg:grid-cols-[1.08fr_0.92fr]">
          <aside className="hidden lg:flex lg:flex-col lg:justify-between">
            <div>
              <Link to="/" className="inline-flex items-center gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-accent text-primary-foreground shadow-[0_18px_40px_-24px_var(--brand-glow)]">
                  <Sparkles size={19} />
                </div>
                <div>
                  <p className="text-xl font-semibold tracking-tight text-foreground">PicTactic</p>
                  <p className="text-sm text-muted-foreground">Studio</p>
                </div>
              </Link>

              <div className="mt-10 max-w-lg space-y-5">
                <span className="brand-chip">{t("landing.badge")}</span>
                <h1 className="text-balance text-4xl font-semibold leading-tight text-foreground">
                  {t("landing.heroTitle")}
                </h1>
                <p className="text-balance text-base leading-relaxed text-muted-foreground">
                  {t("landing.heroDesc")}
                </p>
              </div>
            </div>

            <div className="space-y-4 pr-6">
              {highlights.map((item) => (
                <div key={item.title} className="brand-card flex items-start gap-4 p-5">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-secondary/80 text-primary">
                    <item.icon className="h-5 w-5" />
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm font-semibold text-foreground">{item.title}</p>
                    <p className="text-sm leading-relaxed text-muted-foreground">{item.desc}</p>
                  </div>
                </div>
              ))}
            </div>
          </aside>

          <section className={cn("mx-auto flex w-full max-w-md items-center", className)}>
            <div className="brand-card-strong w-full p-6 sm:p-8">
              <Link to="/" className="inline-flex items-center gap-2.5">
                <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-accent text-primary-foreground">
                  <Sparkles size={16} />
                </div>
                <p className="text-lg font-semibold tracking-tight text-foreground">PicTactic</p>
              </Link>

              {badge ? <p className="mt-6 brand-chip">{badge}</p> : null}

              <div className="mt-5 space-y-2">
                <h2 className="text-3xl font-semibold tracking-tight text-foreground">{title}</h2>
                {subtitle ? (
                  <p className="text-sm leading-relaxed text-muted-foreground">{subtitle}</p>
                ) : null}
              </div>

              <div className="mt-7 space-y-5">{children}</div>

              {footer ? <div className="mt-6 border-t border-border/50 pt-5">{footer}</div> : null}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
