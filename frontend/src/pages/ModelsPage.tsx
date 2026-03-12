import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import DeleteOutlineRoundedIcon from "@mui/icons-material/DeleteOutlineRounded";
import EditRoundedIcon from "@mui/icons-material/EditRounded";
import PlayArrowRoundedIcon from "@mui/icons-material/PlayArrowRounded";
import RefreshRoundedIcon from "@mui/icons-material/RefreshRounded";
import SaveRoundedIcon from "@mui/icons-material/SaveRounded";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Grid,
  MenuItem,
  Snackbar,
  Stack,
  Switch,
  TextField,
  Typography,
} from "@mui/material";
import type { AlertColor } from "@mui/material";
import { deleteModel, getModels, setModel, startInstance } from "../api";

interface NoticeState {
  open: boolean;
  text: string;
  type: AlertColor;
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

export function ModelsPage() {
  const [models, setModels] = useState<Record<string, string>>({});
  const [name, setName] = useState("");
  const [exePath, setExePath] = useState("");
  const [startModel, setStartModel] = useState("default");
  const [startPort, setStartPort] = useState(8888);
  const [selectedCodec, setSelectedCodec] = useState("H264");
  const [selectedResolution, setSelectedResolution] = useState("720p");
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
  const [startDialogOpen, setStartDialogOpen] = useState(false);
  const [modelToDelete, setModelToDelete] = useState("");
  const [notice, setNotice] = useState<NoticeState>({
    open: false,
    text: "",
    type: "success",
  });

  function notify(text: string, type: AlertColor = "success") {
    setNotice({ open: true, text, type });
  }

  async function loadModels() {
    try {
      const data = await getModels();
      const nextModels = data.models || {};
      setModels(nextModels);
      if (!nextModels[startModel]) {
        setStartModel(nextModels.default ? "default" : Object.keys(nextModels)[0] || "default");
      }
    } catch (error) {
      notify((error as Error).message, "error");
    }
  }

  useEffect(() => {
    void loadModels();
  }, []);

  async function onSaveModel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim() || !exePath.trim()) {
      notify("Model name and path are required", "error");
      return;
    }
    try {
      const data = await setModel(name.trim(), exePath.trim());
      setModels(data.models || {});
      notify(data.message || "Model saved");
      setName("");
      setExePath("");
    } catch (error) {
      notify((error as Error).message, "error");
    }
  }

  async function onConfirmDelete() {
    if (!modelToDelete) {
      return;
    }
    try {
      const data = await deleteModel(modelToDelete);
      setModels(data.models || {});
      notify(data.message || "Model deleted");
    } catch (error) {
      notify((error as Error).message, "error");
    } finally {
      setModelToDelete("");
    }
  }

  async function onStartFromModel() {
    if (!startModel) {
      notify("Select a model first", "error");
      return;
    }
    const preset =
      resolutionPresets.find((item) => item.id === selectedResolution) || resolutionPresets[2];
    try {
      const data = await startInstance({
        model: startModel,
        pixelStreamingServerPort: Number(startPort) || 8888,
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
      });
      notify(data.message || "Instance started");
      setStartDialogOpen(false);
    } catch (error) {
      notify((error as Error).message, "error");
    }
  }

  return (
    <Stack spacing={2.5}>
      <Card elevation={0} sx={{ background: "#0f766e", color: "#fff" }}>
        <CardContent>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Models Management
          </Typography>
          <Typography sx={{ opacity: 0.95 }}>
            Configure and maintain executable models for new pixel streaming instances.
          </Typography>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Stack direction={{ xs: "column", md: "row" }} spacing={1.5} alignItems={{ md: "center" }}>
            <Alert severity="info" sx={{ flex: 1 }}>
              Build uploads were moved to the Builds page. Create models from uploaded build executables there.
            </Alert>
            <Button component="a" href="/builds" variant="outlined">
              Open Builds
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Add or Update Model
          </Typography>
          <Box component="form" onSubmit={onSaveModel}>
            <Grid container spacing={2}>
              <Grid size={{ xs: 12, md: 4 }}>
                <TextField
                  fullWidth
                  label="Model Name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  placeholder="model-a"
                />
              </Grid>
              <Grid size={{ xs: 12, md: 8 }}>
                <TextField
                  fullWidth
                  label="Executable Path"
                  value={exePath}
                  onChange={(event) => setExePath(event.target.value)}
                  placeholder="C:\\path\\to\\model.exe"
                />
              </Grid>
              <Grid size={12}>
                <Stack direction="row" spacing={1}>
                  <Button type="submit" variant="contained" startIcon={<SaveRoundedIcon />}>
                    Save Model
                  </Button>
                  <Button
                    variant="outlined"
                    onClick={() => {
                      setName("");
                      setExePath("");
                    }}
                  >
                    Clear
                  </Button>
                </Stack>
              </Grid>
            </Grid>
          </Box>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Start Instance From Model
          </Typography>
          <Stack direction={{ xs: "column", md: "row" }} spacing={1} alignItems={{ md: "center" }}>
            <TextField
              select
              fullWidth
              label="Model"
              value={startModel}
              onChange={(event) => setStartModel(event.target.value)}
            >
              {Object.keys(models).map((modelName) => (
                <MenuItem key={modelName} value={modelName}>
                  {modelName}
                </MenuItem>
              ))}
            </TextField>
            <Button
              variant="contained"
              startIcon={<PlayArrowRoundedIcon />}
              onClick={() => setStartDialogOpen(true)}
            >
              Open Start Form
            </Button>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
            <Typography variant="h6">Available Models</Typography>
            <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={() => void loadModels()}>
              Refresh
            </Button>
          </Box>
          <Grid container spacing={2}>
            {Object.entries(models).map(([modelName, path]) => {
              const isDefault = modelName === "default";
              return (
                <Grid key={modelName} size={{ xs: 12, md: 6 }}>
                  <Card variant="outlined" sx={{ borderColor: isDefault ? "primary.main" : "divider" }}>
                    <CardContent>
                      <Box sx={{ display: "flex", justifyContent: "space-between", gap: 2, alignItems: "start" }}>
                        <Box>
                          <Typography variant="h6">
                            {modelName}
                            {isDefault && (
                              <Typography
                                component="span"
                                variant="caption"
                                sx={{
                                  ml: 1,
                                  px: 1,
                                  py: 0.4,
                                  bgcolor: "primary.main",
                                  color: "#fff",
                                  borderRadius: 1,
                                }}
                              >
                                Default
                              </Typography>
                            )}
                          </Typography>
                          <Typography
                            variant="body2"
                            color="text.secondary"
                            sx={{ mt: 1, fontFamily: "monospace", wordBreak: "break-all" }}
                          >
                            {path}
                          </Typography>
                        </Box>
                        <Stack direction="row" spacing={0.5}>
                          <Button
                            size="small"
                            startIcon={<EditRoundedIcon />}
                            onClick={() => {
                              setName(modelName);
                              setExePath(path);
                            }}
                          >
                            Edit
                          </Button>
                          <Button
                            size="small"
                            startIcon={<PlayArrowRoundedIcon />}
                            onClick={() => {
                              setStartModel(modelName);
                              setStartDialogOpen(true);
                            }}
                          >
                            Start
                          </Button>
                          {!isDefault && (
                            <Button
                              size="small"
                              color="error"
                              startIcon={<DeleteOutlineRoundedIcon />}
                              onClick={() => setModelToDelete(modelName)}
                            >
                              Delete
                            </Button>
                          )}
                        </Stack>
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
              );
            })}
            {Object.keys(models).length === 0 && (
              <Grid size={12}>
                <Alert severity="info">No models configured.</Alert>
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>

      <Dialog open={Boolean(modelToDelete)} onClose={() => setModelToDelete("")}>
        <DialogTitle>Delete Model</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete <strong>{modelToDelete}</strong>?
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setModelToDelete("")}>Cancel</Button>
          <Button color="error" variant="contained" onClick={() => void onConfirmDelete()}>
            Delete
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={startDialogOpen} onClose={() => setStartDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Start Instance From Model</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 0.5 }}>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                select
                fullWidth
                label="Model"
                value={startModel}
                onChange={(event) => setStartModel(event.target.value)}
              >
                {Object.keys(models).map((modelName) => (
                  <MenuItem key={modelName} value={modelName}>
                    {modelName}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <TextField
                select
                fullWidth
                label="Encoder Codec"
                value={selectedCodec}
                onChange={(event) => setSelectedCodec(event.target.value)}
              >
                {codecOptions.map((codec) => (
                  <MenuItem key={codec.value} value={codec.value}>
                    {codec.label}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                select
                fullWidth
                label="Resolution"
                value={selectedResolution}
                onChange={(event) => setSelectedResolution(event.target.value)}
              >
                {resolutionPresets.map((preset) => (
                  <MenuItem key={preset.id} value={preset.id}>
                    {preset.label}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid size={{ xs: 12, md: 2 }}>
              <TextField
                label="Port"
                type="number"
                fullWidth
                value={startPort}
                onChange={(event) => setStartPort(Number(event.target.value) || 0)}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                label="Min Quality"
                type="number"
                fullWidth
                value={encoderMinQuality}
                onChange={(event) => setEncoderMinQuality(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 3 }}>
              <TextField
                label="Max Quality"
                type="number"
                fullWidth
                value={encoderMaxQuality}
                onChange={(event) => setEncoderMaxQuality(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                label="WebRTC Min Bitrate (Mbps)"
                type="number"
                fullWidth
                value={webrtcMinBitrateMbps}
                onChange={(event) => setWebrtcMinBitrateMbps(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                label="WebRTC Start Bitrate (Mbps)"
                type="number"
                fullWidth
                value={webrtcStartBitrateMbps}
                onChange={(event) => setWebrtcStartBitrateMbps(Number(event.target.value))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
              <TextField
                label="WebRTC Max Bitrate (Mbps)"
                type="number"
                fullWidth
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
                label="Full StdOut Log Output"
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
              <TextField
                select
                fullWidth
                label="D3D Renderer"
                value={d3dRenderer}
                onChange={(event) =>
                  setD3dRenderer(event.target.value as "" | "d3d11" | "d3d12")
                }
              >
                {d3dRendererOptions.map((opt) => (
                  <MenuItem key={opt.label} value={opt.value}>
                    {opt.label}
                  </MenuItem>
                ))}
              </TextField>
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
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setStartDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" startIcon={<PlayArrowRoundedIcon />} onClick={() => void onStartFromModel()}>
            Start
          </Button>
        </DialogActions>
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
