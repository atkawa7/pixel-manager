import { useEffect, useState } from "react";
import RefreshRoundedIcon from "@mui/icons-material/RefreshRounded";
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { getAppConfig } from "../api";

type ConfigValue = string | number | boolean | null;

export function SettingsPage() {
  const [configPath, setConfigPath] = useState("");
  const [values, setValues] = useState<Record<string, ConfigValue>>({});
  const [error, setError] = useState("");

  async function load() {
    setError("");
    try {
      const data = await getAppConfig();
      setConfigPath(data.configPath || "");
      setValues(data.config || {});
    } catch (loadError) {
      setError((loadError as Error).message);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  const entries = Object.entries(values).sort(([a], [b]) => a.localeCompare(b));

  return (
    <Stack spacing={2.5}>
      <Card elevation={0} sx={{ background: "linear-gradient(120deg, #334155, #1e293b)", color: "#fff" }}>
        <CardContent>
          <Typography variant="h4" sx={{ fontWeight: 700 }}>
            Settings
          </Typography>
          <Typography sx={{ opacity: 0.95 }}>
            Effective runtime config loaded by the backend process.
          </Typography>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
            <Typography variant="h6">Loaded Configuration</Typography>
            <Button variant="outlined" startIcon={<RefreshRoundedIcon />} onClick={() => void load()}>
              Refresh
            </Button>
          </Box>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Config File: <strong>{configPath || "n/a"}</strong>
          </Typography>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Key</TableCell>
                <TableCell>Value</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {entries.map(([key, value]) => (
                <TableRow key={key}>
                  <TableCell sx={{ fontFamily: "monospace" }}>{key}</TableCell>
                  <TableCell sx={{ fontFamily: "monospace" }}>{String(value)}</TableCell>
                </TableRow>
              ))}
              {entries.length === 0 && (
                <TableRow>
                  <TableCell colSpan={2}>
                    <Typography color="text.secondary">No config values available.</Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </Stack>
  );
}
