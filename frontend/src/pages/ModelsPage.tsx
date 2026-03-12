import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import DeleteOutlineRoundedIcon from "@mui/icons-material/DeleteOutlineRounded";
import EditRoundedIcon from "@mui/icons-material/EditRounded";
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
  Grid,
  Snackbar,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import type { AlertColor } from "@mui/material";
import type { BuildInfo } from "../types";
import { deleteModel, getBuild, getModels, setModel, uploadBuild } from "../api";

interface NoticeState {
  open: boolean;
  text: string;
  type: AlertColor;
}

export function ModelsPage() {
  const [models, setModels] = useState<Record<string, string>>({});
  const [name, setName] = useState("");
  const [exePath, setExePath] = useState("");
  const [uploading, setUploading] = useState(false);
  const [activeBuild, setActiveBuild] = useState<BuildInfo | null>(null);
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
      setModels(data.models || {});
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

  function statusLabel(status: BuildInfo["status"]) {
    if (status === "queued") {
      return "Queued";
    }
    if (status === "extracting_and_scanning") {
      return "Extracting and Scanning";
    }
    if (status === "ready") {
      return "Ready";
    }
    return "Failed";
  }

  function statusSeverity(status: BuildInfo["status"]): AlertColor {
    if (status === "failed") {
      return "error";
    }
    if (status === "ready") {
      return "success";
    }
    return "info";
  }

  async function onUploadBuild(file: File) {
    if (!file.name.toLowerCase().endsWith(".zip")) {
      notify("Only Windows build packages are supported in .ZIP format", "error");
      return;
    }
    const maxBytes = 4 * 1024 * 1024 * 1024;
    if (file.size > maxBytes) {
      notify("Maximum upload size is 4GB", "error");
      return;
    }

    setUploading(true);
    try {
      const created = await uploadBuild(file);
      setActiveBuild(created);
      notify("Build uploaded and queued");
    } catch (error) {
      notify((error as Error).message, "error");
    } finally {
      setUploading(false);
    }
  }

  useEffect(() => {
    if (!activeBuild || activeBuild.status === "ready" || activeBuild.status === "failed") {
      return;
    }

    const timer = setInterval(() => {
      void getBuild(activeBuild.id)
        .then((latest) => {
          setActiveBuild(latest);
        })
        .catch(() => undefined);
    }, 2000);

    return () => clearInterval(timer);
  }, [activeBuild]);

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
          <Typography variant="h6" sx={{ mb: 2 }}>
            Add or Update Model
          </Typography>
          <Stack spacing={1.5} sx={{ mb: 2.5 }}>
            <Alert severity="info">
              Only Windows build packages are supported in <strong>.ZIP</strong> format. Maximum size is{" "}
              <strong>4GB</strong>. Upload packaged build output, not project source files.
            </Alert>
            <Stack direction={{ xs: "column", sm: "row" }} spacing={1} alignItems={{ sm: "center" }}>
              <Button variant="outlined" component="label" disabled={uploading}>
                {uploading ? "Uploading..." : "Upload Build ZIP"}
                <input
                  hidden
                  type="file"
                  accept=".zip,application/zip"
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (file) {
                      void onUploadBuild(file);
                    }
                    event.target.value = "";
                  }}
                />
              </Button>
              <Typography variant="body2" color="text.secondary">
                Storage path: /builds/&lt;build_id&gt;/unzipped_processes
              </Typography>
            </Stack>
            {activeBuild && (
              <Alert severity={statusSeverity(activeBuild.status)}>
                <strong>{statusLabel(activeBuild.status)}</strong>: {activeBuild.message}
              </Alert>
            )}
            {activeBuild && activeBuild.status === "ready" && (
              <Card variant="outlined">
                <CardContent>
                  <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 1 }}>
                    Discovered Executables
                  </Typography>
                  <Stack spacing={1}>
                    {activeBuild.executables.map((exe) => (
                      <Stack
                        key={exe}
                        direction={{ xs: "column", md: "row" }}
                        spacing={1}
                        alignItems={{ md: "center" }}
                      >
                        <Typography
                          variant="body2"
                          color="text.secondary"
                          sx={{ fontFamily: "monospace", wordBreak: "break-all", flex: 1 }}
                        >
                          {exe}
                        </Typography>
                        <Button
                          size="small"
                          onClick={() => {
                            setExePath(exe);
                          }}
                        >
                          Use Path
                        </Button>
                      </Stack>
                    ))}
                  </Stack>
                </CardContent>
              </Card>
            )}
          </Stack>
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
