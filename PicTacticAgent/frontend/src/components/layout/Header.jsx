import { useState, useRef, useEffect } from "react";
import { Sun, Moon, Plus, Hexagon, Globe, Github, User, LogOut, ChevronDown, FileText } from "lucide-react";
import { useNavigate, useLocation } from "react-router-dom";
import { useLocale } from "@/hooks/useLocale";
import { useAuth } from "@/hooks/useAuth";

export function Header({ theme, onToggleTheme, showNewTask, onNewTask }) {
  const { t, locale, setLocale } = useLocale();
  const { user, logout, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [showUserMenu, setShowUserMenu] = useState(false);
  const menuRef = useRef(null);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (menuRef.current && !menuRef.current.contains(event.target)) {
        setShowUserMenu(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleLogout = async () => {
    setShowUserMenu(false);
    await logout();
  };

  return (
    <header className="sticky top-0 z-30 border-b border-border/60 bg-background/80 backdrop-blur-xl backdrop-saturate-150">
      <div className="mx-auto flex max-w-7xl items-center justify-between px-8 py-3.5">
        <div
          className="flex items-center gap-3 cursor-pointer rounded-lg px-1 -mx-1 transition-opacity hover:opacity-80"
          onClick={() => navigate("/app")}
        >
          <div className="relative flex h-9 w-9 items-center justify-center">
            <div className="absolute inset-0 rounded-xl bg-gradient-to-br from-primary/80 to-primary/40 blur-sm opacity-60" />
            <div className="relative flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/70">
              <Hexagon className="h-4 w-4 text-primary-foreground" strokeWidth={2.5} />
            </div>
          </div>
          <div>
            <h1 className="text-base font-semibold tracking-tight text-foreground font-display">
              {t("app.title")}
            </h1>
            <p className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground/70 font-medium">
              {t("app.subtitle")}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={() => setLocale(locale === "en" ? "zh" : "en")}
            className="group flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-all duration-200 hover:bg-secondary hover:text-foreground"
            title={t("lang.toggle")}
          >
            <Globe className="h-[15px] w-[15px]" />
          </button>

          <a
            href="https://github.com/NanmiCoder/PicTacticAgent"
            target="_blank"
            rel="noopener noreferrer"
            className="group flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-all duration-200 hover:bg-secondary hover:text-foreground"
            title="GitHub"
          >
            <Github className="h-[15px] w-[15px]" />
          </a>

          <button
            onClick={onToggleTheme}
            className="group flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-all duration-200 hover:bg-secondary hover:text-foreground"
          >
            {theme === "dark" ? (
              <Sun className="h-[15px] w-[15px] transition-transform duration-300 group-hover:rotate-45" />
            ) : (
              <Moon className="h-[15px] w-[15px] transition-transform duration-300 group-hover:-rotate-12" />
            )}
          </button>

          <button
            onClick={() => navigate("/app/templates")}
            className={`flex items-center gap-1.5 rounded-lg border border-border/50 px-3 py-1.5 text-xs font-medium transition-all duration-200 hover:bg-secondary hover:border-border ${
              location.pathname === "/app/templates"
                ? "bg-secondary text-foreground border-border"
                : "bg-card text-foreground"
            }`}
          >
            <FileText className="h-3.5 w-3.5" />
            {t("nav.templates")}
          </button>

          {showNewTask && (
            <button
              onClick={onNewTask}
              className="flex items-center gap-1.5 rounded-lg border border-border/50 bg-card px-3 py-1.5 text-xs font-medium text-foreground transition-all duration-200 hover:bg-secondary hover:border-border"
            >
              <Plus className="h-3.5 w-3.5" />
              {t("header.newTask")}
            </button>
          )}

          {/* User Menu */}
          {isAuthenticated && user && (
            <div className="relative ml-2" ref={menuRef}>
              <button
                onClick={() => setShowUserMenu(!showUserMenu)}
                className="flex items-center gap-2 rounded-lg border border-border/50 bg-card px-2.5 py-1.5 text-xs font-medium text-foreground transition-all duration-200 hover:bg-secondary hover:border-border"
              >
                {user.avatar_url ? (
                  <img
                    src={user.avatar_url}
                    alt={user.display_name || user.email}
                    referrerPolicy="no-referrer"
                    className="h-5 w-5 rounded-full"
                  />
                ) : (
                  <div className="flex h-5 w-5 items-center justify-center rounded-full bg-primary/20 text-primary">
                    <User className="h-3 w-3" />
                  </div>
                )}
                <span className="max-w-[100px] truncate">
                  {user.display_name || user.email.split("@")[0]}
                </span>
                <ChevronDown className={`h-3.5 w-3.5 transition-transform ${showUserMenu ? "rotate-180" : ""}`} />
              </button>

              {showUserMenu && (
                <div className="absolute right-0 top-full mt-1 w-48 rounded-lg border border-border bg-card shadow-lg z-50">
                  <div className="border-b border-border px-3 py-2">
                    <p className="text-sm font-medium text-foreground truncate">
                      {user.display_name || user.email.split("@")[0]}
                    </p>
                    <p className="text-xs text-muted-foreground truncate">{user.email}</p>
                  </div>
                  <button
                    onClick={handleLogout}
                    className="flex w-full items-center gap-2 px-3 py-2 text-sm text-foreground hover:bg-secondary transition-colors"
                  >
                    <LogOut className="h-4 w-4" />
                    {t("auth.logout")}
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
