import { useEffect } from 'react';
import { Toast } from '@comichub/ui';
import { useUiStore } from '../store/ui.js';

const AUTO_DISMISS_MS = 6_000;

/** Bottom-right toast stack. Transient, non-blocking; each entry self-dismisses. */
export function ToastHost() {
  const toasts = useUiStore((s) => s.toasts);
  const dismiss = useUiStore((s) => s.dismissToast);

  return (
    <div
      aria-live="polite"
      style={{
        position: 'fixed',
        right: 20,
        bottom: 20,
        zIndex: 1000,
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        maxWidth: '100%',
      }}
    >
      {toasts.map((t) => (
        <AutoToast key={t.id} id={t.id} onDismiss={dismiss}>
          <Toast tone={t.tone} title={t.title} onClose={() => dismiss(t.id)}>
            {t.message}
          </Toast>
        </AutoToast>
      ))}
    </div>
  );
}

function AutoToast({
  id,
  onDismiss,
  children,
}: {
  id: string;
  onDismiss: (id: string) => void;
  children: React.ReactNode;
}) {
  useEffect(() => {
    const timer = setTimeout(() => onDismiss(id), AUTO_DISMISS_MS);
    return () => clearTimeout(timer);
  }, [id, onDismiss]);
  return <>{children}</>;
}
