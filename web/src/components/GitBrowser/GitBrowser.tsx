import { useState, type MouseEvent, useRef, useCallback, memo } from "react";
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
  Popper,
  Fade,
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
import { HighlightMenu } from "../HighlightMenu";

interface GitBrowserProps {
  log: GitGraphLogEntry[];
  commitDiff: { hash: string; diff: string } | null;
  onAskAI: (text: string) => void;
}

function GitBrowserComponent({ log, commitDiff, onAskAI }: GitBrowserProps) {
  const { '*': selectedCommit } = useParams();
  const navigate = useNavigate();
  const [view, setView] = useState<'graph' | 'list'>('list');
  const [diffView, setDiffView] = useState<"side-by-side" | "unified">("side-by-side");

  const menuRef = useRef<HTMLDivElement>(null);
  const [highlightMenuState, setHighlightMenuState] = useState<{
    open: boolean;
    anchorEl: { getBoundingClientRect: () => DOMRect } | null;
    selectedText: string;
  }>({
    open: false,
    anchorEl: null,
    selectedText: "",
  });

  const handleCloseHighlightMenu = useCallback(() => {
    setHighlightMenuState((prev) => ({ ...prev, open: false }));
  }, []);

  const handleMouseUp = (event: React.MouseEvent) => {
    if (menuRef.current && menuRef.current.contains(event.target as Node)) {
      return;
    }

    setTimeout(() => {
      const selection = window.getSelection();
      if (selection && selection.toString().trim().length > 0) {
        const range = selection.getRangeAt(0);

        const virtualEl = {
          getBoundingClientRect: () => range.getBoundingClientRect(),
        };

        setHighlightMenuState({
          open: true,
          anchorEl: virtualEl,
          selectedText: selection.toString(),
        });
      } else {
        handleCloseHighlightMenu();
      }
    }, 10);
  };

  const handleCopySuccess = () => {
    setTimeout(() => {
      handleCloseHighlightMenu();
      window.getSelection()?.removeAllRanges();
    }, 500);
  };

  const handleAskAI = (text: string) => {
    onAskAI(text);
    handleCloseHighlightMenu();
  };

  // Note: The scroll-to-close behavior is not implemented here due to multiple
  // independent scrolling containers. The menu will still close on click-away
  // or when the selection is cleared.

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
        <Box sx={{ display: "flex", flexDirection: "column", height: "100%", overflow: "hidden" }} onMouseUp={handleMouseUp}>
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
        <Popper
          open={highlightMenuState.open}
          anchorEl={highlightMenuState.anchorEl}
          placement="top"
          transition
          sx={{ zIndex: 1300 }}
        >
          {({ TransitionProps }) => (
            <Fade {...TransitionProps} timeout={150}>
              <div ref={menuRef}>
                <HighlightMenu
                  selectedText={highlightMenuState.selectedText}
                  onCopySuccess={handleCopySuccess}
                  onAskAI={handleAskAI}
                />
              </div>
            </Fade>
          )}
        </Popper>
      </>
    );
  }

  return (
    <>
      <Box sx={{ height: "100%", display: 'flex', flexDirection: 'column' }} onMouseUp={handleMouseUp}>
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
      <Popper
        open={highlightMenuState.open}
        anchorEl={highlightMenuState.anchorEl}
        placement="top"
        transition
        sx={{ zIndex: 1300 }}
      >
        {({ TransitionProps }) => (
          <Fade {...TransitionProps} timeout={150}>
            <div ref={menuRef}>
              <HighlightMenu
                selectedText={highlightMenuState.selectedText}
                onCopySuccess={handleCopySuccess}
                onAskAI={handleAskAI}
              />
            </div>
          </Fade>
        )}
      </Popper>
    </>
  );
}

export const GitBrowser = memo(GitBrowserComponent);
