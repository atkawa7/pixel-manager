import DashboardOutlinedIcon from "@mui/icons-material/DashboardOutlined";
import HubOutlinedIcon from "@mui/icons-material/HubOutlined";
import Inventory2OutlinedIcon from "@mui/icons-material/Inventory2Outlined";
import DescriptionOutlinedIcon from "@mui/icons-material/DescriptionOutlined";
import SettingsOutlinedIcon from "@mui/icons-material/SettingsOutlined";
import {
  Box,
  Divider,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
} from "@mui/material";
import type { ReactNode } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";

const drawerWidth = 260;

interface NavItem {
  label: string;
  path: string;
  icon: ReactNode;
}

const navItems: NavItem[] = [
  { label: "Portal", path: "/portal", icon: <DashboardOutlinedIcon /> },
  { label: "Managers", path: "/managers", icon: <HubOutlinedIcon /> },
  { label: "Models", path: "/models", icon: <Inventory2OutlinedIcon /> },
  { label: "Settings", path: "/settings", icon: <SettingsOutlinedIcon /> },
];

export function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();

  return (
    <Box sx={{ display: "flex", minHeight: "100vh" }}>
      <Drawer
        variant="permanent"
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          "& .MuiDrawer-paper": {
            width: drawerWidth,
            boxSizing: "border-box",
            background: "#155e63",
            color: "#f8fbfc",
            border: "none",
          },
        }}
      >
        <Toolbar sx={{ px: 3, py: 2, alignItems: "flex-start" }}>
          <Box>
            <Typography variant="overline" sx={{ letterSpacing: 1.6, opacity: 0.8 }}>
              Control Plane
            </Typography>
            <Typography variant="h6" sx={{ fontWeight: 700 }}>
              Pixel Manager
            </Typography>
          </Box>
        </Toolbar>
        <Divider sx={{ borderColor: "rgba(255,255,255,0.14)" }} />
        <List sx={{ mt: 1, px: 1.5 }}>
          {navItems.map((item) => {
            const selected = location.pathname === item.path;
            return (
              <ListItemButton
                key={item.path}
                selected={selected}
                onClick={() => navigate(item.path)}
                sx={{
                  borderRadius: 2,
                  mb: 0.5,
                  color: "rgba(248,251,252,0.9)",
                  "&.Mui-selected": {
                    backgroundColor: "rgba(255,255,255,0.2)",
                  },
                  "&.Mui-selected:hover": {
                    backgroundColor: "rgba(255,255,255,0.26)",
                  },
                }}
              >
                <ListItemIcon sx={{ color: "inherit", minWidth: 36 }}>{item.icon}</ListItemIcon>
                <ListItemText primary={item.label} />
              </ListItemButton>
            );
          })}
        </List>
        <Box sx={{ mt: "auto", px: 2, pb: 3 }}>
          <ListItemButton
            component="a"
            href="/docs"
            sx={{
              borderRadius: 2,
              color: "rgba(248,251,252,0.9)",
              backgroundColor: "rgba(255,255,255,0.08)",
            }}
          >
            <ListItemIcon sx={{ color: "inherit", minWidth: 36 }}>
              <DescriptionOutlinedIcon />
            </ListItemIcon>
            <ListItemText primary="API Docs" />
          </ListItemButton>
        </Box>
      </Drawer>
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          px: { xs: 2, sm: 3, md: 4 },
          py: 3,
          background: "#f3f7f8",
        }}
      >
        <Outlet />
      </Box>
    </Box>
  );
}
