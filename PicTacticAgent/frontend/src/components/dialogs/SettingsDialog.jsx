import { useState, useEffect, useRef } from "react";
import { Settings, User, Globe, Camera, X, Cpu, Eye, EyeOff, ChevronDown } from "lucide-react";
import { Select, SelectItem } from "@/components/ui/Select";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";
import { apiClient } from "@/lib/api";
import { toast } from "sonner";

const TABS = [
  { id: "profile", icon: User, labelKey: "settings.profile" },
  { id: "preferences", icon: Globe, labelKey: "settings.preferences" },
  { id: "providers", icon: Cpu, labelKey: "settings.providers" },
];

const MAX_AVATAR_SIZE = 2 * 1024 * 1024;

const PROVIDER_OPTIONS = [
  { value: "gemini", label: "Gemini" },
  { value: "jiekouai", label: "Jiekouai" },
  { value: "antigravity", label: "Antigravity" },
  { value: "gemini_web", label: "Gemini Web" },
];

function readFileAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = (e) => resolve(e.target?.result);
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

function PasswordInput({ value, onChange, placeholder }) {
  const [visible, setVisible] = useState(false);
  return (
    <div className="relative">
      <input
        type={visible ? "text" : "password"}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        className="w-full rounded-xl border border-border/70 bg-background/50 px-4 py-2.5 pr-10 text-sm text-foreground placeholder:text-muted-foreground outline-none transition-colors focus:border-primary/50 focus:ring-1 focus:ring-primary/25"
      />
      <button
        type="button"
        onClick={() => setVisible(!visible)}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
      >
        {visible ? <EyeOff size={14} /> : <Eye size={14} />}
      </button>
    </div>
  );
}

function ProviderSection({ title, open, onToggle, children }) {
  return (
    <div className="rounded-xl border border-border/50 overflow-hidden">
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium text-foreground hover:bg-secondary/30 transition-colors"
      >
        {title}
        <ChevronDown
          size={14}
          className={cn("transition-transform", open && "rotate-180")}
        />
      </button>
      {open && (
        <div className="border-t border-border/50 px-4 py-4 space-y-4">
          {children}
        </div>
      )}
    </div>
  );
}

function FieldRow({ label, children }) {
  return (
    <div className="space-y-1.5">
      <label className="text-xs font-medium text-muted-foreground">{label}</label>
      {children}
    </div>
  );
}

function TextInput({ value, onChange, placeholder }) {
  return (
    <input
      type="text"
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      className="w-full rounded-xl border border-border/70 bg-background/50 px-4 py-2.5 text-sm text-foreground placeholder:text-muted-foreground outline-none transition-colors focus:border-primary/50 focus:ring-1 focus:ring-primary/25"
    />
  );
}

