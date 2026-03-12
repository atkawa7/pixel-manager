import { Box, Typography } from "@mui/material";
import SwaggerUI from "swagger-ui-react";
import "swagger-ui-react/swagger-ui.css";
import { getOpenAPIURL } from "../api";

export function ApiDocsPage() {
  const specURL = getOpenAPIURL();

  return (
    <Box>
      <Typography variant="h4" sx={{ fontWeight: 700, mb: 1 }}>
        API Docs
      </Typography>
      <Typography color="text.secondary" sx={{ mb: 2 }}>
        OpenAPI spec source: <code>{specURL}</code>
      </Typography>
      <Box
        sx={{
          backgroundColor: "#fff",
          borderRadius: 2,
          border: "1px solid rgba(8,30,44,0.08)",
          overflow: "hidden",
          "& .swagger-ui .topbar": {
            display: "none",
          },
        }}
      >
        <SwaggerUI
          url={specURL}
          defaultModelsExpandDepth={2}
          docExpansion="list"
          displayRequestDuration
        />
      </Box>
    </Box>
  );
}
