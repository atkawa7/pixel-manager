import type {
  AppConfigResponse,
  InstanceDetailsResponse,
  InstancesResponse,
  ManagersResponse,
  ModelsResponse,
  InstanceLogsResponse,
  StartInstancePayload,
  StartInstanceResponse,
  StopAllResponse,
} from "./types";

interface ApiErrorPayload {
  error?: string;
  message?: string;
}

const DEFAULT_API_BASE = import.meta.env.DEV ? "http://localhost:4000" : "/api";
const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) || DEFAULT_API_BASE;

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/, "");
}

function resolveBase(baseURL?: string): string {
  const raw = baseURL && baseURL !== "" ? baseURL : API_BASE;
  const normalized = trimTrailingSlash(raw);

  if (
    import.meta.env.PROD &&
    /^https?:\/\//.test(normalized) &&
    !normalized.toLowerCase().endsWith("/api")
  ) {
    return `${normalized}/api`;
  }

  return normalized;
}

function buildURL(path: string, baseURL?: string): string {
  const base = resolveBase(baseURL);
  return `${base}${path}`;
}

async function parseJson<T>(response: Response): Promise<T> {
  const data = (await response.json().catch(() => ({}))) as ApiErrorPayload & T;
  if (!response.ok) {
    const message = data.error || data.message || `Request failed (${response.status})`;
    throw new Error(message);
  }
  return data;
}

export async function getModels(baseURL = ""): Promise<ModelsResponse> {
  const response = await fetch(buildURL("/models", baseURL));
  return parseJson<ModelsResponse>(response);
}

export async function setModel(
  name: string,
  exePath: string,
  baseURL = "",
): Promise<ModelsResponse> {
  const response = await fetch(buildURL("/models", baseURL), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, exePath }),
  });
  return parseJson<ModelsResponse>(response);
}

export async function deleteModel(name: string): Promise<ModelsResponse> {
  const response = await fetch(buildURL(`/models/${encodeURIComponent(name)}`), {
    method: "DELETE",
  });
  return parseJson<ModelsResponse>(response);
}

export async function getManagers(): Promise<ManagersResponse> {
  const response = await fetch(buildURL("/managers"));
  return parseJson<ManagersResponse>(response);
}

export async function getAppConfig(): Promise<AppConfigResponse> {
  const response = await fetch(buildURL("/config"));
  return parseJson<AppConfigResponse>(response);
}

export async function getInstances(baseURL = ""): Promise<InstancesResponse> {
  const response = await fetch(buildURL("/instances", baseURL));
  return parseJson<InstancesResponse>(response);
}

export async function startInstance(
  payload: StartInstancePayload,
): Promise<StartInstanceResponse> {
  const response = await fetch(buildURL("/instances"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return parseJson<StartInstanceResponse>(response);
}

export async function stopInstance(id: string): Promise<{ message?: string }> {
  const response = await fetch(buildURL(`/instances/${encodeURIComponent(id)}`), {
    method: "DELETE",
  });
  return parseJson<{ message?: string }>(response);
}

export async function getInstanceDetails(id: string): Promise<InstanceDetailsResponse> {
  const response = await fetch(buildURL(`/instances/${encodeURIComponent(id)}`));
  return parseJson<InstanceDetailsResponse>(response);
}

export async function getInstanceLogs(
  id: string,
  tail = 200,
): Promise<InstanceLogsResponse> {
  const response = await fetch(
    buildURL(`/instances/${encodeURIComponent(id)}/logs?tail=${tail}`),
  );
  return parseJson<InstanceLogsResponse>(response);
}

export async function stopAllInstances(): Promise<StopAllResponse> {
  const response = await fetch(buildURL("/instances"), { method: "DELETE" });
  return parseJson<StopAllResponse>(response);
}

export function getOpenAPIURL(baseURL = ""): string {
  return buildURL("/openapi.json", baseURL);
}
