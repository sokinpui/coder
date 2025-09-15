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
  useTheme,
} from '@mui/material'
import { AddComment as AddCommentIcon } from '@mui/icons-material'
import { drawerWidth, getCollapsedDrawerWidth } from './constants'

interface SidebarProps {
  open: boolean
  onNewChat: () => void
}

export function Sidebar({ open, onNewChat }: SidebarProps) {
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
      <Toolbar variant="dense" />
      <Box>
        <List>
          <ListItem disablePadding sx={{ display: 'block' }}>
            <ListItemButton
              onClick={onNewChat}
              sx={{
                minHeight: 48,
                justifyContent: open ? 'initial' : 'center',
                px: 2.5,
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
        <Divider />
      </Box>
    </Drawer>
  )
}
