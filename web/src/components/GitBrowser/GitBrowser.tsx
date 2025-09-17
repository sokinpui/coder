import {
  Box,
  List,
  ListItem,
  ListItemText,
  Typography,
  Divider,
  Chip,
} from "@mui/material";
import type { GitLogEntry } from "../../types";

interface GitBrowserProps {
  log: GitLogEntry[];
}

export function GitBrowser({ log }: GitBrowserProps) {
  if (log.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <Typography color="text.secondary">No git history found or still loading...</Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ overflowY: "auto", height: "100%" }}>
      <List>
        {log.map((entry, index) => (
          <div key={entry.hash}>
            <ListItem>
              <ListItemText
                primary={
                  <Box sx={{ display: "flex", alignItems: "center", gap: 2, flexWrap: 'wrap' }}>
                    <Typography variant="body1" component="span" sx={{ flexShrink: 0 }}>
                      {entry.subject}
                    </Typography>
                    <Chip
                      label={entry.hash.substring(0, 7)}
                      size="small"
                      variant="outlined"
                      component="span"
                    />
                  </Box>
                }
                secondary={`${entry.authorName}, ${entry.relativeDate}`}
                secondaryTypographyProps={{ sx: { mt: 0.5 } }}
              />
            </ListItem>
            {index < log.length - 1 && <Divider component="li" />}
          </div>
        ))}
      </List>
    </Box>
  );
}
