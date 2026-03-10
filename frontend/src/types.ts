export interface Instance {
  pixelStreamingId: string;
  host: string;
  port: number;
  pid: number;
  model: string;
  startTime: string;
  userId?: string;
  subscribed?: boolean;
  lastSubscribed?: string;
}

export interface ModelsResponse {
  models: Record<string, string>;
  message?: string;
}

export interface ManagersResponse {
  count: number;
  managers: Array<{ host: string; url: string }>;
}

export interface InstancesResponse {
  count: number;
  maxInstances: number;
  active: Instance[];
}

export interface StartInstancePayload {
  model: string;
  pixelStreamingServerPort: number;
}

export interface StartInstanceResponse {
  message?: string;
  error?: string;
}

export interface StopAllResponse {
  message: string;
  total: number;
  stopped: number;
  failed: number;
  errors?: string[];
}
