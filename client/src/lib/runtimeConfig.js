function trimTrailingSlash(value) {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}

function withLeadingSlash(path) {
  if (!path) {
    return "/";
  }
  return path.startsWith("/") ? path : `/${path}`;
}

export function buildApiURL(path, opts = {}) {
  const apiBaseURL = opts.apiBaseURL ?? "";
  const cleanPath = withLeadingSlash(path);
  if (!apiBaseURL) {
    return cleanPath;
  }
  return `${trimTrailingSlash(apiBaseURL)}${cleanPath}`;
}

export function toAmountCents(amount) {
  const parsed = typeof amount === "number" ? amount : Number(amount);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    throw new Error("amount must be greater than zero");
  }
  const cents = Math.round(parsed * 100);
  if (cents <= 0) {
    throw new Error("amount is too small");
  }
  return cents;
}

export function buildWebSocketURL(opts = {}) {
  const wsBaseURL = opts.wsBaseURL ?? "";
  if (wsBaseURL) {
    const cleanBase = trimTrailingSlash(wsBaseURL);
    return cleanBase.endsWith("/ws") ? cleanBase : `${cleanBase}/ws`;
  }

  const apiBaseURL = opts.apiBaseURL ?? "";
  if (apiBaseURL) {
    const cleanBase = trimTrailingSlash(apiBaseURL);
    if (cleanBase.startsWith("https://")) {
      return `wss://${cleanBase.slice("https://".length)}/ws`;
    }
    if (cleanBase.startsWith("http://")) {
      return `ws://${cleanBase.slice("http://".length)}/ws`;
    }
  }

  const locationProtocol = opts.locationProtocol ?? window.location.protocol;
  const locationHost = opts.locationHost ?? window.location.host;
  const protocol = locationProtocol === "https:" ? "wss" : "ws";
  return `${protocol}://${locationHost}/ws`;
}
