export interface Instance {
  pixelStreamingId: string;
  pixelStreamingIp?: string;
  host: string;
  port: number;
  pixelStreamingServerPort?: number;
  pid: number;
  model: string;
  executablePath?: string;
  args?: string[];
  startTime: string;
  userId?: string;
  subscribed?: boolean;
  lastSubscribed?: string;
}

export interface InstanceDetailsResponse extends Instance {
  exists: boolean;
  message?: string;
}

export interface ModelsResponse {
  models: Record<string, string>;
  message?: string;
}

export interface ManagersResponse {
  count: number;
  managers: Array<{ host: string; name: string; url: string }>;
}

export interface InstancesResponse {
  count: number;
  maxInstances: number;
  active: Instance[];
}

export interface StartInstancePayload {
  model: string;
  pixelStreamingServerPort: number;
  encoderCodec?: string;
  resX?: number;
  resY?: number;
  encoderMinQuality?: number;
  encoderMaxQuality?: number;
  webrtcMinBitrateMbps?: number;
  webrtcStartBitrateMbps?: number;
  webrtcMaxBitrateMbps?: number;
  pixelStreamingHudStats?: boolean;
  stdOut?: boolean;
  fullStdOutLogOutput?: boolean;
  webrtcDisableReceiveAudio?: boolean;
  webrtcDisableTransmitAudio?: boolean;
  d3dRenderer?: "" | "d3d11" | "d3d12";
  d3dDebug?: boolean;
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

export interface InstanceLogsResponse {
  instanceId: string;
  tail: number;
  lines: string[];
}

export interface AppConfigResponse {
  configPath: string;
  config: Record<string, string | number | boolean | null>;
}
