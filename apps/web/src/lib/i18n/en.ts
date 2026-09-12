export const en = {
  common: {
    brand: "RexiO Pay",
    poweredBy: "Powered by RexiO Pay",
    language: "Language",
    english: "English",
    bangla: "বাংলা",
    loading: "Loading…",
    saving: "Saving…",
    cancel: "Cancel",
    close: "Close",
    back: "Back",
    next: "Next",
    skip: "Skip for now",
    done: "Done",
    save: "Save",
    saved: "Saved",
    copy: "Copy",
    copied: "Copied",
    retry: "Try again",
    errorTitle: "Something went wrong",
    errorGeneric: "Something went wrong. Please try again.",
    errorUnauthorized: "Your session has expired. Please sign in again.",
    errorNotConnected: "You are not connected yet.",
    errorPlanLimit:
      "Your current plan has reached this limit. Upgrade your plan to continue.",
    comingSoon: "Coming soon",
    notAvailable:
      "This action needs a backend update that is not ready yet.",
    takaSymbol: "৳",
    currency: "BDT",
    all: "All",
    search: "Search",
    refresh: "Refresh",
    empty: "Nothing here yet.",
    confirm: "Confirm",
    yes: "Yes",
    no: "No",
    signOut: "Sign out",
  },
  errors: {
    plan_limit_exceeded: "Your current plan has reached this limit.",
    domain_not_whitelisted: "That domain is not in your whitelist.",
    idempotency_conflict: "That request conflicts with an earlier one.",
    device_disabled: "That device is disabled.",
    unauthorized: "You are not authorized for this.",
    forbidden: "You do not have permission for this.",
    not_found: "We could not find that.",
    user_not_found: "That person does not have an account yet.",
    invalid_request: "That request is not valid.",
    internal_error: "The server ran into a problem. Try again shortly.",
  },
  status: {
    succeeded: "Succeeded",
    pending: "Pending",
    expired: "Expired",
    canceled: "Canceled",
    active: "Active",
    disabled: "Disabled",
    offline: "Offline",
    verified: "Verified",
    review: "In review",
  },
} as const;

/** Structural shape of the dictionary with plain string leaves. */
type DeepStringify<T> = {
  [K in keyof T]: T[K] extends string
    ? string
    : T[K] extends readonly (infer U)[]
      ? U extends string
        ? string[]
        : DeepStringify<U>[]
      : DeepStringify<T[K]>;
};

export type Dict = DeepStringify<typeof en>;
