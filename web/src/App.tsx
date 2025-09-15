import { useState, useContext } from 'react'
import {
  Box,
  Typography,
  useTheme,
  AppBar,
  Toolbar,
  CssBaseline,
  IconButton,
} from '@mui/material'
import {
  Brightness4 as Brightness4Icon,
  Brightness7 as Brightness7Icon,
  Menu as MenuIcon,
} from '@mui/icons-material'
import { AppContext } from './AppContext'
import { useWebSocket } from './hooks/useWebSocket'
import { Sidebar, drawerWidth, getCollapsedDrawerWidth } from './components/Sidebar'
import { MessageList } from './components/MessageList'
import { ChatInput } from './components/ChatInput'

function App() {
  const { messages, sendMessage, cwd, isGenerating, cancelGeneration } = useWebSocket(`ws://${location.host}/ws`)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const theme = useTheme()
  const { toggleColorMode } = useContext(AppContext)
  const collapsedDrawerWidth = getCollapsedDrawerWidth(theme)

  const handleSidebarToggle = () => {
    setSidebarOpen(!sidebarOpen)
  }

  const handleNewChat = () => {
    sendMessage(':new')
  }

  const currentDrawerWidth = sidebarOpen ? drawerWidth : collapsedDrawerWidth

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <CssBaseline />
      <AppBar
        position="fixed"
        elevation={1}
        sx={{
          zIndex: theme.zIndex.drawer + 1,
          width: `calc(100% - ${currentDrawerWidth}px)`,
          marginLeft: `${currentDrawerWidth}px`,
          transition: theme.transitions.create(['width', 'margin'], {
            easing: theme.transitions.easing.sharp,
            duration: sidebarOpen ? theme.transitions.duration.enteringScreen : theme.transitions.duration.leavingScreen,
          }),
        }}
      >
        <Toolbar variant="dense">
          <IconButton
            color="inherit"
            aria-label="open drawer"
            onClick={handleSidebarToggle}
            edge="start"
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
            Coder
          </Typography>
          <Typography variant="caption" sx={{ mr: 2, color: 'text.secondary' }}>
            {cwd}
          </Typography>
          <IconButton sx={{ ml: 1 }} onClick={toggleColorMode} color="inherit">
            {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
          </IconButton>
        </Toolbar>
      </AppBar>
      <Sidebar open={sidebarOpen} onNewChat={handleNewChat} isGenerating={isGenerating} />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          display: 'flex',
          flexDirection: 'column',
          height: '100vh',
          bgcolor: 'background.default',
          color: 'text.primary',
        }}
      >
        <Toolbar variant="dense" />
        <MessageList messages={messages} isGenerating={isGenerating} />
        <ChatInput sendMessage={sendMessage} cancelGeneration={cancelGeneration} isGenerating={isGenerating} />
      </Box>
    </Box>
  )
}

export default App;
