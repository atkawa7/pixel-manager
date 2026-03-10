import React from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { CssBaseline, ThemeProvider, createTheme } from "@mui/material";
import { AppLayout } from "./ui/AppLayout";
import { PortalPage } from "./pages/PortalPage";
import { ManagersPage } from "./pages/ManagersPage";
import { ModelsPage } from "./pages/ModelsPage";
import "./styles.css";

const theme = createTheme({
  palette: {
    mode: "light",
    primary: {
      main: "#155e63",
    },
    secondary: {
      main: "#b45309",
    },
    background: {
      default: "#f3f7f8",
      paper: "#ffffff",
    },
  },
  shape: {
    borderRadius: 12,
  },
  typography: {
    fontFamily: '"IBM Plex Sans", "Segoe UI", sans-serif',
  },
});

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <BrowserRouter>
        <Routes>
          <Route element={<AppLayout />}>
            <Route path="/" element={<Navigate to="/portal.html" replace />} />
            <Route path="/portal.html" element={<PortalPage />} />
            <Route path="/managers.html" element={<ManagersPage />} />
            <Route path="/models.html" element={<ModelsPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}

createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
