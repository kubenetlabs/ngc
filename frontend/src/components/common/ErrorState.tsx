import { AlertCircle } from "lucide-react";

interface ErrorStateProps {
  error: Error | null;
  onRetry?: () => void;
  message?: string;
}

export function ErrorState({ error, onRetry, message }: ErrorStateProps) {
  if (!error) return null;

  return (
    <div className="mt-6 rounded-lg border border-red-500/30 bg-red-500/5 p-6 text-center">
      <AlertCircle className="mx-auto h-8 w-8 text-red-400" />
      <p className="mt-2 text-sm text-red-400">{message || String(error)}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-3 rounded-md bg-red-600 px-3 py-1.5 text-sm text-white hover:bg-red-700"
        >
          Retry
        </button>
      )}
    </div>
  );
}
