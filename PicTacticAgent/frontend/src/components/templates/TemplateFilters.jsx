import { useState, useEffect, useRef } from "react";
import { Search } from "lucide-react";
import { useLocale } from "@/hooks/useLocale";
import { cn } from "@/lib/utils";

export function TemplateFilters({ search, category, categories, onSearchChange, onCategoryChange }) {
  const { t } = useLocale();
  const [localSearch, setLocalSearch] = useState(search);
  const timerRef = useRef(null);

  useEffect(() => {
    setLocalSearch(search);
  }, [search]);

  const handleInputChange = (e) => {
    const value = e.target.value;
    setLocalSearch(value);
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      onSearchChange(value);
    }, 300);
  };

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const allCategories = [null, ...(categories || [])];

  return (
    <div className="space-y-3">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          type="text"
          value={localSearch}
          onChange={handleInputChange}
          placeholder={t("templates.searchPlaceholder")}
          className="w-full rounded-xl border border-border/50 bg-card py-2.5 pl-10 pr-4 text-sm text-foreground placeholder:text-muted-foreground/60 focus:border-primary/50 focus:outline-none focus:ring-1 focus:ring-primary/30 transition-colors"
        />
      </div>

      <div className="flex flex-wrap gap-1.5">
        {allCategories.map((cat) => (
          <button
            key={cat || "all"}
            onClick={() => onCategoryChange(cat)}
            className={cn(
              "rounded-full px-3 py-1 text-xs font-medium transition-all duration-150",
              category === cat
                ? "bg-primary text-primary-foreground shadow-sm"
                : "bg-secondary/50 text-muted-foreground ring-1 ring-border/30 hover:bg-secondary hover:text-foreground"
            )}
          >
            {cat ? t(`templates.category.${cat}`, cat) : t("templates.category.all")}
          </button>
        ))}
      </div>
    </div>
  );
}
