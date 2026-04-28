import { helpers, minLength } from "@vuelidate/validators";

export const passwordMinLength = minLength(6);

export const noSpaces = helpers.withMessage(
  "Password cannot contain spaces",
  (value: string) => !/\s/.test(value ?? ""),
);

export const hasNumber = helpers.withMessage(
  "Password must contain at least one number",
  helpers.regex(/\d/),
);

export const hasUppercase = helpers.withMessage(
  "Password must contain at least one uppercase letter",
  helpers.regex(/[A-Z]/),
);

export const hasSpecialChar = helpers.withMessage(
  "Password must contain at least one special character",
  helpers.regex(/[!@#$%^&*(),.?":{}|<>]/),
);
