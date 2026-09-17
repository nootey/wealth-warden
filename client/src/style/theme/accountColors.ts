import styleHelper from "../../utils/style_helper.ts";
import { ACCOUNT_TOKENS, tokenColor } from "./tokens.ts";

function baseColorFor(type?: string): string {
  const t = (type || "other_asset").toLowerCase();
  const token = ACCOUNT_TOKENS[t] ?? ACCOUNT_TOKENS.other_asset!;
  return tokenColor(token);
}

export type AccountTypeColor = {
  bg: string;
  fg: string;
  border: string;
};

export function colorForAccountType(type?: string): AccountTypeColor {
  const border = baseColorFor(type);
  const bg = styleHelper.shadeHsl(border, -0.2);
  const fg = styleHelper.shadeHsl(border, +0.2);
  return { bg, fg, border };
}
