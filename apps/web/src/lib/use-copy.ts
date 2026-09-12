"use client";

import { useCallback, useEffect, useRef, useState } from "react";

/** Copy-to-clipboard with a short "copied" flash. */
export function useCopy(timeoutMs = 1500): {
  copied: string | null;
  copy: (value: string, tag?: string) => void;
} {
  const [copied, setCopied] = useState<string | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  const copy = useCallback(
    (value: string, tag = value) => {
      void navigator.clipboard.writeText(value).then(() => {
        setCopied(tag);
        if (timer.current) clearTimeout(timer.current);
        timer.current = setTimeout(() => setCopied(null), timeoutMs);
      });
    },
    [timeoutMs],
  );

  return { copied, copy };
}