export function SettingsDialog({ open, onClose, user, onSave, initialTab = "profile" }) {
  const { t, locale, setLocale } = useLocale();
  const [activeTab, setActiveTab] = useState(initialTab);
  const [displayName, setDisplayName] = useState(user?.display_name || "");
  const [avatarPreview, setAvatarPreview] = useState(user?.avatar_url || null);
  const [avatarData, setAvatarData] = useState(null);
  const [saving, setSaving] = useState(false);
  const fileInputRef = useRef(null);

  // Sync initialTab when dialog opens
  useEffect(() => {
    if (open) setActiveTab(initialTab);
  }, [open, initialTab]);

  // Provider settings state
  const [providerConfig, setProviderConfig] = useState(null);
  const [providerLoading, setProviderLoading] = useState(false);
  const [providerSaving, setProviderSaving] = useState(false);
  const [openSections, setOpenSections] = useState({ gemini: true, jiekouai: false, antigravity: false, gemini_web: false });

  // Load provider settings when switching to providers tab
  useEffect(() => {
    if (activeTab === "providers" && !providerConfig) {
      setProviderLoading(true);
      apiClient.getProviderSettings()
        .then((data) => {
          setProviderConfig(data);
        })
        .catch((err) => {
          console.error("Failed to load provider settings:", err);
          toast.error(t("settings.loadFailed"));
        })
        .finally(() => setProviderLoading(false));
    }
  }, [activeTab, providerConfig, t]);

  if (!open) return null;

  const handleAvatarChange = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > MAX_AVATAR_SIZE) {
      toast.error(t("settings.avatarTooLarge"));
      return;
    }
    const dataUrl = await readFileAsDataUrl(file);
    setAvatarPreview(dataUrl);
    setAvatarData(dataUrl);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const updates = {};
      if (displayName.trim() !== (user?.display_name || "")) {
        updates.display_name = displayName.trim();
      }
      if (avatarData) {
        updates.avatar = avatarData;
      }
      if (Object.keys(updates).length === 0) {
        onClose();
        return;
      }
      const updated = await apiClient.updateProfile(updates);
      onSave?.(updated);
      toast.success(t("settings.saveSuccess"));
      onClose();
    } catch (err) {
      toast.error(t("settings.saveFailed"));
      console.error("Profile update failed:", err);
    } finally {
      setSaving(false);
    }
  };

  const handleProviderSave = async () => {
    if (!providerConfig) return;
    setProviderSaving(true);
    try {
      const updated = await apiClient.updateProviderSettings(providerConfig);
      setProviderConfig(updated);
      toast.success(t("settings.saveSuccess"));
    } catch (err) {
      toast.error(t("settings.saveFailed"));
      console.error("Provider settings update failed:", err);
    } finally {
      setProviderSaving(false);
    }
  };

  const updateProviderField = (provider, field, value) => {
    setProviderConfig((prev) => ({
      ...prev,
      [provider]: {
        ...prev[provider],
        [field]: value,
      },
    }));
  };

  const toggleSection = (key) => {
    setOpenSections((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const handleKeyDown = (e) => {
    if (e.key === "Escape") onClose();
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      onKeyDown={handleKeyDown}
    >
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />

      <div className="relative z-10 flex h-[70vh] w-full max-w-2xl overflow-hidden rounded-2xl border border-border/50 bg-card shadow-2xl">
        {/* Sidebar */}
        <div className="flex w-48 shrink-0 flex-col border-r border-border/50 bg-secondary/30 p-4">
          <div className="mb-6 flex items-center gap-2.5">
            <Settings className="h-4 w-4 text-primary" />
            <span className="text-sm font-semibold text-foreground">{t("settings.title")}</span>
          </div>
          <nav className="flex flex-col gap-1">
            {TABS.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={cn(
                    "flex items-center gap-2.5 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
                    activeTab === tab.id
                      ? "bg-primary/15 text-primary"
                      : "text-muted-foreground hover:bg-secondary hover:text-foreground"
                  )}
                >
                  <Icon size={15} />
                  {t(tab.labelKey)}
                </button>
              );
            })}
          </nav>
        </div>

        {/* Content */}
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex items-center justify-between border-b border-border/50 px-6 py-4">
            <h3 className="text-base font-semibold text-foreground">
              {t(TABS.find((tab) => tab.id === activeTab)?.labelKey || "settings.title")}
            </h3>
            <button
              onClick={onClose}
              className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
            >
              <X size={16} />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-6">
            {/* Profile Tab */}
            {activeTab === "profile" ? (
              <div className="space-y-6">
                <div className="flex flex-col items-center gap-3">
                  <button
                    onClick={() => fileInputRef.current?.click()}
                    className="group relative h-20 w-20 overflow-hidden rounded-full border-2 border-dashed border-border/70 transition-colors hover:border-primary/50"
                  >
                    {avatarPreview ? (
                      <img src={avatarPreview} alt="" className="h-full w-full object-cover" />
                    ) : (
                      <div className="flex h-full w-full items-center justify-center bg-secondary/60">
                        <User className="h-8 w-8 text-muted-foreground" />
                      </div>
                    )}
                    <div className="absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 transition-opacity group-hover:opacity-100">
                      <Camera className="h-5 w-5 text-white" />
                    </div>
                  </button>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept="image/png,image/jpeg,image/webp,image/gif"
                    onChange={handleAvatarChange}
                    className="hidden"
                  />
                  <span className="text-xs text-muted-foreground">{t("settings.avatarUpload")}</span>
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">{t("settings.displayName")}</label>
                  <input
                    type="text"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder={t("settings.displayNamePlaceholder")}
                    maxLength={50}
                    className="w-full rounded-xl border border-border/70 bg-background/50 px-4 py-2.5 text-sm text-foreground placeholder:text-muted-foreground outline-none transition-colors focus:border-primary/50 focus:ring-1 focus:ring-primary/25"
                  />
                </div>
              </div>
            ) : null}

            {/* Preferences Tab */}
            {activeTab === "preferences" ? (
              <div className="space-y-6">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">{t("settings.language")}</label>
                  <p className="text-xs text-muted-foreground">{t("settings.languageDesc")}</p>
                  <div className="mt-3 flex gap-2">
                    {[
                      { id: "zh", label: "中文" },
                      { id: "en", label: "English" },
                    ].map((lang) => (
                      <button
                        key={lang.id}
                        onClick={() => setLocale(lang.id)}
                        className={cn(
                          "rounded-xl px-4 py-2.5 text-sm font-medium transition-colors",
                          locale === lang.id
                            ? "bg-primary/15 text-primary ring-1 ring-primary/30"
                            : "bg-secondary/50 text-muted-foreground hover:bg-secondary hover:text-foreground"
                        )}
                      >
                        {lang.label}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            ) : null}

            {/* Providers Tab */}
            {activeTab === "providers" ? (
              providerLoading ? (
                <div className="flex items-center justify-center py-12">
                  <div className="animate-spin rounded-full h-6 w-6 border-2 border-primary border-t-transparent" />
                </div>
              ) : providerConfig ? (
                <div className="space-y-5">
                  {/* Default Provider */}
                  <FieldRow label={t("settings.defaultProvider")}>
                    <Select
                      value={providerConfig.default_provider}
                      onValueChange={(val) =>
                        setProviderConfig((prev) => ({ ...prev, default_provider: val }))
                      }
                      className="rounded-xl bg-background/50 px-4 py-2.5"
                    >
                      {PROVIDER_OPTIONS.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </SelectItem>
                      ))}
                    </Select>
                  </FieldRow>

                  {/* Gemini */}
                  <ProviderSection
                    title="Gemini"
                    open={openSections.gemini}
                    onToggle={() => toggleSection("gemini")}
                  >
                    <FieldRow label={t("settings.apiKey")}>
                      <PasswordInput
                        value={providerConfig.gemini?.api_key || ""}
                        onChange={(e) => updateProviderField("gemini", "api_key", e.target.value)}
                        placeholder="AIza..."
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.imageModel")}>
                      <TextInput
                        value={providerConfig.gemini?.model || ""}
                        onChange={(e) => updateProviderField("gemini", "model", e.target.value)}
                        placeholder="gemini-3-pro-image-preview"
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.llmModel")}>
                      <TextInput
                        value={providerConfig.gemini?.llm_model || ""}
                        onChange={(e) => updateProviderField("gemini", "llm_model", e.target.value)}
                        placeholder="gemini-3-flash-preview"
                      />
                    </FieldRow>
                  </ProviderSection>

                  {/* Jiekouai */}
                  <ProviderSection
                    title="Jiekouai"
                    open={openSections.jiekouai}
                    onToggle={() => toggleSection("jiekouai")}
                  >
                    <FieldRow label={t("settings.apiKey")}>
                      <PasswordInput
                        value={providerConfig.jiekouai?.api_key || ""}
                        onChange={(e) => updateProviderField("jiekouai", "api_key", e.target.value)}
                        placeholder="sk-..."
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.apiBaseUrl")}>
                      <TextInput
                        value={providerConfig.jiekouai?.api_base_url || ""}
                        onChange={(e) => updateProviderField("jiekouai", "api_base_url", e.target.value)}
                        placeholder="https://api.jiekou.ai/v3"
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.llmBaseUrl")}>
                      <TextInput
                        value={providerConfig.jiekouai?.llm_base_url || ""}
                        onChange={(e) => updateProviderField("jiekouai", "llm_base_url", e.target.value)}
                        placeholder="https://api.jiekou.ai/openai"
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.llmModel")}>
                      <TextInput
                        value={providerConfig.jiekouai?.llm_model || ""}
                        onChange={(e) => updateProviderField("jiekouai", "llm_model", e.target.value)}
                        placeholder="gpt-4o"
                      />
                    </FieldRow>
                  </ProviderSection>

                  {/* Antigravity */}
                  <ProviderSection
                    title="Antigravity"
                    open={openSections.antigravity}
                    onToggle={() => toggleSection("antigravity")}
                  >
                    <FieldRow label={t("settings.apiKey")}>
                      <PasswordInput
                        value={providerConfig.antigravity?.api_key || ""}
                        onChange={(e) => updateProviderField("antigravity", "api_key", e.target.value)}
                        placeholder="sk-..."
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.apiBaseUrl")}>
                      <TextInput
                        value={providerConfig.antigravity?.api_base_url || ""}
                        onChange={(e) => updateProviderField("antigravity", "api_base_url", e.target.value)}
                        placeholder="http://127.0.0.1:8045/v1"
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.imageModel")}>
                      <TextInput
                        value={providerConfig.antigravity?.model || ""}
                        onChange={(e) => updateProviderField("antigravity", "model", e.target.value)}
                        placeholder="gemini-3-pro-image"
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.llmModel")}>
                      <TextInput
                        value={providerConfig.antigravity?.llm_model || ""}
                        onChange={(e) => updateProviderField("antigravity", "llm_model", e.target.value)}
                        placeholder="gemini-3-flash"
                      />
                    </FieldRow>
                  </ProviderSection>

                  {/* Gemini Web */}
                  <ProviderSection
                    title="Gemini Web"
                    open={openSections.gemini_web}
                    onToggle={() => toggleSection("gemini_web")}
                  >
                    <FieldRow label="Cookie (__Secure-1PSID)">
                      <PasswordInput
                        value={providerConfig.gemini_web?.secure_1psid || ""}
                        onChange={(e) => updateProviderField("gemini_web", "secure_1psid", e.target.value)}
                        placeholder="g.a000..."
                      />
                    </FieldRow>
                    <FieldRow label="Cookie (__Secure-1PSIDTS)">
                      <PasswordInput
                        value={providerConfig.gemini_web?.secure_1psidts || ""}
                        onChange={(e) => updateProviderField("gemini_web", "secure_1psidts", e.target.value)}
                        placeholder="sidts-..."
                      />
                    </FieldRow>
                    <FieldRow label={t("settings.imageModel")}>
                      <TextInput
                        value={providerConfig.gemini_web?.model || ""}
                        onChange={(e) => updateProviderField("gemini_web", "model", e.target.value)}
                        placeholder="gemini-3-pro"
                      />
                    </FieldRow>
                    <FieldRow label="Proxy">
                      <TextInput
                        value={providerConfig.gemini_web?.proxy || ""}
                        onChange={(e) => updateProviderField("gemini_web", "proxy", e.target.value)}
                        placeholder="http://127.0.0.1:7890"
                      />
                    </FieldRow>
                  </ProviderSection>
                </div>
              ) : null
            ) : null}
          </div>

          {/* Footer buttons */}
          {activeTab === "profile" ? (
            <div className="flex items-center justify-end gap-3 border-t border-border/50 px-6 py-4">
              <button
                onClick={onClose}
                className="rounded-xl px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
              >
                {t("sidebar.cancel")}
              </button>
              <button
                onClick={handleSave}
                disabled={saving}
                className={cn(
                  "rounded-xl px-5 py-2 text-sm font-semibold transition-all",
                  saving
                    ? "cursor-not-allowed bg-primary/50 text-primary-foreground/70"
                    : "bg-primary text-primary-foreground hover:bg-primary/90"
                )}
              >
                {saving ? t("settings.saving") : t("settings.save")}
              </button>
            </div>
          ) : null}

          {activeTab === "providers" && providerConfig ? (
            <div className="flex items-center justify-end gap-3 border-t border-border/50 px-6 py-4">
              <button
                onClick={onClose}
                className="rounded-xl px-4 py-2 text-sm font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
              >
                {t("sidebar.cancel")}
              </button>
              <button
                onClick={handleProviderSave}
                disabled={providerSaving}
                className={cn(
                  "rounded-xl px-5 py-2 text-sm font-semibold transition-all",
                  providerSaving
                    ? "cursor-not-allowed bg-primary/50 text-primary-foreground/70"
                    : "bg-primary text-primary-foreground hover:bg-primary/90"
                )}
              >
                {providerSaving ? t("settings.saving") : t("settings.save")}
              </button>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
