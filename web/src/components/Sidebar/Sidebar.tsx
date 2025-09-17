import {
  Box,
  Divider,
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
  useTheme,
} from '@mui/material'
import { AddComment as AddCommentIcon, Code as CodeIcon, History as HistoryIcon } from '@mui/icons-material'
import { drawerWidth, getCollapsedDrawerWidth } from './constants'

interface SidebarProps {
  open: boolean
  onNewChat: () => void
  isGenerating: boolean
  onHistoryOpen: () => void
}

export function Sidebar({ open, onNewChat, isGenerating, onHistoryOpen }: SidebarProps) {
  const theme = useTheme()
  const collapsedDrawerWidth = getCollapsedDrawerWidth(theme)
  const currentDrawerWidth = open ? drawerWidth : collapsedDrawerWidth

  return (
    <Drawer
      variant="permanent"
      anchor="left"
      sx={{
        width: currentDrawerWidth,
        flexShrink: 0,
        whiteSpace: 'nowrap',
        boxSizing: 'border-box',
        '& .MuiDrawer-paper': {
          width: currentDrawerWidth,
          transition: theme.transitions.create('width', {
            easing: theme.transitions.easing.sharp,
            duration: open ? theme.transitions.duration.enteringScreen : theme.transitions.duration.leavingScreen,
          }),
          overflowX: 'hidden',
          boxSizing: 'border-box',
        },
      }}
    >
      <Toolbar variant="dense" sx={{ justifyContent: open ? 'initial' : 'center' }}>
        {open ? (
          <Typography variant="h6" noWrap component="div">
            Coder
          </Typography>
        ) : (
          <CodeIcon />
        )}
      </Toolbar>
      <Divider />
      <Box>
        <List>
          <ListItem disablePadding sx={{ display: 'block' }}>
            <ListItemButton
              onClick={onNewChat}
              disabled={isGenerating}
              sx={{
                minHeight: 48,
                justifyContent: open ? 'initial' : 'center',
                px: 2.5,
                mx: 1,
                width: 'auto',
                borderRadius: (theme) => theme.shape.borderRadius,
              }}
            >
              <ListItemIcon
                sx={{
                  minWidth: 0,
                  mr: open ? 3 : 'auto',
                  justifyContent: 'center',
                }}
              >
                <AddCommentIcon />
              </ListItemIcon>
              <ListItemText primary="New Chat" sx={{ opacity: open ? 1 : 0 }} />
            </ListItemButton>
          </ListItem>
        </List>
        <List>
          <ListItem disablePadding sx={{ display: 'block' }}>
            <ListItemButton
              onClick={onHistoryOpen}
              disabled={isGenerating}
              sx={{
                minHeight: 48,
                justifyContent: open ? 'initial' : 'center',
                px: 2.5,
                mx: 1,
                width: 'auto',
                borderRadius: (theme) => theme.shape.borderRadius,
              }}
            >
              <ListItemIcon
                sx={{
                  minWidth: 0,
                  mr: open ? 3 : 'auto',
                  justifyContent: 'center',
                }}
              >
                <HistoryIcon />
              </ListItemIcon>
              <ListItemText primary="History" sx={{ opacity: open ? 1 : 0 }} />
            </ListItemButton>
          </ListItem>
        </List>
        <Divider />
      </Box>
    </Drawer>
  )
}
