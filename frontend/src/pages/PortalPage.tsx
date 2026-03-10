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
  Grid,
  InputLabel,
  MenuItem,
  Select,
  Snackbar,
  Stack,
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
  getInstanceDetails,
  getInstances,
  getModels,
  startInstance,
  stopAllInstances,
  stopInstance,
} from "../api";
import type { Instance } from "../types";

interface NoticeState {
  open: boolean;
  type: AlertColor;
  text: string;
}

export function PortalPage() {
  const [models, setModels] = useState<Record<string, string>>({});
  const [selectedModel, setSelectedModel] = useState("default");
  const [port, setPort] = useState<number>(8888);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [details, setDetails] = useState<unknown>(null);
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

  useEffect(() => {
    void Promise.all([loadModels(), loadInstances()]).catch((error: Error) => {
      showMessage(error.message, "error");
    });
  }, []);

  async function onStartInstance() {
    try {
      const data = await startInstance({
        model: selectedModel,
        pixelStreamingServerPort: Number(port),
      });
      showMessage(data.message || "Instance started");
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
      const data = await getInstanceDetails(id);
      setDetails(data);
    } catch (error) {
      showMessage((error as Error).message, "error");
    }
  }

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
      <Card elevation={0} sx={{ background: "linear-gradient(120deg, #155e63, #1f7a82)", color: "#fff" }}>
        <CardContent>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Distributed Pixel Manager
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
            <Grid size={{ xs: 12, md: 6 }}>
              <FormControl fullWidth>
                <InputLabel id="model-label">Model</InputLabel>
                <Select
                  labelId="model-label"
                  value={selectedModel}
                  label="Model"
                  onChange={(event) => setSelectedModel(event.target.value)}
                >
                  {modelNames.length === 0 && <MenuItem value="default">default</MenuItem>}
                  {modelNames.map((name) => (
                    <MenuItem key={name} value={name}>
                      {name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
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

      <Dialog open={Boolean(details)} onClose={() => setDetails(null)} fullWidth maxWidth="md">
        <DialogTitle>Instance Details</DialogTitle>
        <DialogContent>
          <Box
            component="pre"
            sx={{
              p: 2,
              m: 0,
              borderRadius: 2,
              overflow: "auto",
              backgroundColor: "rgba(17, 24, 39, 0.95)",
              color: "#c9f4f6",
            }}
          >
            {JSON.stringify(details, null, 2)}
          </Box>
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
