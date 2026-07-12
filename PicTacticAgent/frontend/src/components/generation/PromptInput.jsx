import { useState, useEffect } from "react";
import { ArrowUp, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useLocale } from "@/hooks/useLocale";

export function PromptInput({ onSubmit, disabled = false, isGenerating = false, placeholder, initialValue = "" }) {
  const [prompt, setPrompt] = useState(initialValue);

  useEffect(() => {
    if (initialValue) setPrompt(initialValue);
  }, [initialValue]);
  const { t } = useLocale();

  const handleSubmit = (e) => {
    e.preventDefault();
    if (prompt.trim() && !disabled) {
      onSubmit(prompt.trim());
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="w-full">
      <div className="group relative rounded-2xl bg-card ring-1 ring-border/50 transition-all duration-200 focus-within:ring-primary/40 focus-within:ring-2">
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled}
          rows={4}
          className="w-full resize-none bg-transparent px-4 pt-4 pb-14 text-sm text-foreground placeholder-muted-foreground/50 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50 leading-relaxed"
        />
        <div className="absolute bottom-3 left-3 right-3 flex items-center justify-between">
          <span className="text-[10px] text-muted-foreground/40">
            {prompt.length > 0 && t("prompt.chars", { count: prompt.length })}
          </span>
          <button
            type="submit"
            disabled={disabled || !prompt.trim()}
            className={cn(
              "flex h-8 w-8 items-center justify-center rounded-lg transition-all duration-200",
              prompt.trim() && !disabled
                ? "bg-primary text-primary-foreground shadow-sm shadow-primary/20 hover:shadow-primary/30 hover:brightness-110"
                : "bg-secondary text-muted-foreground/40 cursor-not-allowed"
            )}
          >
            {isGenerating ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <ArrowUp className="h-4 w-4" />
            )}
          </button>
        </div>
      </div>
    </form>
  );
}
