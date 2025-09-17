import {
  Box,
  Typography,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Collapse,
  CircularProgress,
  IconButton,
  Tooltip,
  useTheme,
} from "@mui/material";
import {
  Folder as FolderIcon,
  Description as FileIcon,
  ExpandLess,
  ExpandMore,
  ChevronLeft,
  ChevronRight,
  UnfoldMore as UnfoldMoreIcon,
  UnfoldLess as UnfoldLessIcon,
} from "@mui/icons-material";
import {
  useContext,
  useState,
  useRef,
  useCallback,
  useEffect,
  memo,
} from "react";
import type { SourceNode } from "../../types";
import { AppContext } from "../../AppContext";
import { CopyButton } from "../CopyButton";
import CodeMirror, { type ReactCodeMirrorProps } from "@uiw/react-codemirror";
import { EditorView, lineNumbers } from "@codemirror/view";
import { oneDark } from "@codemirror/theme-one-dark";
import { githubLight } from "@uiw/codemirror-theme-github";
import { go } from "@codemirror/lang-go";
import { javascript } from "@codemirror/lang-javascript";
import { json } from "@codemirror/lang-json";
import { markdown } from "@codemirror/lang-markdown";
import { html } from "@codemirror/lang-html";
import { css } from "@codemirror/lang-css";

interface SourceBrowserProps {
  tree: SourceNode | null;
  activeFile: { path: string; content: string } | null;
  onFileSelect: (path: string) => void;
  showLineNumbers: boolean;
  cwd: string;
}

interface TreeNodeProps {
  node: SourceNode;
  onFileSelect: (path: string) => void;
  level: number;
  expandedNodes: Set<string>;
  onToggleNode: (path: string) => void;
}

const TreeNode = memo(function TreeNode({
  node,
  onFileSelect,
  level,
  expandedNodes,
  onToggleNode,
}: TreeNodeProps) {
  const isExpanded = expandedNodes.has(node.path);

  const handleClick = () => {
    if (node.type === "directory") {
      onToggleNode(node.path);
    } else {
      onFileSelect(node.path);
    }
  };

  return (
    <>
      <ListItemButton onClick={handleClick} sx={{ pl: 2 + level * 2 }}>
        <ListItemIcon sx={{ minWidth: 32 }}>
          {node.type === "directory" ? (
            <FolderIcon fontSize="small" />
          ) : (
            <FileIcon fontSize="small" />
          )}
        </ListItemIcon>
        <ListItemText
          primary={node.name}
          primaryTypographyProps={{ variant: "caption", noWrap: true }}
        />
        {node.type === "directory" && (isExpanded ? <ExpandLess /> : <ExpandMore />)}
      </ListItemButton>
      {node.type === "directory" && (
        <Collapse in={isExpanded} timeout="auto" unmountOnExit>
          <List component="div" disablePadding>
            {node.children?.map((child) => (
              <TreeNode
                key={child.path}
                node={child}
                onFileSelect={onFileSelect}
                level={level + 1}
                expandedNodes={expandedNodes}
                onToggleNode={onToggleNode}
              />
            ))}
          </List>
        </Collapse>
      )}
    </>
  );
});

const getLanguageExtension = (filePath: string) => {
  const extension = filePath.split(".").pop() || "";
  switch (extension) {
    case "go":
      return go();
    case "js":
    case "jsx":
    case "ts":
    case "tsx":
      return javascript({ jsx: true, typescript: true });
    case "json":
      return json();
    case "md":
      return markdown();
    case "html":
      return html();
    case "css":
      return css();
    default:
      return undefined;
  }
};

// Custom theme to make the editor look like a static viewer
const readOnlyTheme = EditorView.theme({
  // Hide the cursor
  ".cm-cursor, .cm-dropCursor": { border: "none" },
  // Make selection invisible
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection":
    {
      backgroundColor: "transparent !important",
    },
  // Remove focus ring
  "&.cm-focused": { outline: "none" },
  ".cm-content": {
    fontSize: "0.9rem",
  },
});

