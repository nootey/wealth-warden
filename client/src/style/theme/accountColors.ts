import styleHelper from "../../utils/style_helper.ts";

const baseColors: Record<string, string> = {
  cash: "#6E64CC",
  investment: "#486EE8",
  crypto: "#78BEFF",
  property: "#55EDD9",
  vehicle: "#30BF70",
  other_asset: "#71D17B",
  credit_card: "#ef4444",
  loan: "#f97316",
  other_liability: "#eab308",
};

function baseColorFor(type?: string): string {
  const t = (type || "other_asset").toLowerCase();
  return baseColors[t] ?? baseColors.other_asset!;
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
