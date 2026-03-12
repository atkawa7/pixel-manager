import { useEffect, useMemo, useState } from "react";
import AddCircleOutlineRoundedIcon from "@mui/icons-material/AddCircleOutlineRounded";
import RefreshRoundedIcon from "@mui/icons-material/RefreshRounded";
import UploadFileRoundedIcon from "@mui/icons-material/UploadFileRounded";
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
  Pagination,
  Snackbar,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import type { AlertColor } from "@mui/material";
import type { BuildInfo } from "../types";
import {
  getBuild,
  getBuildExecutables,
  listBuilds,
  setModel,
  uploadBuild,
  uploadBuildFromURL,
} from "../api";

interface NoticeState {
  open: boolean;
  text: string;
  type: AlertColor;
}

export function BuildsPage() {
  const [builds, setBuilds] = useState<BuildInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [notice, setNotice] = useState<NoticeState>({
    open: false,
    text: "",
    type: "success",
  });
  const [creatingFrom, setCreatingFrom] = useState<BuildInfo | null>(null);
  const [modelName, setModelName] = useState("");
  const [modelPath, setModelPath] = useState("");
  const [sourceURL, setSourceURL] = useState("");

  function notify(text: string, type: AlertColor = "success") {
    setNotice({ open: true, text, type });
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

  async function loadBuilds() {
    setLoading(true);
    try {
      const data = await listBuilds(page, pageSize);
      setBuilds(data.builds || []);
      setTotal(data.total || 0);
      setTotalPages(data.totalPages || 0);
      if (data.page && data.page !== page) {
        setPage(data.page);
      }
    } catch (error) {
      notify((error as Error).message, "error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadBuilds();
  }, [page, pageSize]);

  const activeBuildIDs = useMemo(
    () =>
      builds
        .filter((b) => b.status === "queued" || b.status === "extracting_and_scanning")
        .map((b) => b.id),
    [builds],
  );

  useEffect(() => {
    if (activeBuildIDs.length === 0) {
      return;
    }

    const timer = setInterval(() => {
      activeBuildIDs.forEach((id) => {
        void getBuild(id)
          .then((latest) => {
            setBuilds((prev) => prev.map((item) => (item.id === latest.id ? latest : item)));
          })
          .catch(() => undefined);
      });
    }, 2000);

    return () => clearInterval(timer);
  }, [activeBuildIDs]);

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
      await uploadBuild(file);
      if (page !== 1) {
        setPage(1);
      } else {
        void loadBuilds();
      }
      notify("Build uploaded and queued");
    } catch (error) {
      notify((error as Error).message, "error");
    } finally {
      setUploading(false);
    }
  }

  async function onImportFromURL() {
    const trimmedURL = sourceURL.trim();
    if (!trimmedURL) {
      notify("Public link is required", "error");
      return;
    }
    if (!/^https?:\/\//i.test(trimmedURL)) {
      notify("Public link must start with http:// or https://", "error");
      return;
    }

    setUploading(true);
    try {
      await uploadBuildFromURL(trimmedURL);
      setSourceURL("");
      if (page !== 1) {
        setPage(1);
      } else {
        void loadBuilds();
      }
      notify("Build imported and queued");
    } catch (error) {
      notify((error as Error).message, "error");
    } finally {
      setUploading(false);
    }
  }

  async function openCreateModel(build: BuildInfo, preferredPath: string) {
    let path = preferredPath;
    if (!path) {
      try {
        const data = await getBuildExecutables(build.id);
        setBuilds((prev) =>
          prev.map((item) =>
            item.id === build.id ? { ...item, executables: data.executables || [] } : item,
          ),
        );
        path = data.executables?.[0] || "";
      } catch (error) {
        notify((error as Error).message, "error");
        return;
      }
    }

    if (!path) {
      notify("No executable found for this build", "error");
      return;
    }

    const baseName = build.fileName.replace(/\.zip$/i, "").trim();
    setCreatingFrom(build);
    setModelName(baseName || "new-model");
    setModelPath(path);
  }

  async function onCreateModel() {
    if (!creatingFrom) {
      return;
    }
    if (!modelName.trim() || !modelPath.trim()) {
      notify("Model name and executable path are required", "error");
      return;
    }
    try {
      await setModel(modelName.trim(), modelPath.trim());
      notify("Model created from build");
      setCreatingFrom(null);
      setModelName("");
      setModelPath("");
    } catch (error) {
      notify((error as Error).message, "error");
    }
  }

  return (
    <Stack spacing={2.5}>
      <Card elevation={0} sx={{ background: "#0f766e", color: "#fff" }}>
        <CardContent>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Builds
          </Typography>
          <Typography sx={{ opacity: 0.95 }}>
            Upload Windows build ZIP packages, monitor processing, and create models from extracted executables.
          </Typography>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Stack spacing={1.5}>
            <Alert severity="info">
              Only Windows build packages are supported in <strong>.ZIP</strong> format. Maximum file size is{" "}
              <strong>4GB</strong>. Upload packaged build outputs, not project source files.
            </Alert>
            <Stack direction={{ xs: "column", sm: "row" }} spacing={1} alignItems={{ sm: "center" }}>
              <Button
                variant="contained"
                startIcon={<UploadFileRoundedIcon />}
                component="label"
                disabled={uploading}
              >
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
            <Stack direction={{ xs: "column", md: "row" }} spacing={1}>
              <TextField
                fullWidth
                label="Import from public link"
                placeholder="https://..."
                value={sourceURL}
                onChange={(event) => setSourceURL(event.target.value)}
                disabled={uploading}
                helperText="Supports public Dropbox, OneDrive, Google Drive, and WeShare links."
              />
              <Button
                variant="outlined"
                onClick={() => void onImportFromURL()}
                disabled={uploading}
                sx={{ minWidth: { md: 180 } }}
              >
                {uploading ? "Importing..." : "Import Link"}
              </Button>
            </Stack>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
            <Typography variant="h6">Uploaded Builds</Typography>
            <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={() => void loadBuilds()}>
              {loading ? "Loading..." : "Refresh"}
            </Button>
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Showing page {totalPages === 0 ? 0 : page} of {totalPages} ({total} total builds)
          </Typography>
          <Grid container spacing={2}>
            {builds.map((build) => (
              <Grid key={build.id} size={{ xs: 12 }}>
                <Card variant="outlined">
                  <CardContent>
                    <Stack spacing={1.1}>
                      <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
                        {build.fileName}
                      </Typography>
                      <Alert severity={statusSeverity(build.status)}>
                        <strong>{statusLabel(build.status)}</strong>: {build.message}
                      </Alert>
                      <Typography variant="body2" color="text.secondary">
                        Build ID: <code>{build.id}</code>
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        Extracted: <code>{build.extractedDir}</code>
                      </Typography>

                      {(build.executables || []).map((exe) => (
                        <Stack
                          key={`${build.id}-${exe}`}
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
                            startIcon={<AddCircleOutlineRoundedIcon />}
                            onClick={() => void openCreateModel(build, exe)}
                          >
                            Create Model From
                          </Button>
                        </Stack>
                      ))}

                      {build.status === "ready" && (build.executables || []).length === 0 && (
                        <Button size="small" onClick={() => void openCreateModel(build, "")}>
                          Load Executables
                        </Button>
                      )}
                    </Stack>
                  </CardContent>
                </Card>
              </Grid>
            ))}
            {builds.length === 0 && (
              <Grid size={12}>
                <Alert severity="info">No builds uploaded yet.</Alert>
              </Grid>
            )}
          </Grid>
          {totalPages > 1 && (
            <Box sx={{ mt: 2, display: "flex", justifyContent: "center" }}>
              <Pagination
                page={page}
                count={totalPages}
                color="primary"
                onChange={(_, nextPage) => setPage(nextPage)}
              />
            </Box>
          )}
        </CardContent>
      </Card>

      <Dialog open={Boolean(creatingFrom)} onClose={() => setCreatingFrom(null)} maxWidth="sm" fullWidth>
        <DialogTitle>Create Model From Build</DialogTitle>
        <DialogContent>
          <Stack spacing={1.5} sx={{ mt: 0.5 }}>
            <TextField
              fullWidth
              label="Model Name"
              value={modelName}
              onChange={(event) => setModelName(event.target.value)}
            />
            <TextField
              fullWidth
              label="Executable Path"
              value={modelPath}
              onChange={(event) => setModelPath(event.target.value)}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreatingFrom(null)}>Cancel</Button>
          <Button variant="contained" onClick={() => void onCreateModel()}>
            Save Model
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
