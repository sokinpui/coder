import { useState } from "react";
import {
  Box,
  List,
  ListItem,
  ListItemText,
  Typography,
  Divider,
  Chip,
  IconButton,
  Collapse,
} from "@mui/material";
import { MoreHoriz as MoreHorizIcon } from "@mui/icons-material";
import type { GitLogEntry } from "../../types";

interface GitBrowserProps {
  log: GitLogEntry[];
}

export function GitBrowser({ log }: GitBrowserProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggleExpand = (hash: string) => {
    setExpanded((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(hash)) {
        newSet.delete(hash);
      } else {
        newSet.add(hash);
      }
      return newSet;
    });
  };

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
                    {entry.body && (
                      <IconButton
                        size="small"
                        onClick={() => toggleExpand(entry.hash)}
                      >
                        <MoreHorizIcon fontSize="small" />
                      </IconButton>
                    )}
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
            {entry.body && (
              <Collapse in={expanded.has(entry.hash)} timeout="auto" unmountOnExit>
                <Box sx={{ pl: 2, pr: 2, pb: 1 }}>
                  <Typography
                    variant="body2"
                    component="pre"
                    sx={{
                      whiteSpace: "pre-wrap",
                      wordBreak: "break-word",
                      fontFamily: "monospace",
                      bgcolor: "action.hover",
                      p: 1.5,
                      borderRadius: 1,
                    }}
                  >
                    {entry.body}
                  </Typography>
                </Box>
              </Collapse>
            )}
            {index < log.length - 1 && <Divider component="li" />}
          </div>
        ))}
      </List>
    </Box>
  );
}
