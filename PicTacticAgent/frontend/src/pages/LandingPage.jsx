/**
 * Landing Page - shown when user is not authenticated
 */
import { Link } from "react-router-dom";
import {
  ArrowRight,
  BarChart3,
  CheckCircle2,
  ImagePlus,
  Layers,
  Sparkles,
  Wand2,
  Workflow,
} from "lucide-react";
import { useLocale } from "@/hooks/useLocale";

export default function LandingPage() {
  const { t } = useLocale();

  const features = [
    {
      icon: Workflow,
      title: t("landing.feature1Title"),
      desc: t("landing.feature1Desc"),
      tone: "from-primary/20 to-primary/0",
    },
    {
      icon: BarChart3,
      title: t("landing.feature2Title"),
      desc: t("landing.feature2Desc"),
      tone: "from-accent/20 to-accent/0",
    },
    {
      icon: Layers,
      title: t("landing.feature3Title"),
      desc: t("landing.feature3Desc"),
      tone: "from-foreground/12 to-transparent",
    },
  ];

  const steps = [
    { icon: Wand2, title: t("landing.step1Title"), desc: t("landing.step1Desc") },
    { icon: ImagePlus, title: t("landing.step2Title"), desc: t("landing.step2Desc") },
    { icon: BarChart3, title: t("landing.step3Title"), desc: t("landing.step3Desc") },
    { icon: CheckCircle2, title: t("landing.step4Title"), desc: t("landing.step4Desc") },
  ];

  return (
    <div className="brand-shell min-h-screen overflow-hidden text-foreground">
      <header className="sticky top-0 z-40 border-b border-border/50 bg-background/70 backdrop-blur-xl">
        <div className="mx-auto flex w-full max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-10">
          <Link to="/" className="inline-flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-accent text-primary-foreground shadow-[0_16px_36px_-20px_var(--brand-glow)]">
              <Sparkles size={16} />
            </div>
            <span className="text-lg font-semibold tracking-tight">PicTactic Studio</span>
          </Link>

          <div className="flex items-center gap-2 sm:gap-3">
            <Link to="/auth/login" className="brand-btn-secondary px-4 py-2.5 text-sm">
              {t("auth.login")}
            </Link>
            <Link to="/auth/register" className="brand-btn-primary px-4 py-2.5 text-sm">
              {t("landing.ctaPrimary")}
            </Link>
          </div>
        </div>
      </header>

      <main>
        <section className="mx-auto grid w-full max-w-7xl gap-10 px-4 pb-16 pt-14 sm:px-6 lg:grid-cols-[1.02fr_0.98fr] lg:px-10 lg:pb-24 lg:pt-20">
          <div className="space-y-7">
            <span className="brand-chip">{t("landing.badge")}</span>

            <div className="space-y-4">
              <h1 className="text-balance text-4xl font-semibold leading-tight sm:text-5xl lg:text-6xl">
                {t("landing.heroTitle")}
              </h1>
              <p className="text-balance max-w-xl text-base leading-relaxed text-muted-foreground sm:text-lg">
                {t("landing.heroDesc")}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <Link to="/auth/register" className="brand-btn-primary px-6 py-3.5 text-base">
                {t("landing.startFree")}
                <ArrowRight size={16} />
              </Link>
              <Link to="/auth/login" className="brand-btn-secondary px-6 py-3.5 text-base">
                {t("landing.ctaSecondary")}
              </Link>
            </div>

            <div className="grid max-w-xl grid-cols-3 gap-3 pt-3">
              {[
                { label: "6D", desc: "Scoring" },
                { label: "5x", desc: "Variants" },
                { label: "1-Click", desc: "Provider switch" },
              ].map((item) => (
                <div key={item.label} className="brand-card p-4 text-center">
                  <p className="text-lg font-semibold tracking-tight text-foreground">{item.label}</p>
                  <p className="text-xs text-muted-foreground">{item.desc}</p>
                </div>
              ))}
            </div>
          </div>

          <div className="flex items-center">
            <div className="brand-card-strong w-full p-5 sm:p-6">
              <div className="flex items-center justify-between border-b border-border/50 pb-4">
                <p className="text-sm font-semibold text-foreground">Creative Session</p>
                <span className="brand-chip">Live Loop</span>
              </div>

              <div className="mt-4 space-y-3">
                {[
                  { role: "Prompt", text: t("landing.preview.step1"), done: true },
                  { role: "Reference", text: t("landing.preview.step2"), done: true },
                  { role: "Pipeline", text: t("landing.preview.step3"), done: true },
                  { role: "Result", text: t("landing.preview.step4"), done: false },
                ].map((item) => (
                  <div
                    key={item.role}
                    className="flex items-start gap-3 rounded-2xl border border-border/60 bg-card/75 px-4 py-3"
                  >
                    <div
                      className={`mt-0.5 flex h-6 w-6 items-center justify-center rounded-full ${
                        item.done ? "bg-primary/20 text-primary" : "bg-secondary text-muted-foreground"
                      }`}
                    >
                      <CheckCircle2 className="h-3.5 w-3.5" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-xs font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                        {item.role}
                      </p>
                      <p className="text-sm text-foreground/90">{item.text}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className="mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 lg:px-10 lg:py-12">
          <div className="mb-8 flex flex-col gap-3 sm:mb-10 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
                {t("landing.featuresTitle")}
              </h2>
              <p className="mt-2 max-w-2xl text-sm text-muted-foreground sm:text-base">
                {t("landing.featuresSubtitle")}
              </p>
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            {features.map((feature) => (
              <article
                key={feature.title}
                className={`brand-card group relative overflow-hidden p-6 transition-transform duration-300 hover:-translate-y-1`}
              >
                <div
                  className={`pointer-events-none absolute inset-0 bg-gradient-to-br ${feature.tone} opacity-50 transition-opacity duration-300 group-hover:opacity-80`}
                />
                <div className="relative">
                  <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary/80 text-primary">
                    <feature.icon className="h-5 w-5" />
                  </div>
                  <h3 className="text-lg font-semibold text-foreground">{feature.title}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{feature.desc}</p>
                </div>
              </article>
            ))}
          </div>
        </section>

        <section className="mx-auto w-full max-w-7xl px-4 py-10 sm:px-6 lg:px-10 lg:py-14">
          <div className="brand-card p-6 sm:p-8">
            <div className="mb-8 space-y-2">
              <h2 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
                {t("landing.howTitle")}
              </h2>
              <p className="text-sm text-muted-foreground sm:text-base">{t("landing.howSubtitle")}</p>
            </div>

            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              {steps.map((step, index) => (
                <div key={step.title} className="rounded-2xl border border-border/60 bg-card/70 p-4">
                  <div className="mb-3 flex items-center justify-between">
                    <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/15 text-primary">
                      <step.icon className="h-4 w-4" />
                    </div>
                    <span className="text-xs font-semibold text-muted-foreground">0{index + 1}</span>
                  </div>
                  <p className="text-sm font-semibold text-foreground">{step.title}</p>
                  <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{step.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="mx-auto w-full max-w-7xl px-4 pb-20 pt-8 text-center sm:px-6 lg:px-10 lg:pb-24">
          <div className="brand-card-strong p-8 sm:p-12">
            <h2 className="text-balance text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
              {t("landing.ctaTitle")}
            </h2>
            <p className="mx-auto mt-3 max-w-2xl text-sm text-muted-foreground sm:text-base">
              {t("landing.ctaSubtitle")}
            </p>
            <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
              <Link to="/auth/register" className="brand-btn-primary px-6 py-3.5 text-base">
                {t("landing.ctaButton")}
                <ArrowRight size={16} />
              </Link>
              <Link to="/auth/login" className="brand-btn-secondary px-6 py-3.5 text-base">
                {t("auth.login")}
              </Link>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-border/55 bg-background/60 py-6 text-center text-xs text-muted-foreground">
        PicTactic Studio · AI-Powered Cover Image Pipeline
      </footer>
    </div>
  );
}
