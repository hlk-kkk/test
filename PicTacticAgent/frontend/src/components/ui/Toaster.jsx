import { Toaster as SonnerToaster } from "sonner";

export function Toaster() {
  return (
    <SonnerToaster
      position="top-right"
      richColors
      toastOptions={{
        className:
          "!rounded-lg !border !border-border/50 !bg-card !text-foreground !shadow-lg",
      }}
    />
  );
}