export function SourceBrowser({
  tree,
  activeFile,
  onFileSelect,
  showLineNumbers,
  cwd,
}: SourceBrowserProps) {
  const { codeTheme } = useContext(AppContext);
  const muiTheme = useTheme();
  const [treeWidth, setTreeWidth] = useState(300);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [isResizing, setIsResizing] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  const handleToggleNode = useCallback((path: string) => {
    setExpandedNodes((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(path)) {
        newSet.delete(path);
      } else {
        newSet.add(path);
      }
      return newSet;
    });
  }, []);

  const handleExpandAll = () => {
    if (!tree) return;
    const allDirPaths = new Set<string>();
    const traverse = (node: SourceNode) => {
      if (node.type === "directory") {
        allDirPaths.add(node.path);
        node.children?.forEach(traverse);
      }
    };
    tree.children?.forEach(traverse);
    setExpandedNodes(allDirPaths);
  };

  const handleCollapseAll = () => {
    setExpandedNodes(new Set());
  };

  useEffect(() => {
    if (tree) {
      const initialExpanded = new Set<string>();
      tree.children?.forEach((node) => {
        if (node.type === "directory") initialExpanded.add(node.path);
      });
      setExpandedNodes(initialExpanded);
    }
  }, [tree]);

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (isResizing && containerRef.current) {
        const newWidth =
          e.clientX - containerRef.current.getBoundingClientRect().left;
        if (
          newWidth > 150 &&
          newWidth < containerRef.current.clientWidth * 0.7
        ) {
          setTreeWidth(newWidth);
        }
      }
    },
    [isResizing],
  );

  useEffect(() => {
    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    }
    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  const toggleCollapse = () => {
    setIsCollapsed(!isCollapsed);
  };

  const langExtension = activeFile
    ? getLanguageExtension(activeFile.path)
    : undefined;

  const customBgTheme = EditorView.theme({
    "&": {
      backgroundColor: muiTheme.palette.background.paper,
    },
    ".cm-gutters": {
      backgroundColor: muiTheme.palette.background.paper,
    },
  });

  const extensions: ReactCodeMirrorProps["extensions"] = [
    EditorView.lineWrapping,
    readOnlyTheme,
    customBgTheme,
  ];
  if (langExtension) {
    extensions.push(langExtension);
  }
  if (showLineNumbers) {
    extensions.push(lineNumbers());
  }

  return (
    <Box
      ref={containerRef}
      sx={{
        display: "flex",
        height: "100%",
        overflow: "hidden",
        bgcolor: "background.paper",
      }}
    >
      {!isCollapsed && (
        <Box
          sx={{
            width: treeWidth,
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
            flexShrink: 0,
            transition: (theme) =>
              theme.transitions.create("width", {
                easing: theme.transitions.easing.sharp,
                duration: theme.transitions.duration.leavingScreen,
              }),
          }}
        >
          <Box
            sx={{
              p: 1,
              borderBottom: 1,
              borderColor: "divider",
              display: "flex",
              alignItems: "center",
              justifyContent: "flex-end",
              flexShrink: 0,
            }}
          >
            <Box sx={{ display: "flex", alignItems: "center" }}>
              <Tooltip title="Expand All">
                <IconButton onClick={handleExpandAll} size="small">
                  <UnfoldMoreIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title="Collapse All">
                <IconButton onClick={handleCollapseAll} size="small">
                  <UnfoldLessIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <IconButton onClick={toggleCollapse} size="small">
                <ChevronLeft />
              </IconButton>
            </Box>
          </Box>
          <Box
            sx={{
              overflowY: "auto",
              flexGrow: 1,
              borderRight: 1,
              borderColor: "divider",
            }}
          >
            {tree ? (
              <List dense>
                {tree.children?.map((node) => (
                  <TreeNode
                    key={node.path}
                    node={node}
                    onFileSelect={onFileSelect}
                    level={0}
                    expandedNodes={expandedNodes}
                    onToggleNode={handleToggleNode}
                  />
                ))}
              </List>
            ) : (
              <Box
                sx={{
                  display: "flex",
                  justifyContent: "center",
                  alignItems: "center",
                  height: "100%",
                }}
              >
                <CircularProgress />
              </Box>
            )}
          </Box>
        </Box>
      )}

      {!isCollapsed && (
        <Box
          onMouseDown={handleMouseDown}
          sx={{
            width: "5px",
            cursor: "col-resize",
            backgroundColor: "transparent",
            flexShrink: 0,
            "&:hover": {
              backgroundColor: "divider",
            },
            transition: "background-color 0.2s",
          }}
        />
      )}

      <Box
        sx={{
          flexGrow: 1,
          position: "relative",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <Box
          sx={{
            p: 1,
            borderBottom: 1,
            borderColor: "divider",
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            flexShrink: 0,
          }}
        >
          <Box
            sx={{
              display: "flex",
              alignItems: "center",
              flexGrow: 1,
              minWidth: 0,
            }}
          >
            {isCollapsed && (
              <IconButton onClick={toggleCollapse} size="small" sx={{ mr: 1 }}>
                <ChevronRight />
              </IconButton>
            )}
            {activeFile && (
              <Typography variant="caption" noWrap>
                {activeFile.path}
              </Typography>
            )}
          </Box>
          {activeFile && <CopyButton content={activeFile.content} />}
        </Box>
        <Box sx={{ overflow: "auto", flexGrow: 1 }}>
          {activeFile ? (
            <CodeMirror
              value={activeFile.content}
              height="100%"
              theme={codeTheme === "dark" ? oneDark : githubLight}
              extensions={extensions}
              readOnly={true}
              editable={false}
              basicSetup={false}
            />
          ) : (
            <Box
              sx={{
                display: "flex",
                justifyContent: "center",
                alignItems: "center",
                height: "100%",
              }}
            >
              <Typography variant="body2" color="text.secondary">
                Select a file to view its content
              </Typography>
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  );
}
