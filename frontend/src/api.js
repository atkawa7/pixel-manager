async function parseJson(response) {
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    const message = data?.error || data?.message || `Request failed (${response.status})`;
    throw new Error(message);
  }
  return data;
}

export async function getModels(baseURL = "") {
  const response = await fetch(`${baseURL}/models`);
  return parseJson(response);
}

export async function setModel(name, exePath, baseURL = "") {
  const response = await fetch(`${baseURL}/models`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, exePath }),
  });
  return parseJson(response);
}

export async function deleteModel(name) {
  const response = await fetch(`/models/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
  return parseJson(response);
}

export async function getManagers() {
  const response = await fetch("/managers");
  return parseJson(response);
}

export async function getInstances(baseURL = "") {
  const response = await fetch(`${baseURL}/instances`);
  return parseJson(response);
}

export async function startInstance(payload) {
  const response = await fetch("/instances", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return parseJson(response);
}

export async function stopInstance(id) {
  const response = await fetch(`/instances/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  return parseJson(response);
}

export async function getInstanceDetails(id) {
  const response = await fetch(`/instances/${encodeURIComponent(id)}`);
  return parseJson(response);
}

export async function stopAllInstances() {
  const response = await fetch("/instances", { method: "DELETE" });
  return parseJson(response);
}
