import { minLength, withMessage } from "@regle/rules";
import type { MaybeInput } from "@regle/core";

// Empty values pass; `required` is what enforces presence.
const optional =
  (test: (value: string) => boolean) => (value: MaybeInput<string>) =>
    !value || test(value);

export const passwordMinLength = minLength(6);

export const noSpaces = withMessage(
  optional((value) => !/\s/.test(value)),
  "Password cannot contain spaces",
);

export const hasNumber = withMessage(
  optional((value) => /\d/.test(value)),
  "Password must contain at least one number",
);

export const hasUppercase = withMessage(
  optional((value) => /[A-Z]/.test(value)),
  "Password must contain at least one uppercase letter",
);

export const hasSpecialChar = withMessage(
  optional((value) => /[!@#$%^&*(),.?":{}|<>]/.test(value)),
  "Password must contain at least one special character",
);
