import { useEffect, useState } from "react";
import PlayArrowRoundedIcon from "@mui/icons-material/PlayArrowRounded";
import RefreshRoundedIcon from "@mui/icons-material/RefreshRounded";
import StopCircleRoundedIcon from "@mui/icons-material/StopCircleRounded";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  Snackbar,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from "@mui/material";
import type { AlertColor } from "@mui/material";
import {
  getInstanceLogs,
  getInstanceDetails,
  getInstances,
  getModels,
  startInstance,
  stopAllInstances,
  stopInstance,
} from "../api";
import type { Instance, InstanceDetailsResponse } from "../types";

interface NoticeState {
  open: boolean;
  type: AlertColor;
  text: string;
}

interface ResolutionPreset {
  id: string;
  label: string;
  resX: number;
  resY: number;
}

const resolutionPresets: ResolutionPreset[] = [
  { id: "360p", label: "360p (640x360)", resX: 640, resY: 360 },
  { id: "480p", label: "480p (854x480)", resX: 854, resY: 480 },
  { id: "720p", label: "720p (1280x720)", resX: 1280, resY: 720 },
  { id: "1440p", label: "1440p (2560x1440)", resX: 2560, resY: 1440 },
  { id: "1080p", label: "1080p (1920x1080)", resX: 1920, resY: 1080 },
];

const codecOptions = [
  { label: "H.264", value: "H264" },
  { label: "VP8", value: "VP8" },
  { label: "VP9", value: "VP9" },
  { label: "AV1", value: "AV1" },
];

const d3dRendererOptions = [
  { label: "Auto", value: "" },
  { label: "d3d11", value: "d3d11" },
  { label: "d3d12", value: "d3d12" },
];

