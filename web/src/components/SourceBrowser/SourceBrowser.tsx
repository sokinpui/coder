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
} from '@mui/material'
import {
  Folder as FolderIcon,
  Description as FileIcon,
  ExpandLess,
  ExpandMore,
  ChevronLeft,
  ChevronRight,
} from '@mui/icons-material'
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter'
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { useContext, useState, useRef, useCallback, useEffect } from 'react'
import type { SourceNode } from '../../types'
import { AppContext } from '../../AppContext'
import { CopyButton } from '../CopyButton'

interface SourceBrowserProps {
  tree: SourceNode | null
  activeFile: { path: string; content: string } | null
  onFileSelect: (path: string) => void
}

interface TreeNodeProps {
  node: SourceNode
  onFileSelect: (path: string) => void
  level: number
}

function TreeNode({ node, onFileSelect, level }: TreeNodeProps) {
  const [open, setOpen] = useState(level === 0)

  const handleClick = () => {
    if (node.type === 'directory') {
      setOpen(!open)
    } else {
      onFileSelect(node.path)
    }
  }

  return (
    <>
      <ListItemButton onClick={handleClick} sx={{ pl: 2 + level * 2 }}>
        <ListItemIcon sx={{ minWidth: 32 }}>
          {node.type === 'directory' ? <FolderIcon fontSize="small" /> : <FileIcon fontSize="small" />}
        </ListItemIcon>
        <ListItemText primary={node.name} primaryTypographyProps={{ variant: 'body2', noWrap: true }} />
        {node.type === 'directory' && (open ? <ExpandLess /> : <ExpandMore />)}
      </ListItemButton>
      {node.type === 'directory' && (
        <Collapse in={open} timeout="auto" unmountOnExit>
          <List component="div" disablePadding>
            {node.children?.map((child) => (
              <TreeNode key={child.path} node={child} onFileSelect={onFileSelect} level={level + 1} />
            ))}
          </List>
        </Collapse>
      )}
    </>
  )
}

export function SourceBrowser({ tree, activeFile, onFileSelect }: SourceBrowserProps) {
  const { codeTheme } = useContext(AppContext)
  const syntaxTheme = codeTheme === 'dark' ? oneDark : oneLight
  const [treeWidth, setTreeWidth] = useState(300)
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [isResizing, setIsResizing] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    setIsResizing(true)
  }, [])

  const handleMouseUp = useCallback(() => {
    setIsResizing(false)
  }, [])

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      if (isResizing && containerRef.current) {
        const newWidth = e.clientX - containerRef.current.getBoundingClientRect().left
        if (newWidth > 150 && newWidth < containerRef.current.clientWidth * 0.7) {
          setTreeWidth(newWidth)
        }
      }
    },
    [isResizing],
  )

  useEffect(() => {
    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
    }
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isResizing, handleMouseMove, handleMouseUp])

  const toggleCollapse = () => {
    setIsCollapsed(!isCollapsed)
  }

  return (
    <Box ref={containerRef} sx={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      {!isCollapsed && (
        <Box
          sx={{
            width: treeWidth,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
            flexShrink: 0,
            transition: (theme) =>
              theme.transitions.create('width', {
                easing: theme.transitions.easing.sharp,
                duration: theme.transitions.duration.leavingScreen,
              }),
          }}
        >
          <Box sx={{ p: 1, borderBottom: 1, borderColor: 'divider', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
            <Typography variant="caption">Project Files</Typography>
            <IconButton onClick={toggleCollapse} size="small">
              <ChevronLeft />
            </IconButton>
          </Box>
          <Box sx={{ overflowY: 'auto', flexGrow: 1, borderRight: 1, borderColor: 'divider' }}>
            {tree ? (
              <List dense>
                {tree.children?.map((node) => (
                  <TreeNode key={node.path} node={node} onFileSelect={onFileSelect} level={0} />
                ))}
              </List>
            ) : (
              <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
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
            width: '5px',
            cursor: 'col-resize',
            backgroundColor: 'transparent',
            flexShrink: 0,
            '&:hover': {
              backgroundColor: 'divider',
            },
            transition: 'background-color 0.2s',
          }}
        />
      )}

      <Box sx={{ flexGrow: 1, position: 'relative', display: 'flex', flexDirection: 'column' }}>
        <Box
          sx={{
            p: 1,
            borderBottom: 1,
            borderColor: 'divider',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            flexShrink: 0,
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', flexGrow: 1, minWidth: 0 }}>
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
        <Box sx={{ overflow: 'auto', flexGrow: 1 }}>
          {activeFile ? (
            <SyntaxHighlighter
              style={syntaxTheme}
              language={activeFile.path.split('.').pop() || 'text'}
              customStyle={{
                margin: 0,
                height: '100%',
                padding: '8px',
                whiteSpace: 'pre-wrap',
                fontSize: '0.8rem',
                overflowWrap: 'break-word',
              }}
              showLineNumbers
            >
              {activeFile.content}
            </SyntaxHighlighter>
          ) : (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
              <Typography variant="body2" color="text.secondary">
                Select a file to view its content
              </Typography>
            </Box>
          )}
        </Box>
      </Box>
    </Box>
  )
}
