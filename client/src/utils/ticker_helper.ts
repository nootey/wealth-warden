import type { InvestmentType } from "../models/investment_models.ts";

export interface TickerParts {
  name: string;
  exchange: string;
  currency: string;
}

// buildTicker composes the canonical symbol the backend stores and sends to
// Yahoo. Mirrors finance.BuildSymbol on the server.
//   crypto:    TICKER-CURRENCY (currency defaults to USD)
//   stock/ETF: TICKER.EXCHANGE (or bare TICKER for US listings)
function buildTicker(
  parts: TickerParts,
  investmentType: InvestmentType,
): string {
  const name = parts.name.trim().toUpperCase();
  if (!name) return "";

  if (investmentType === "crypto") {
    const currency = (parts.currency || "USD").trim().toUpperCase();
    return `${name}-${currency}`;
  }

  const exchange = parts.exchange.trim().toUpperCase();
  return exchange ? `${name}.${exchange}` : name;
}

// parseTicker splits a stored symbol back into its parts. Mirrors
// finance.ParseSymbol on the server.
function parseTicker(
  raw: string,
  investmentType: InvestmentType,
): TickerParts {
  const value = (raw || "").trim().toUpperCase();

  if (investmentType === "crypto") {
    const [name = "", currency = ""] = value.split("-");
    return { name, exchange: "", currency };
  }

  const [name = "", exchange = ""] = value.split(".");
  return { name, exchange, currency: "" };
}

export default { buildTicker, parseTicker };
