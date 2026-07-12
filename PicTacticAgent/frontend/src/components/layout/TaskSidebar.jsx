import { useState, useEffect } from "react";
import {
  ChevronLeft,
  ChevronRight,
  Clock3,
  FileText,
  History,
  Loader2,
  LogOut,
  Plus,
  RefreshCw,
  Settings,
  Sparkles,
  Star,
  Trash2,
  User,
  Video,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";
import { useAuth } from "@/hooks/useAuth";
import { SettingsDialog } from "@/components/dialogs/SettingsDialog";
import { apiClient } from "@/lib/api";
import { saveUser } from "@/services/auth";

const statusClassMap = {
  completed: "bg-emerald-500/18 text-emerald-300 ring-1 ring-emerald-500/35",
  failed: "bg-destructive/12 text-destructive ring-1 ring-destructive/30",
  pending: "bg-muted text-muted-foreground ring-1 ring-border/70",
  generating: "bg-primary/18 text-primary ring-1 ring-primary/35",
  enhancing: "bg-primary/18 text-primary ring-1 ring-primary/35",
  reviewing: "bg-accent/20 text-accent-foreground ring-1 ring-accent/35",
  cancelled: "bg-muted text-muted-foreground/75 ring-1 ring-border/65",
  queued: "bg-muted text-muted-foreground ring-1 ring-border/70",
  processing: "bg-primary/18 text-primary ring-1 ring-primary/35",
};

const RUNNING_STATUSES = ["pending", "enhancing", "generating", "reviewing", "queued", "processing"];

function getStatusBadge(status, t) {
  const label = t(`sidebar.status.${status}`) || status;
  const className = statusClassMap[status] || "bg-muted text-muted-foreground ring-1 ring-border/70";
  return { label, className };
}

function timeAgo(dateStr, t) {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
  if (seconds < 60) return t("sidebar.timeAgo.justNow");
  if (seconds < 3600) return t("sidebar.timeAgo.minutes", { count: Math.floor(seconds / 60) });
  if (seconds < 86400) return t("sidebar.timeAgo.hours", { count: Math.floor(seconds / 3600) });
  if (seconds < 2592000) return t("sidebar.timeAgo.days", { count: Math.floor(seconds / 86400) });
  return date.toLocaleDateString("zh-CN");
}

export function TaskSidebar({
  tasks,
  selectedTaskId,
  loading,
  onSelect,
  onRefresh,
  onNewTask,
  onDelete,
}) {
  const [collapsed, setCollapsed] = useState(false);
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsInitialTab, setSettingsInitialTab] = useState("profile");
  const { t } = useLocale();
  const { user, logout, updateUser, isAuthenticated } = useAuth();
  const navigate = useNavigate();

  // First-use guidance: auto-open provider settings if no API keys configured
  useEffect(() => {
    if (!isAuthenticated) return;
    apiClient.getProviderSettings().then((data) => {
      const hasKey = [data.gemini?.api_key, data.jiekouai?.api_key, data.antigravity?.api_key]
        .some((k) => k && k !== "****");
      if (!hasKey) {
        setSettingsInitialTab("providers");
        setSettingsOpen(true);
      }
    }).catch(() => {});
  }, [isAuthenticated]);

  const handleLogout = async () => {
    await logout();
  };

  return (
    <div className="relative flex-shrink-0">
      <aside
        className={cn(
          "relative z-20 flex h-screen flex-col border-r border-sidebar-border/80 bg-sidebar/95 backdrop-blur-xl transition-all duration-300 ease-[cubic-bezier(0.25,0.8,0.25,1)]",
          collapsed ? "w-0 -translate-x-full opacity-0" : "w-80 translate-x-0"
        )}
      >
        <div className="border-b border-sidebar-border/75 px-4 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-accent text-primary-foreground shadow-[0_18px_42px_-24px_var(--brand-glow)]">
              <Sparkles size={16} />
            </div>
            <div>
              <p className="text-sm font-semibold tracking-tight text-sidebar-foreground">PicTactic Studio</p>
              <p className="text-xs text-muted-foreground">Creative Workspace</p>
            </div>
          </div>

          <button
            onClick={onNewTask}
            className={cn(
              "mt-4 flex w-full items-center justify-center gap-2 rounded-2xl px-4 py-3 text-sm font-semibold transition-all",
              "bg-gradient-to-r from-primary to-accent text-primary-foreground shadow-[0_18px_38px_-24px_var(--brand-glow)]",
              "hover:-translate-y-0.5"
            )}
          >
            <Plus size={16} />
            <span>{t("sidebar.newTask")}</span>
          </button>

          <button
            onClick={() => navigate("/app/templates")}
            className="mt-2 flex w-full items-center justify-center gap-2 rounded-2xl border border-border/50 bg-card/70 px-4 py-2.5 text-sm font-medium text-sidebar-foreground transition-all hover:bg-card hover:border-border"
          >
            <FileText size={15} />
            <span>{t("nav.templates") || "Templates"}</span>
          </button>
        </div>

        <div className="flex flex-1 flex-col overflow-hidden px-3 py-3">
          <div className="mb-3 flex items-center justify-between px-2">
            <div className="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
              <History size={12} />
              {t("sidebar.history")}
            </div>

            <button
              onClick={onRefresh}
              disabled={loading}
              className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground disabled:opacity-45"
              aria-label={t("sidebar.history")}
            >
              <RefreshCw className={cn("h-3.5 w-3.5", loading && "animate-spin")} />
            </button>
          </div>

          <div className="flex-1 space-y-1.5 overflow-y-auto scrollbar-hide pr-1">
            {loading && tasks.length === 0 ? (
              <div className="flex items-center justify-center py-16">
                <Loader2 className="h-4 w-4 animate-spin text-primary" />
              </div>
            ) : null}

            {!loading && tasks.length === 0 ? (
              <div className="mx-2 mt-10 rounded-2xl border border-sidebar-border/65 bg-card/60 p-6 text-center">
                <div className="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-xl bg-secondary/70 text-muted-foreground">
                  <Clock3 className="h-4 w-4" />
                </div>
                <p className="text-xs text-muted-foreground">{t("sidebar.noHistory")}</p>
              </div>
            ) : null}

            {tasks.map((task) => {
              const isSelected = task.task_id === selectedTaskId;
              const status = getStatusBadge(task.status, t);

              return (
                <div key={task.task_id} className="relative">
                  <button
                    onClick={() => onSelect(task.task_id)}
                    className={cn(
                      "group relative w-full overflow-hidden rounded-2xl border p-2.5 text-left transition-all",
                      isSelected
                        ? "border-primary/45 bg-card/90 shadow-[0_18px_48px_-36px_var(--brand-glow)]"
                        : "border-transparent bg-card/55 hover:border-sidebar-border/80 hover:bg-card/80"
                    )}
                  >
                    {isSelected ? (
                      <div className="absolute inset-y-2 left-0 w-0.5 rounded-r bg-primary" />
                    ) : null}

                    {onDelete ? (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteConfirmId(deleteConfirmId === task.task_id ? null : task.task_id);
                        }}
                        className="absolute right-2 top-2 z-10 rounded-lg p-1 text-muted-foreground opacity-0 transition-all hover:bg-destructive/15 hover:text-destructive group-hover:opacity-100"
                      >
                        <Trash2 size={12} />
                      </button>
                    ) : null}

                    <div className="flex items-start gap-2.5">
                      {task.task_type === "video" ? (
                        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-border/70 bg-secondary/60">
                          <Video className="h-4 w-4 text-muted-foreground" />
                        </div>
                      ) : task.thumbnail_url ? (
                        <div className="h-11 w-11 shrink-0 overflow-hidden rounded-xl border border-border/70 bg-secondary/60">
                          <img
                            src={task.thumbnail_url}
                            alt=""
                            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                            onError={(e) => {
                              const fallback = task.thumbnail_original_url;
                              if (fallback && !e.target.dataset.fallback) {
                                e.target.dataset.fallback = "1";
                                e.target.src = fallback;
                              }
                            }}
                          />
                        </div>
                      ) : (
                        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-border/70 bg-secondary/60">
                          <History className="h-4 w-4 text-muted-foreground" />
                        </div>
                      )}

                      <div className="min-w-0 flex-1 space-y-1">
                        <p className="line-clamp-2 text-[12px] font-medium leading-relaxed text-sidebar-foreground">
                          {task.original_prompt || "-"}
                        </p>

                        <div className="flex items-center gap-1.5">
                          <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium", status.className)}>
                            {status.label}
                          </span>

                          {RUNNING_STATUSES.includes(task.status) ? (
                            <span className="flex items-center gap-1 text-[10px] text-primary">
                              <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-primary" />
                              Live
                            </span>
                          ) : null}
                        </div>

                        <div className="flex items-center gap-2 text-[10px] text-muted-foreground">
                          {task.best_score > 0 ? (
                            <span className="inline-flex items-center gap-1 tabular-nums">
                              <Star className="h-3 w-3 fill-primary/70 text-primary/70" />
                              {(task.best_score ?? 0).toFixed(2)}
                            </span>
                          ) : null}
                          <span>{timeAgo(task.created_at, t)}</span>
                        </div>
                      </div>
                    </div>
                  </button>

                  {deleteConfirmId === task.task_id ? (
                    <div className="mt-1 flex items-center justify-between rounded-xl border border-destructive/25 bg-card/95 p-2.5 shadow-lg backdrop-blur-sm">
                      <span className="text-xs text-muted-foreground">{t("sidebar.deleteConfirm")}</span>
                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={() => setDeleteConfirmId(null)}
                          className="rounded-lg px-2.5 py-1 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-secondary"
                        >
                          {t("sidebar.cancel")}
                        </button>
                        <button
                          onClick={() => {
                            onDelete(task.task_id);
                            setDeleteConfirmId(null);
                          }}
                          className="rounded-lg bg-destructive/90 px-2.5 py-1 text-[11px] font-medium text-white transition-colors hover:bg-destructive"
                        >
                          {t("sidebar.confirm")}
                        </button>
                      </div>
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        </div>

        <div className="border-t border-sidebar-border/75 p-4">
          <div className="flex items-center justify-between rounded-2xl border border-border/65 bg-card/75 p-3">
            <div className="flex min-w-0 items-center gap-2.5">
              {user?.avatar_url ? (
                <img
                  src={user.avatar_url}
                  alt={user.display_name || user.email}
                  referrerPolicy="no-referrer"
                  className="h-9 w-9 rounded-full border border-border/60 object-cover"
                />
              ) : (
                <div className="flex h-9 w-9 items-center justify-center rounded-full border border-border/65 bg-secondary/70">
                  <User className="h-4 w-4 text-muted-foreground" />
                </div>
              )}

              <div className="min-w-0">
                <p className="truncate text-xs font-semibold text-sidebar-foreground">
                  {user?.display_name || user?.email?.split("@")[0] || t("auth.guest")}
                </p>
                <p className="truncate text-[11px] text-muted-foreground">{user?.email || t("auth.noEmail")}</p>
              </div>
            </div>

            {isAuthenticated ? (
              <div className="flex items-center gap-1">
                <button
                  onClick={() => { setSettingsInitialTab("profile"); setSettingsOpen(true); }}
                  className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground"
                  title={t("settings.title")}
                >
                  <Settings size={15} />
                </button>
                <button
                  onClick={handleLogout}
                  className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground"
                  title={t("auth.logout")}
                >
                  <LogOut size={15} />
                </button>
              </div>
            ) : null}
          </div>
        </div>
      </aside>

      <button
        onClick={() => setCollapsed(!collapsed)}
        className={cn(
          "absolute top-6 z-30 rounded-r-xl border-y border-r border-sidebar-border/80 bg-sidebar/95 p-1.5 text-muted-foreground transition-all hover:text-primary",
          collapsed ? "left-0" : "left-80"
        )}
        aria-label="toggle sidebar"
      >
        {collapsed ? <ChevronRight size={15} /> : <ChevronLeft size={15} />}
      </button>

      <SettingsDialog
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        user={user}
        initialTab={settingsInitialTab}
        onSave={(updated) => {
          updateUser(updated);
          saveUser({ ...user, ...updated });
        }}
      />
    </div>
  );
}
