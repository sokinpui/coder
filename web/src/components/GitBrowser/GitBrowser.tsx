import { useState } from "react";
import {
  Box,
  List,
  Button,
  ListItem,
  ListItemButton,
  ListItemText,
  Typography,
  Divider,
  Chip,
  CircularProgress,
  IconButton,
  Tooltip,
} from "@mui/material";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import ViewDayIcon from "@mui/icons-material/ViewDay";
import VerticalSplitIcon from "@mui/icons-material/VerticalSplit";
import type { GitLogEntry } from "../../types";
import { SideBySideDiffViewer } from "../SideBySideDiffViewer";
import { UnifiedDiffViewer } from "../UnifiedDiffViewer";

interface GitBrowserProps {
  log: GitLogEntry[];
  getCommitDiff: (hash: string) => void;
  commitDiff: { hash: string; diff: string } | null;
}

export function GitBrowser({ log, getCommitDiff, commitDiff }: GitBrowserProps) {
  const [selectedCommit, setSelectedCommit] = useState<string | null>(null);
  const [diffView, setDiffView] = useState<"side-by-side" | "unified">("side-by-side");

  const handleCommitSelect = (hash: string) => {
    setSelectedCommit(hash);
    getCommitDiff(hash);
  };

  const handleBackToLog = () => {
    setSelectedCommit(null);
  };

  const handleToggleDiffView = () => {
    setDiffView((prev) => (prev === "side-by-side" ? "unified" : "side-by-side"));
  };

  if (log.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <Typography color="text.secondary">No git history found or still loading...</Typography>
      </Box>
    )
  }

  if (selectedCommit) {
    const selectedCommitData = log.find((entry) => entry.hash === selectedCommit);
    return (
      <Box sx={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }}>
        <Box sx={{ p: 1, borderBottom: 1, borderColor: 'divider', display: 'flex', alignItems: 'center', flexShrink: 0 }}>
          <Button startIcon={<ArrowBackIcon />} onClick={handleBackToLog} sx={{ mr: 2 }}>
            Commits
          </Button>
          {selectedCommitData && (
            <Box sx={{ flexGrow: 1 }}>
              <Typography variant="body1" sx={{ fontWeight: 'bold' }}>{selectedCommitData.subject}</Typography>
              <Typography variant="caption" color="text.secondary">
                {selectedCommitData.authorName}, {selectedCommitData.relativeDate}
              </Typography>
            </Box>
          )}
          <Tooltip title={diffView === "side-by-side" ? "Unified view" : "Side-by-side view"}>
            <IconButton onClick={handleToggleDiffView}>
              {diffView === "side-by-side" ? <ViewDayIcon /> : <VerticalSplitIcon />}
            </IconButton>
          </Tooltip>
        </Box>
        <Box sx={{ flexGrow: 1, overflow: 'hidden' }}>
          {commitDiff && commitDiff.hash === selectedCommit ? (
            diffView === "side-by-side" ? (
              <SideBySideDiffViewer diff={commitDiff.diff} />
            ) : (
              <UnifiedDiffViewer diff={commitDiff.diff} />
            )
          ) : (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
              <CircularProgress />
            </Box>
          )}
        </Box>
      </Box>
    );
  }

  return (
    <Box sx={{ height: "100%", overflowY: "auto" }}>
      <List>
        {log.map((entry) => (
          <div key={entry.hash}>
            <ListItemButton onClick={() => handleCommitSelect(entry.hash)}>
              <ListItemText
                primary={
                  <Box
                    sx={{
                      display: "flex",
                      alignItems: "center",
                      gap: 2,
                      flexWrap: "wrap",
                    }}
                  >
                    <Typography variant="body1" component="span" sx={{ flexShrink: 0 }}>
                      {entry.subject}
                    </Typography>
                    <Chip label={entry.hash.substring(0, 7)} size="small" variant="outlined" component="span" />
                  </Box>
                }
                secondary={`${entry.authorName}, ${entry.relativeDate}`}
                secondaryTypographyProps={{ sx: { mt: 0.5 } }}
              />
            </ListItemButton>
            <Divider component="li" />
          </div>
        ))}
      </List>
    </Box>
  );
}
