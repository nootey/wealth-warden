import styleHelper from "../../utils/style_helper.ts";
import { useThemeStore } from "../../services/stores/theme_store.ts";
import { accentTokens, tokenColor } from "./tokens.ts";

function baseColorFor(type?: string): string {
  const t = (type || "other_asset").toLowerCase();
  const tokens = accentTokens(useThemeStore().accent).account;
  const token = tokens[t] ?? tokens.other_asset!;
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