export function PortalPage() {
  const allModelsValue = "__all_models__";
  const [models, setModels] = useState<Record<string, string>>({});
  const [selectedModel, setSelectedModel] = useState("default");
  const [selectedCodec, setSelectedCodec] = useState("H264");
  const [selectedResolution, setSelectedResolution] = useState("720p");
  const [port, setPort] = useState<number>(8888);
  const [encoderMinQuality, setEncoderMinQuality] = useState<number>(-1);
  const [encoderMaxQuality, setEncoderMaxQuality] = useState<number>(-1);
  const [webrtcMinBitrateMbps, setWebrtcMinBitrateMbps] = useState<number>(1);
  const [webrtcStartBitrateMbps, setWebrtcStartBitrateMbps] = useState<number>(10);
  const [webrtcMaxBitrateMbps, setWebrtcMaxBitrateMbps] = useState<number>(100);
  const [pixelStreamingHudStats, setPixelStreamingHudStats] = useState<boolean>(false);
  const [stdOut, setStdOut] = useState<boolean>(false);
  const [fullStdOutLogOutput, setFullStdOutLogOutput] = useState<boolean>(false);
  const [webrtcDisableReceiveAudio, setWebrtcDisableReceiveAudio] = useState<boolean>(false);
  const [webrtcDisableTransmitAudio, setWebrtcDisableTransmitAudio] = useState<boolean>(false);
  const [d3dRenderer, setD3dRenderer] = useState<"" | "d3d11" | "d3d12">("");
  const [d3dDebug, setD3dDebug] = useState<boolean>(false);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [detailsInstanceId, setDetailsInstanceId] = useState<string>("");
  const [details, setDetails] = useState<InstanceDetailsResponse | null>(null);
  const [instanceLogs, setInstanceLogs] = useState<string[]>([]);
  const [notice, setNotice] = useState<NoticeState>({
    open: false,
    type: "success",
    text: "",
  });

  const modelNames = Object.keys(models);

  async function loadModels() {
    const data = await getModels();
    const next = data.models || {};
    setModels(next);
    if (!next[selectedModel]) {
      setSelectedModel(next.default ? "default" : Object.keys(next)[0] || "default");
    }
  }

  async function loadInstances() {
    const data = await getInstances();
    setInstances(data.active || []);
  }

  function showMessage(text: string, type: AlertColor = "success") {
    setNotice({ open: true, text, type });
  }

  function formatLaunchCommand(instance: InstanceDetailsResponse): string {
    const exe = instance.executablePath || "<unknown executable>";
    const args = instance.args || [];
    if (args.length === 0) {
      return exe;
    }
    return `${exe}\n${args.join("\n")}`;
  }

  useEffect(() => {
    void Promise.all([loadModels(), loadInstances()]).catch((error: Error) => {
      showMessage(error.message, "error");
    });
  }, []);

  async function onStartInstance() {
    const preset =
      resolutionPresets.find((item) => item.id === selectedResolution) || resolutionPresets[2];
    const targetModels =
      selectedModel === allModelsValue
        ? modelNames
        : [selectedModel];
    if (targetModels.length === 0) {
      showMessage("No models configured", "error");
      return;
    }

    try {
      const results = await Promise.allSettled(
        targetModels.map((modelName) =>
          startInstance({
            model: modelName,
            pixelStreamingServerPort: Number(port),
            encoderCodec: selectedCodec,
            resX: preset.resX,
            resY: preset.resY,
            encoderMinQuality,
            encoderMaxQuality,
            webrtcMinBitrateMbps,
            webrtcStartBitrateMbps,
            webrtcMaxBitrateMbps,
            pixelStreamingHudStats,
            stdOut,
            fullStdOutLogOutput,
            webrtcDisableReceiveAudio,
            webrtcDisableTransmitAudio,
            d3dRenderer,
            d3dDebug,
          }),
        ),
      );
      const successCount = results.filter((item) => item.status === "fulfilled").length;
      const failureCount = results.length - successCount;
      if (failureCount === 0) {
        showMessage(
          successCount === 1
            ? "Instance started"
            : `Started ${successCount} instance(s) across all models`,
        );
      } else {
        const firstError = results.find((item) => item.status === "rejected") as
          | PromiseRejectedResult
          | undefined;
        showMessage(
          `Started ${successCount}/${results.length}. Failed ${failureCount}. ${firstError?.reason?.message || ""}`.trim(),
          "error",
        );
      }
      await loadInstances();
    } catch (error) {
      showMessage((error as Error).message, "error");
    }
  }

  async function onStopInstance(id: string) {
    if (!window.confirm(`Stop instance ${id}?`)) {
      return;
    }
    try {
      const data = await stopInstance(id);
      showMessage(data.message || "Instance stopped");
      await loadInstances();
    } catch (error) {
      showMessage((error as Error).message, "error");
    }
  }

  async function onShowDetails(id: string) {
    try {
      const [data, logs] = await Promise.all([
        getInstanceDetails(id),
        getInstanceLogs(id, 300),
      ]);
      setDetailsInstanceId(id);
      setDetails(data);
      setInstanceLogs(logs.lines || []);
    } catch (error) {
      showMessage((error as Error).message, "error");
    }
  }

  useEffect(() => {
    if (!detailsInstanceId) {
      return;
    }

    const interval = setInterval(() => {
      void getInstanceLogs(detailsInstanceId, 300)
        .then((logs) => setInstanceLogs(logs.lines || []))
        .catch(() => {
          // Keep existing logs on polling failure; user still has previous output.
        });
    }, 2000);

    return () => clearInterval(interval);
  }, [detailsInstanceId]);

  async function onStopAll() {
    if (instances.length === 0) {
      showMessage("No active instances to stop", "info");
      return;
    }
    if (!window.confirm(`Stop all ${instances.length} instance(s) across the cluster?`)) {
      return;
    }
    try {
      const result = await stopAllInstances();
      showMessage(
        `Stop complete: total ${result.total}, stopped ${result.stopped}, failed ${result.failed}`,
      );
      await loadInstances();
    } catch (error) {
      showMessage((error as Error).message, "error");
    }
  }

  return (
    <Stack spacing={2.5}>
      <Card elevation={0} sx={{ background: "#155e63", color: "#fff" }}>
        <CardContent>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Instances
          </Typography>
          <Typography variant="body1" sx={{ opacity: 0.9 }}>
            Manage Unreal Engine pixel streaming instances across your cluster.
          </Typography>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Start New Instance
          </Typography>
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 3 }}>
              <FormControl fullWidth>
                <InputLabel id="model-label">Model</InputLabel>
                <Select
                  labelId="model-label"
                  value={selectedModel}
                  label="Model"
                  onChange={(event) => setSelectedModel(event.target.value)}
                >
                  {modelNames.length === 0 && <MenuItem value="default">default</MenuItem>}
                  {modelNames.length > 1 && (
                    <MenuItem value={allModelsValue}>All Models</MenuItem>
                  )}
                  {modelNames.map((name) => (
                    <MenuItem key={name} value={name}>
                      {name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <FormControl fullWidth>
                <InputLabel id="codec-label">Encoder Codec</InputLabel>
                <Select
                  labelId="codec-label"
                  value={selectedCodec}
                  label="Encoder Codec"
                  onChange={(event) => setSelectedCodec(event.target.value)}
                >
                  {codecOptions.map((codec) => (
                    <MenuItem key={codec.value} value={codec.value}>
                      {codec.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <FormControl fullWidth>
                <InputLabel id="resolution-label">Resolution</InputLabel>
                <Select
                  labelId="resolution-label"
                  value={selectedResolution}
                  label="Resolution"
                  onChange={(event) => setSelectedResolution(event.target.value)}
                >
                  {resolutionPresets.map((preset) => (
                    <MenuItem key={preset.id} value={preset.id}>
                      {preset.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <TextField
                label="Pixel Streaming Port"
                type="number"
                fullWidth
                value={port}
                onChange={(event) => setPort(Number(event.target.value) || 0)}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <Button
                fullWidth
                variant="contained"
                size="large"
                startIcon={<PlayArrowRoundedIcon />}
                onClick={onStartInstance}
                sx={{ height: "100%" }}
              >
                Start
              </Button>
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                label="Min Quality"
                type="number"
                fullWidth
                helperText="-1 or 0-100 (preferred over QP)"
                value={encoderMinQuality}
                onChange={(event) => setEncoderMinQuality(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                label="Max Quality"
                type="number"
                fullWidth
                helperText="-1 or 0-100"
                value={encoderMaxQuality}
                onChange={(event) => setEncoderMaxQuality(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                label="WebRTC Min Bitrate (Mbps)"
                type="number"
                fullWidth
                helperText=">= 1"
                value={webrtcMinBitrateMbps}
                onChange={(event) => setWebrtcMinBitrateMbps(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                label="WebRTC Start Bitrate (Mbps)"
                type="number"
                fullWidth
                helperText=">= 1"
                value={webrtcStartBitrateMbps}
                onChange={(event) => setWebrtcStartBitrateMbps(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                label="WebRTC Max Bitrate (Mbps)"
                type="number"
                fullWidth
                helperText=">= 1"
                value={webrtcMaxBitrateMbps}
                onChange={(event) => setWebrtcMaxBitrateMbps(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={pixelStreamingHudStats}
                    onChange={(event) => setPixelStreamingHudStats(event.target.checked)}
                  />
                }
                label="PixelStreaming HUD Stats"
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <FormControlLabel
                control={<Switch checked={stdOut} onChange={(event) => setStdOut(event.target.checked)} />}
                label="Enable StdOut"
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={fullStdOutLogOutput}
                    onChange={(event) => setFullStdOutLogOutput(event.target.checked)}
                  />
                }
                label="Enable Full StdOut Log Output"
              />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={webrtcDisableReceiveAudio}
                    onChange={(event) => setWebrtcDisableReceiveAudio(event.target.checked)}
                  />
                }
                label="Disable WebRTC Receive Audio"
              />
            </Grid>
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={webrtcDisableTransmitAudio}
                    onChange={(event) => setWebrtcDisableTransmitAudio(event.target.checked)}
                  />
                }
                label="Disable WebRTC Transmit Audio"
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <FormControl fullWidth>
                <InputLabel id="d3d-renderer-label">D3D Renderer</InputLabel>
                <Select
                  labelId="d3d-renderer-label"
                  value={d3dRenderer}
                  label="D3D Renderer"
                  onChange={(event) =>
                    setD3dRenderer(event.target.value as "" | "d3d11" | "d3d12")
                  }
                >
                  {d3dRendererOptions.map((opt) => (
                    <MenuItem key={opt.label} value={opt.value}>
                      {opt.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 8 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={d3dDebug}
                    onChange={(event) => setD3dDebug(event.target.checked)}
                  />
                }
                label="Use D3D Debug Device (-d3ddebug)"
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
            <Typography variant="h6">Active Instances</Typography>
            <Stack direction="row" spacing={1}>
              <Button
                variant="outlined"
                color="error"
                startIcon={<StopCircleRoundedIcon />}
                onClick={onStopAll}
              >
                Stop All
              </Button>
              <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={loadInstances}>
                Refresh
              </Button>
            </Stack>
          </Box>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>ID</TableCell>
                <TableCell>Host</TableCell>
                <TableCell>Port</TableCell>
                <TableCell>PID</TableCell>
                <TableCell>Model</TableCell>
                <TableCell>Started</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {instances.map((instance) => (
                <TableRow key={instance.pixelStreamingId}>
                  <TableCell sx={{ fontFamily: "monospace", maxWidth: 220 }}>
                    {instance.pixelStreamingId}
                  </TableCell>
                  <TableCell>{instance.host}</TableCell>
                  <TableCell>{instance.port}</TableCell>
                  <TableCell>{instance.pid}</TableCell>
                  <TableCell>{instance.model}</TableCell>
                  <TableCell>{instance.startTime}</TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button size="small" onClick={() => onShowDetails(instance.pixelStreamingId)}>
                        Details
                      </Button>
                      <Button
                        size="small"
                        color="error"
                        variant="outlined"
                        onClick={() => onStopInstance(instance.pixelStreamingId)}
                      >
                        Stop
                      </Button>
                    </Stack>
                  </TableCell>
                </TableRow>
              ))}
              {instances.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7}>
                    <Typography color="text.secondary">No active instances.</Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog
        open={Boolean(details)}
        onClose={() => {
          setDetails(null);
          setDetailsInstanceId("");
          setInstanceLogs([]);
        }}
        fullWidth
        maxWidth="md"
      >
        <DialogTitle>Instance Details</DialogTitle>
        <DialogContent>
          <Stack spacing={2}>
            <Box>
              <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 600 }}>
                Launch Command
              </Typography>
              <Box
                component="pre"
                sx={{
                  p: 2,
                  m: 0,
                  borderRadius: 2,
                  overflow: "auto",
                  backgroundColor: "#0f172a",
                  color: "#d8e9ff",
                  maxHeight: 240,
                }}
              >
                {details ? formatLaunchCommand(details) : ""}
              </Box>
            </Box>
            <Box>
              <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 600 }}>
                Instance Metadata
              </Typography>
              <Box
                component="pre"
                sx={{
                  p: 2,
                  m: 0,
                  borderRadius: 2,
                  overflow: "auto",
                  backgroundColor: "rgba(17, 24, 39, 0.95)",
                  color: "#c9f4f6",
                  maxHeight: 260,
                }}
              >
                {JSON.stringify(details, null, 2)}
              </Box>
            </Box>
            <Box>
              <Typography variant="subtitle1" sx={{ mb: 1, fontWeight: 600 }}>
                Process Logs
              </Typography>
              <Box
                component="pre"
                sx={{
                  p: 2,
                  m: 0,
                  borderRadius: 2,
                  overflow: "auto",
                  backgroundColor: "#0b1220",
                  color: "#d6e3ff",
                  minHeight: 220,
                  maxHeight: 360,
                }}
              >
                {instanceLogs.length > 0 ? instanceLogs.join("\n") : "No logs captured yet."}
              </Box>
            </Box>
          </Stack>
        </DialogContent>
      </Dialog>

      <Snackbar
        open={notice.open}
        autoHideDuration={4000}
        onClose={() => setNotice((prev) => ({ ...prev, open: false }))}
      >
        <Alert severity={notice.type} variant="filled">
          {notice.text}
        </Alert>
      </Snackbar>
    </Stack>
  );
}
