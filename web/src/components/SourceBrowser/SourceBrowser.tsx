import {
  Box,
  Typography,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Collapse,
  CircularProgress,
  IconButton,
  TextField,
  Tooltip,
  Link,
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
  useMemo,
} from "react";
import type { SourceNode } from "../../types";
import { AppContext } from "../../AppContext";
import ReactMarkdown from "react-markdown";
import { CodeBlock } from "../CodeBlock";
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

const filterTree = (
  nodes: SourceNode[],
  query: string,
): SourceNode[] => {
  if (!query) {
    return nodes;
  }
  const lowerCaseQuery = query.toLowerCase();

  const filter = (node: SourceNode): SourceNode | null => {
    // If the node name itself matches, include it and all its children.
    if (node.name.toLowerCase().includes(lowerCaseQuery)) {
      return node;
    }

    // If it's a directory, check its children even if the directory name doesn't match.
    if (node.type === "directory") {
      const filteredChildren = node.children
        ?.map(filter)
        .filter((n): n is SourceNode => n !== null);

      // If any children matched, include this directory but only with the matching children.
      if (filteredChildren && filteredChildren.length > 0) {
        return { ...node, children: filteredChildren };
      }
    }
    return null;
  };

  return nodes.map(filter).filter((n): n is SourceNode => n !== null);
};

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
  // Allow default selection highlighting
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
}: SourceBrowserProps) {
  const { codeTheme } = useContext(AppContext);
  const muiTheme = useTheme();
  const [treeWidth, setTreeWidth] = useState(300);
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [isResizing, setIsResizing] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleLinkClick = (
    e: React.MouseEvent<HTMLAnchorElement>,
    href: string | undefined,
  ) => {
    e.preventDefault();
    if (!href || !activeFile) return;

    // Check for external link
    if (/^(https?:)?\/\//.test(href)) {
      window.open(href, "_blank", "noopener,noreferrer");
      return;
    }

    // Resolve internal link relative to the current file's directory
    const pathParts = activeFile.path.split("/");
    pathParts.pop(); // remove filename

    const targetPath = href.startsWith("/")
      ? href.substring(1) // Absolute path from repo root
      : (pathParts.length > 0 ? pathParts.join("/") + "/" : "") + href;

    const targetPathParts = targetPath.split("/");
    const resolvedPathParts: string[] = [];

    for (const part of targetPathParts) {
      if (part === "." || part === "") continue;

      if (part === "..") {
        if (resolvedPathParts.length > 0) resolvedPathParts.pop();
      } else {
        resolvedPathParts.push(part);
      }
    }

    const newPath = resolvedPathParts.join("/");
    onFileSelect(newPath);
  };

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

  const displayedTree = useMemo(() => {
    if (!tree) return [];
    return filterTree(tree.children || [], searchQuery);
  }, [tree, searchQuery]);

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

  const isMarkdown = activeFile?.path.toLowerCase().endsWith(".md");

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
              display: "flex",
              alignItems: "center",
              justifyContent: "space-between",
              flexShrink: 0,
              borderBottom: 1,
              borderColor: 'divider',
            }}
          >
            <TextField
              variant="standard"
              size="small"
              placeholder="Search..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              sx={{ flexGrow: 1, mr: 1 }}
            />
            <Box sx={{ display: "flex", alignItems: "center" }}>
              <Tooltip title="Expand All" enterDelay={1000}>
                <IconButton onClick={handleExpandAll} size="small">
                  <UnfoldMoreIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title="Collapse All" enterDelay={1000}>
                <IconButton onClick={handleCollapseAll} size="small">
                  <UnfoldLessIcon fontSize="small" />
                </IconButton>
              </Tooltip>
              <Tooltip title="Collapse panel" enterDelay={1000}>
                <IconButton onClick={toggleCollapse} size="small">
                  <ChevronLeft />
                </IconButton>
              </Tooltip>
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
                {displayedTree.map((node) => (
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
              <Tooltip title="Expand panel" enterDelay={1000}>
                <IconButton onClick={toggleCollapse} size="small" sx={{ mr: 1 }}>
                  <ChevronRight />
                </IconButton>
              </Tooltip>
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
            isMarkdown ? (
              <Box
                sx={{
                  p: 2,
                  "& pre": {
                    whiteSpace: "pre-wrap",
                    wordWrap: "break-word",
                    fontFamily: "monospace",
                  },
                  "& code": {
                    fontFamily: "monospace",
                    backgroundColor: "action.hover",
                    px: 0.5,
                    borderRadius: (theme) => theme.shape.borderRadius / 2,
                  },
                  "& pre > code": {
                    display: "block",
                    p: 1,
                    backgroundColor: "action.selected",
                    borderRadius: (theme) => theme.shape.borderRadius / 2,
                  },
                }}
              >
                <ReactMarkdown
                  components={{
                    code({ className, children, ...props }) {
                      const match = /language-(\w+)/.exec(className || "");
                      if (match) {
                        return (
                          <CodeBlock language={match[1]}>{children}</CodeBlock>
                        );
                      }
                      return (
                        <code className={className} {...props}>
                          {children}
                        </code>
                      );
                    },
                    a: ({ ...props }) => {
                      return (
                        <Link
                          href={props.href}
                          onClick={(e) =>
                            handleLinkClick(
                              e as React.MouseEvent<HTMLAnchorElement>,
                              props.href,
                            )
                          }
                          sx={{ cursor: "pointer" }}
                        >
                          {props.children}
                        </Link>
                      );
                    },
                  }}
                >
                  {activeFile.content}
                </ReactMarkdown>
              </Box>
            ) : (
              <CodeMirror
                value={activeFile.content}
                height="100%"
                theme={codeTheme === "dark" ? oneDark : githubLight}
                extensions={extensions}
                readOnly={true}
                editable={false}
                basicSetup={false}
              />
            )
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
