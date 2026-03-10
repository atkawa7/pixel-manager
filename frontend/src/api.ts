import type {
  InstancesResponse,
  ManagersResponse,
  ModelsResponse,
  StartInstancePayload,
  StartInstanceResponse,
  StopAllResponse,
} from "./types";

interface ApiErrorPayload {
  error?: string;
  message?: string;
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
  const response = await fetch(`${baseURL}/models`);
  return parseJson<ModelsResponse>(response);
}

export async function setModel(
  name: string,
  exePath: string,
  baseURL = "",
): Promise<ModelsResponse> {
  const response = await fetch(`${baseURL}/models`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, exePath }),
  });
  return parseJson<ModelsResponse>(response);
}

export async function deleteModel(name: string): Promise<ModelsResponse> {
  const response = await fetch(`/models/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
  return parseJson<ModelsResponse>(response);
}

export async function getManagers(): Promise<ManagersResponse> {
  const response = await fetch("/managers");
  return parseJson<ManagersResponse>(response);
}

export async function getInstances(baseURL = ""): Promise<InstancesResponse> {
  const response = await fetch(`${baseURL}/instances`);
  return parseJson<InstancesResponse>(response);
}

export async function startInstance(
  payload: StartInstancePayload,
): Promise<StartInstanceResponse> {
  const response = await fetch("/instances", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return parseJson<StartInstanceResponse>(response);
}

export async function stopInstance(id: string): Promise<{ message?: string }> {
  const response = await fetch(`/instances/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  return parseJson<{ message?: string }>(response);
}

export async function getInstanceDetails(id: string): Promise<unknown> {
  const response = await fetch(`/instances/${encodeURIComponent(id)}`);
  return parseJson<unknown>(response);
}

export async function stopAllInstances(): Promise<StopAllResponse> {
  const response = await fetch("/instances", { method: "DELETE" });
  return parseJson<StopAllResponse>(response);
}
