import { useState, useContext } from 'react'
import {
  Box,
  Typography,
  useTheme,
  AppBar,
  Toolbar,
  CssBaseline,
  IconButton,
	FormControl,
	Select,
	MenuItem,
	Divider,
	type SelectChangeEvent,
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
  const {
		messages,
		sendMessage,
		cwd,
		isGenerating,
		tokenCount,
		cancelGeneration,
		mode,
		model,
		availableModes,
		availableModels,
	} = useWebSocket(`ws://${location.host}/ws`)
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

	const handleModeChange = (event: SelectChangeEvent) => {
		sendMessage(`:mode ${event.target.value}`)
	}

	const handleModelChange = (event: SelectChangeEvent) => {
		sendMessage(`:model ${event.target.value}`)
  }

  const currentDrawerWidth = sidebarOpen ? drawerWidth : collapsedDrawerWidth

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <CssBaseline />
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
        <AppBar position="static" elevation={1}>
          <Toolbar variant="dense">
            <IconButton
              color="inherit"
              aria-label="open drawer"
              onClick={handleSidebarToggle}
              edge="start"
            >
              <MenuIcon />
            </IconButton>
					<Box sx={{ flexGrow: 1 }} />
					<Typography variant="body2" sx={{ color: 'inherit' }}>
						{`Tokens: ${tokenCount}`}
					</Typography>

					<Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)' }} />

					<Typography variant="body2" sx={{ color: 'inherit' }}>
						{cwd}
					</Typography>

					<Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)' }} />

					<FormControl size="small" variant="standard" sx={{ minWidth: 120 }} disabled={isGenerating}>
						<Select
							value={mode}
							onChange={handleModeChange}
							disableUnderline
							sx={{ color: 'inherit', '& .MuiSelect-icon': { color: 'inherit' } }}
						>
							{availableModes.map((m) => (
								<MenuItem key={m} value={m}>{m}</MenuItem>
							))}
						</Select>
					</FormControl>

					<Divider orientation="vertical" flexItem sx={{ mx: 1.5, my: 1, borderColor: 'rgba(255, 255, 255, 0.2)' }} />

					<FormControl size="small" variant="standard" sx={{ minWidth: 200 }} disabled={isGenerating}>
						<Select
							value={model}
							onChange={handleModelChange}
							disableUnderline
							sx={{ color: 'inherit', '& .MuiSelect-icon': { color: 'inherit' } }}
						>
							{availableModels.map((m) => (
								<MenuItem key={m} value={m}>{m}</MenuItem>
							))}
						</Select>
					</FormControl>
          <IconButton sx={{ ml: 1 }} onClick={toggleColorMode} color="inherit">
            {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
          </IconButton>
          </Toolbar>
        </AppBar>
        <MessageList messages={messages} isGenerating={isGenerating} />
        <ChatInput sendMessage={sendMessage} cancelGeneration={cancelGeneration} isGenerating={isGenerating} />
      </Box>
    </Box>
  )
}

export default App;
