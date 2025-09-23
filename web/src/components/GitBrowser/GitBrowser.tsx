import { useState, type MouseEvent, memo } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Box,
  List,
  Button,
  ListItemButton,
  ListItemText,
  Typography,
  Divider,
  Chip,
  CircularProgress,
  IconButton,
  Tooltip,
  ToggleButtonGroup,
  ToggleButton,
} from "@mui/material";
import ArrowBackIcon from "@mui/icons-material/ArrowBack";
import ViewDayIcon from "@mui/icons-material/ViewDay";
import VerticalSplitIcon from "@mui/icons-material/VerticalSplit";
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import ListIcon from '@mui/icons-material/List';
import type { GitGraphLogEntry } from "../../types";
import { SideBySideDiffViewer } from "../SideBySideDiffViewer";
import { UnifiedDiffViewer } from "../UnifiedDiffViewer";
import { GitGraph } from "../GitGraph";

interface GitBrowserProps {
  log: GitGraphLogEntry[];
  commitDiff: { hash: string; diff: string } | null;
}

function GitBrowserComponent({ log, commitDiff }: GitBrowserProps) {
  const { '*': selectedCommit } = useParams();
  const navigate = useNavigate();
  const [view, setView] = useState<'graph' | 'list'>('list');
  const [diffView, setDiffView] = useState<"side-by-side" | "unified">("side-by-side");

  const handleCommitSelect = (hash: string) => {
    navigate(`/git/${hash}`);
  };

  const handleBackToLog = () => {
    navigate('/git');
  };

  const handleToggleDiffView = () => {
    setDiffView((prev) => (prev === "side-by-side" ? "unified" : "side-by-side"));
  };

  const handleViewChange = (_event: MouseEvent<HTMLElement>, newView: 'graph' | 'list' | null) => {
    if (newView !== null) {
      setView(newView);
    }
  };

  if (log.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <Typography color="text.secondary">No git history found or still loading...</Typography>
      </Box>
    )
  }

  if (selectedCommit) {
    const selectedCommitData = log.find((entry) => entry.hash.startsWith(selectedCommit));
    return (
      <>
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
            <Tooltip title={diffView === "side-by-side" ? "Unified view" : "Side-by-side view"} enterDelay={1000}>
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
      </>
    );
  }

  return (
    <>
      <Box sx={{ height: "100%", display: 'flex', flexDirection: 'column' }}>
        <Box sx={{ p: 1, borderBottom: 1, borderColor: 'divider', display: 'flex', justifyContent: 'flex-end' }}>
          <ToggleButtonGroup value={view} exclusive onChange={handleViewChange} size="small">
            <Tooltip title="Graph view" enterDelay={1000}>
              <ToggleButton value="graph" aria-label="graph view">
                <AccountTreeIcon />
              </ToggleButton>
            </Tooltip>
            <Tooltip title="List view" enterDelay={1000}>
              <ToggleButton value="list" aria-label="list view">
                <ListIcon />
              </ToggleButton>
            </Tooltip>
          </ToggleButtonGroup>
        </Box>
        <Box sx={{ flexGrow: 1, overflow: "hidden" }}>
          {view === 'graph' ? (
            <GitGraph log={log} onCommitSelect={handleCommitSelect} />
          ) : (
            <List sx={{ height: '100%', overflowY: 'auto' }}>
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
                          {entry.refs.map(ref => (
                            <Chip key={ref} label={ref} size="small" color="secondary" component="span" />
                          ))}
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
          )}
        </Box>
      </Box>
    </>
  );
}

export const GitBrowser = memo(GitBrowserComponent);
