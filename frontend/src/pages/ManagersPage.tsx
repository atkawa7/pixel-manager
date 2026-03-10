import { useEffect, useState } from "react";
import RefreshRoundedIcon from "@mui/icons-material/RefreshRounded";
import OpenInNewRoundedIcon from "@mui/icons-material/OpenInNewRounded";
import InfoOutlinedIcon from "@mui/icons-material/InfoOutlined";
import {
  Alert,
  Box,
  Button,
  Card,
  CardActions,
  CardContent,
  Chip,
  Dialog,
  DialogContent,
  DialogTitle,
  Grid,
  LinearProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { getInstances, getManagers, getModels } from "../api";
import type { Instance } from "../types";

interface ManagerNode {
  host: string;
  name: string;
  url: string;
}

interface DetailsState {
  manager: ManagerNode;
  maxInstances: number;
  models: Record<string, string>;
  hostInstances: Instance[];
}

async function checkManagerHealth(url: string): Promise<boolean> {
  try {
    const response = await fetch(`${url}/instances`, { signal: AbortSignal.timeout(3000) });
    return response.ok;
  } catch {
    return false;
  }
}

export function ManagersPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [managers, setManagers] = useState<ManagerNode[]>([]);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [healthMap, setHealthMap] = useState<Record<string, boolean>>({});
  const [details, setDetails] = useState<DetailsState | null>(null);

  async function loadManagers() {
    setError("");
    setLoading(true);
    try {
      const [managersData, instancesData] = await Promise.all([getManagers(), getInstances()]);
      const nextManagers = managersData.managers || [];
      setManagers(nextManagers);
      setInstances(instancesData.active || []);

      const checks = await Promise.all(
        nextManagers.map(async (manager) => ({
          host: manager.host,
          healthy: await checkManagerHealth(manager.url),
        })),
      );
      const nextHealth: Record<string, boolean> = {};
      for (const item of checks) {
        nextHealth[item.host] = item.healthy;
      }
      setHealthMap(nextHealth);
    } catch (loadError) {
      setError((loadError as Error).message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadManagers();
    const interval = setInterval(() => {
      void loadManagers();
    }, 10000);
    return () => clearInterval(interval);
  }, []);

  async function openDetails(manager: ManagerNode) {
    try {
      const [instancesData, modelsData] = await Promise.all([
        getInstances(manager.url),
        getModels(manager.url),
      ]);
      const hostInstances = (instancesData.active || []).filter(
        (item) => item.host === manager.host,
      );
      setDetails({
        manager,
        maxInstances: instancesData.maxInstances,
        models: modelsData.models || {},
        hostInstances,
      });
    } catch (detailError) {
      setError((detailError as Error).message);
    }
  }

  const totalManagers = managers.length;
  const activeManagers = Object.values(healthMap).filter(Boolean).length;
  const totalInstances = instances.length;
  const currentHost = window.location.hostname;

  return (
    <Stack spacing={2.5}>
      <Card elevation={0} sx={{ background: "#b45309", color: "#fff" }}>
        <CardContent>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Cluster Managers
          </Typography>
          <Typography sx={{ opacity: 0.95 }}>
            Node status, availability, and per-manager instance distribution.
          </Typography>
        </CardContent>
      </Card>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="overline">Total Managers</Typography>
              <Typography variant="h4" sx={{ fontWeight: 700 }}>
                {totalManagers}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="overline">Online</Typography>
              <Typography variant="h4" sx={{ fontWeight: 700 }}>
                {activeManagers}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="overline">Total Instances</Typography>
              <Typography variant="h4" sx={{ fontWeight: 700 }}>
                {totalInstances}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card>
        <CardContent>
          <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
            <Typography variant="h6">Active Managers</Typography>
            <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={() => void loadManagers()}>
              Refresh
            </Button>
          </Box>
          {loading && <LinearProgress sx={{ mb: 2 }} />}
          {error && <Alert severity="error">{error}</Alert>}
          <Grid container spacing={2}>
            {managers.map((manager) => {
              const healthy = Boolean(healthMap[manager.host]);
              const isCurrent =
                manager.host === currentHost || String(manager.url || "").includes(currentHost);
              const hostInstances = instances.filter((item) => item.host === manager.host).length;
              return (
                <Grid key={manager.host} size={{ xs: 12, md: 6, lg: 4 }}>
                  <Card variant="outlined" sx={{ borderWidth: isCurrent ? 2 : 1 }}>
                    <CardContent>
                      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1.5 }}>
                        <Typography variant="h6">{manager.name || manager.host}</Typography>
                        <Chip
                          size="small"
                          label={healthy ? "Online" : "Offline"}
                          color={healthy ? "success" : "default"}
                        />
                      </Stack>
                      <Stack direction="row" spacing={1} sx={{ mb: 1.5 }}>
                        {isCurrent && <Chip size="small" color="primary" label="Current Node" />}
                        <Chip size="small" label={`${hostInstances} instance(s)`} />
                      </Stack>
                      <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                        Host: {manager.host}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ wordBreak: "break-all" }}>
                        {manager.url}
                      </Typography>
                    </CardContent>
                    <CardActions sx={{ px: 2, pb: 2 }}>
                      <Button
                        size="small"
                        startIcon={<InfoOutlinedIcon />}
                        onClick={() => void openDetails(manager)}
                      >
                        Details
                      </Button>
                      {!isCurrent && (
                        <Button
                          size="small"
                          component="a"
                          href={manager.url}
                          target="_blank"
                          rel="noreferrer"
                          startIcon={<OpenInNewRoundedIcon />}
                        >
                          Open
                        </Button>
                      )}
                    </CardActions>
                  </Card>
                </Grid>
              );
            })}
            {!loading && managers.length === 0 && (
              <Grid size={12}>
                <Alert severity="warning">No managers registered in the cluster.</Alert>
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>

      <Dialog open={Boolean(details)} onClose={() => setDetails(null)} fullWidth maxWidth="lg">
        <DialogTitle>Manager Details</DialogTitle>
        <DialogContent>
          {details && (
            <Stack spacing={2}>
              <Card variant="outlined">
                <CardContent>
                  <Typography variant="subtitle2" color="text.secondary">
                    Manager
                  </Typography>
                  <Typography variant="h6">
                    {details.manager.name || details.manager.host}
                  </Typography>
                  <Typography variant="body2">
                    Host: {details.manager.host}
                  </Typography>
                  <Typography variant="body2" sx={{ mt: 1 }}>
                    URL: {details.manager.url}
                  </Typography>
                  <Typography variant="body2">
                    Active Instances: {details.hostInstances.length} / {details.maxInstances}
                  </Typography>
                </CardContent>
              </Card>

              <Card variant="outlined">
                <CardContent>
                  <Typography variant="h6" sx={{ mb: 1 }}>
                    Models
                  </Typography>
                  <Stack direction="row" flexWrap="wrap" gap={1}>
                    {Object.keys(details.models).map((model) => (
                      <Chip key={model} label={model} color={model === "default" ? "primary" : "default"} />
                    ))}
                    {Object.keys(details.models).length === 0 && (
                      <Typography color="text.secondary">No models available.</Typography>
                    )}
                  </Stack>
                </CardContent>
              </Card>

              <Card variant="outlined">
                <CardContent>
                  <Typography variant="h6" sx={{ mb: 1 }}>
                    Running Instances
                  </Typography>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>ID</TableCell>
                        <TableCell>Model</TableCell>
                        <TableCell>Port</TableCell>
                        <TableCell>Started</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {details.hostInstances.map((instance) => (
                        <TableRow key={instance.pixelStreamingId}>
                          <TableCell sx={{ fontFamily: "monospace" }}>
                            {instance.pixelStreamingId.slice(0, 8)}...
                          </TableCell>
                          <TableCell>{instance.model}</TableCell>
                          <TableCell>{instance.port}</TableCell>
                          <TableCell>{new Date(instance.startTime).toLocaleString()}</TableCell>
                        </TableRow>
                      ))}
                      {details.hostInstances.length === 0 && (
                        <TableRow>
                          <TableCell colSpan={4}>
                            <Typography color="text.secondary">No instances running.</Typography>
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </Stack>
          )}
        </DialogContent>
      </Dialog>
    </Stack>
  );
}
